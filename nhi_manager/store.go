// Nexus IA v6.0 — Persistencia del NHI Manager en PostgreSQL
//
// Guarda y recupera:
//   - Identidades (NHI) activas
//   - Registros de revocación
//   - Audit trail completo (inmutable)
//   - Claves públicas Dilithium
//
// Usa la misma conexión PostgreSQL que db/. Añade su propio schema.
package nhi_manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store es el repositorio de persistencia del NHI Manager.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye un Store desde un pool de PostgreSQL existente.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del NHI Manager.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS nhi_identities (
    agent_id        TEXT PRIMARY KEY,
    agent_type      TEXT NOT NULL,
    did             TEXT NOT NULL UNIQUE,
    cert_x509_pem   BYTEA NOT NULL,
    public_key      BYTEA NOT NULL,
    allowed_actions TEXT[] NOT NULL DEFAULT '{}',
    allowed_assets  TEXT[] NOT NULL DEFAULT '{}',
    issued_at       TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    revoke_reason   TEXT
);

CREATE INDEX IF NOT EXISTS idx_nhi_did ON nhi_identities(did);
CREATE INDEX IF NOT EXISTS idx_nhi_expires ON nhi_identities(expires_at);
CREATE INDEX IF NOT EXISTS idx_nhi_revoked ON nhi_identities(revoked_at) WHERE revoked_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS nhi_revocations (
    agent_id    TEXT PRIMARY KEY REFERENCES nhi_identities(agent_id) ON DELETE CASCADE,
    did         TEXT NOT NULL,
    reason      TEXT NOT NULL,
    message     TEXT,
    revoked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_by  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS nhi_audit (
    id                  TEXT PRIMARY KEY,
    actor_id            TEXT NOT NULL,
    actor_did           TEXT NOT NULL,
    agent_type          TEXT NOT NULL,
    action              TEXT NOT NULL,
    target              TEXT NOT NULL,
    payload_hash        TEXT NOT NULL,
    result              TEXT NOT NULL,
    reason              TEXT,
    session_id          TEXT,
    trace_id            TEXT,
    remote_addr         TEXT,
    timestamp           TIMESTAMPTZ NOT NULL,
    metadata            JSONB,
    sig_alg             TEXT NOT NULL,
    signature           TEXT NOT NULL,
    signing_key_id      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_actor ON nhi_audit(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON nhi_audit(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON nhi_audit(action);

CREATE TABLE IF NOT EXISTS nhi_signing_keys (
    key_id      TEXT PRIMARY KEY,
    public_key  BYTEA NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    active      BOOLEAN NOT NULL DEFAULT TRUE
);
`

// Migrate crea las tablas del NHI Manager si no existen.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate nhi schema: %w", err)
	}
	return nil
}

// SaveIdentity persiste una identidad NHI.
// Hace UPSERT por agent_id.
func (s *Store) SaveIdentity(ctx context.Context, nhi *NHI) error {
	actions := make([]string, len(nhi.AllowedActions))
	for i, a := range nhi.AllowedActions {
		actions[i] = string(a)
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO nhi_identities
			(agent_id, agent_type, did, cert_x509_pem, public_key,
			 allowed_actions, allowed_assets, issued_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (agent_id) DO UPDATE SET
			agent_type = EXCLUDED.agent_type,
			did = EXCLUDED.did,
			cert_x509_pem = EXCLUDED.cert_x509_pem,
			public_key = EXCLUDED.public_key,
			allowed_actions = EXCLUDED.allowed_actions,
			allowed_assets = EXCLUDED.allowed_assets,
			issued_at = EXCLUDED.issued_at,
			expires_at = EXCLUDED.expires_at
	`, nhi.AgentID, string(nhi.AgentType), nhi.DID.String(),
		nhi.CertX509PEM, nhi.PublicKey,
		actions, nhi.AllowedAssets,
		nhi.IssuedAt, nhi.ExpiresAt)

	if err != nil {
		return fmt.Errorf("save nhi: %w", err)
	}
	return nil
}

// GetIdentity recupera una identidad por AgentID.
// Devuelve nil, nil si no existe.
func (s *Store) GetIdentity(ctx context.Context, agentID string) (*NHI, error) {
	var (
		nhi             NHI
		agentTypeStr    string
		didStr          string
		actions         []string
		revokedAt       *time.Time
		revokeReason    *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT agent_id, agent_type, did, cert_x509_pem, public_key,
		       allowed_actions, allowed_assets, issued_at, expires_at,
		       revoked_at, revoke_reason
		FROM nhi_identities
		WHERE agent_id = $1
	`, agentID).Scan(
		&nhi.AgentID, &agentTypeStr, &didStr, &nhi.CertX509PEM, &nhi.PublicKey,
		&actions, &nhi.AllowedAssets, &nhi.IssuedAt, &nhi.ExpiresAt,
		&revokedAt, &revokeReason)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get nhi: %w", err)
	}

	nhi.AgentType = AgentType(agentTypeStr)
	nhi.DID = DID(didStr)
	nhi.AllowedActions = make([]ActionType, len(actions))
	for i, a := range actions {
		nhi.AllowedActions[i] = ActionType(a)
	}
	nhi.RevokedAt = revokedAt
	if revokeReason != nil {
		nhi.RevokeReason = *revokeReason
	}

	return &nhi, nil
}

// SaveAudit persiste una entrada del audit trail.
// Las entradas son INMUTABLES: no se pueden modificar.
func (s *Store) SaveAudit(ctx context.Context, entry *AuditEntry) error {
	var metadataJSON []byte
	if entry.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO nhi_audit
			(id, actor_id, actor_did, agent_type, action, target,
			 payload_hash, result, reason, session_id, trace_id, remote_addr,
			 timestamp, metadata, sig_alg, signature, signing_key_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, entry.ID, entry.ActorID, entry.ActorDID, string(entry.AgentType),
		entry.Action, entry.Target, entry.PayloadHash, entry.Result, entry.Reason,
		entry.SessionID, entry.TraceID, entry.RemoteAddr,
		entry.Timestamp, metadataJSON,
		entry.SignatureAlgorithm, entry.Signature, entry.SigningKeyID)

	if err != nil {
		return fmt.Errorf("save audit: %w", err)
	}
	return nil
}

// ListAudit devuelve las entradas de audit de un agente.
// Limit controla el número máximo de entradas devueltas (0 = sin límite).
func (s *Store) ListAudit(ctx context.Context, agentID string, limit int) ([]*AuditEntry, error) {
	query := `
		SELECT id, actor_id, actor_did, agent_type, action, target,
		       payload_hash, result, reason, session_id, trace_id, remote_addr,
		       timestamp, metadata, sig_alg, signature, signing_key_id
		FROM nhi_audit
		WHERE actor_id = $1
		ORDER BY timestamp DESC
	`
	if limit > 0 {
		query += " LIMIT $2"
	}

	var rows pgx.Rows
	var err error
	if limit > 0 {
		rows, err = s.pool.Query(ctx, query, agentID, limit)
	} else {
		rows, err = s.pool.Query(ctx, query, agentID)
	}
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		var (
			entry       AuditEntry
			agentTypeSt string
			metadataRaw []byte
		)
		err := rows.Scan(
			&entry.ID, &entry.ActorID, &entry.ActorDID, &agentTypeSt,
			&entry.Action, &entry.Target, &entry.PayloadHash, &entry.Result, &entry.Reason,
			&entry.SessionID, &entry.TraceID, &entry.RemoteAddr,
			&entry.Timestamp, &metadataRaw,
			&entry.SignatureAlgorithm, &entry.Signature, &entry.SigningKeyID)
		if err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}

		entry.AgentType = AgentType(agentTypeSt)
		if metadataRaw != nil {
			_ = json.Unmarshal(metadataRaw, &entry.Metadata)
		}
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// SaveSigningKey guarda una clave pública de firma.
func (s *Store) SaveSigningKey(ctx context.Context, keyID string, publicKey []byte) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO nhi_signing_keys (key_id, public_key, active)
		VALUES ($1, $2, TRUE)
		ON CONFLICT (key_id) DO UPDATE SET
			public_key = EXCLUDED.public_key,
			active = TRUE
	`, keyID, publicKey)
	if err != nil {
		return fmt.Errorf("save signing key: %w", err)
	}
	return nil
}

// GetSigningKey recupera la clave pública activa.
// Devuelve nil, nil si no existe.
func (s *Store) GetSigningKey(ctx context.Context, keyID string) ([]byte, error) {
	var pubKey []byte
	err := s.pool.QueryRow(ctx, `
		SELECT public_key FROM nhi_signing_keys
		WHERE key_id = $1 AND active = TRUE
	`, keyID).Scan(&pubKey)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get signing key: %w", err)
	}
	return pubKey, nil
}
