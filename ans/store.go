// Nexus IA v6.0 — Persistencia del ANS en PostgreSQL
//
// Guarda y recupera el directorio de agentes (Entry) con todas
// sus credenciales, permisos y estado.
package ans

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persiste el estado del ANS.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye el store desde un pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del ANS.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS ans_entries (
    name            TEXT PRIMARY KEY,
    agent_id        TEXT NOT NULL UNIQUE,
    agent_type      TEXT NOT NULL,
    did             TEXT UNIQUE,
    cert_x509_pem   BYTEA,
    public_key      BYTEA,
    allowed_actions TEXT[] NOT NULL DEFAULT '{}',
    allowed_assets  TEXT[] NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'ACTIVE',
    last_action     TEXT,
    last_seen       TIMESTAMPTZ,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ans_agent_id ON ans_entries(agent_id);
CREATE INDEX IF NOT EXISTS idx_ans_did ON ans_entries(did);
CREATE INDEX IF NOT EXISTS idx_ans_status ON ans_entries(status);
CREATE INDEX IF NOT EXISTS idx_ans_type ON ans_entries(agent_type);
CREATE INDEX IF NOT EXISTS idx_ans_expires ON ans_entries(expires_at);
`

// Migrate crea el schema del ANS.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate ans schema: %w", err)
	}
	return nil
}

// SaveEntry persiste (UPSERT) una entrada.
func (s *Store) SaveEntry(ctx context.Context, e *Entry) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ans_entries
			(name, agent_id, agent_type, did, cert_x509_pem, public_key,
			 allowed_actions, allowed_assets, status, last_action, last_seen,
			 registered_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (name) DO UPDATE SET
			agent_id = EXCLUDED.agent_id,
			agent_type = EXCLUDED.agent_type,
			did = EXCLUDED.did,
			cert_x509_pem = EXCLUDED.cert_x509_pem,
			public_key = EXCLUDED.public_key,
			allowed_actions = EXCLUDED.allowed_actions,
			allowed_assets = EXCLUDED.allowed_assets,
			status = EXCLUDED.status,
			last_action = EXCLUDED.last_action,
			last_seen = EXCLUDED.last_seen,
			updated_at = NOW(),
			expires_at = EXCLUDED.expires_at
	`, string(e.Name), e.AgentID, e.AgentType, e.DID,
		e.CertX509PEM, e.PublicKey,
		e.AllowedActions, e.AllowedAssets, string(e.Status),
		e.LastAction, e.LastSeen,
		e.RegisteredAt, time.Now(), e.ExpiresAt)

	if err != nil {
		return fmt.Errorf("save ans entry: %w", err)
	}
	return nil
}

// GetEntry recupera una entrada por nombre.
// Devuelve nil, nil si no existe.
func (s *Store) GetEntry(ctx context.Context, name AgentName) (*Entry, error) {
	var (
		e            Entry
		nameStr      string
		statusStr    string
		didStr       *string
		lastAction   *string
		lastSeen     *time.Time
		certX509PEM  []byte
		publicKey    []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT name, agent_id, agent_type, did, cert_x509_pem, public_key,
		       allowed_actions, allowed_assets, status, last_action, last_seen,
		       registered_at, updated_at, expires_at
		FROM ans_entries
		WHERE name = $1
	`, string(name)).Scan(
		&nameStr, &e.AgentID, &e.AgentType, &didStr, &certX509PEM, &publicKey,
		&e.AllowedActions, &e.AllowedAssets, &statusStr, &lastAction, &lastSeen,
		&e.RegisteredAt, &e.UpdatedAt, &e.ExpiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get ans entry: %w", err)
	}

	e.Name = AgentName(nameStr)
	e.Status = AgentStatus(statusStr)
	if didStr != nil {
		e.DID = *didStr
	}
	if lastAction != nil {
		e.LastAction = *lastAction
	}
	if lastSeen != nil {
		e.LastSeen = *lastSeen
	}
	e.CertX509PEM = certX509PEM
	e.PublicKey = publicKey

	return &e, nil
}

// GetEntryByAgentID recupera una entrada por AgentID.
func (s *Store) GetEntryByAgentID(ctx context.Context, agentID string) (*Entry, error) {
	var (
		e            Entry
		nameStr      string
		statusStr    string
		didStr       *string
		lastAction   *string
		lastSeen     *time.Time
		certX509PEM  []byte
		publicKey    []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT name, agent_id, agent_type, did, cert_x509_pem, public_key,
		       allowed_actions, allowed_assets, status, last_action, last_seen,
		       registered_at, updated_at, expires_at
		FROM ans_entries
		WHERE agent_id = $1
	`, agentID).Scan(
		&nameStr, &e.AgentID, &e.AgentType, &didStr, &certX509PEM, &publicKey,
		&e.AllowedActions, &e.AllowedAssets, &statusStr, &lastAction, &lastSeen,
		&e.RegisteredAt, &e.UpdatedAt, &e.ExpiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get ans entry by agent_id: %w", err)
	}

	e.Name = AgentName(nameStr)
	e.Status = AgentStatus(statusStr)
	if didStr != nil {
		e.DID = *didStr
	}
	if lastAction != nil {
		e.LastAction = *lastAction
	}
	if lastSeen != nil {
		e.LastSeen = *lastSeen
	}
	e.CertX509PEM = certX509PEM
	e.PublicKey = publicKey

	return &e, nil
}

// DeleteEntry elimina una entrada del ANS.
func (s *Store) DeleteEntry(ctx context.Context, name AgentName) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM ans_entries WHERE name = $1`, string(name))
	if err != nil {
		return fmt.Errorf("delete ans entry: %w", err)
	}
	return nil
}

// ListAll devuelve todas las entradas.
func (s *Store) ListAll(ctx context.Context) ([]*Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT name, agent_id, agent_type, did, cert_x509_pem, public_key,
		       allowed_actions, allowed_assets, status, last_action, last_seen,
		       registered_at, updated_at, expires_at
		FROM ans_entries
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list ans: %w", err)
	}
	defer rows.Close()

	var out []*Entry
	for rows.Next() {
		var (
			e           Entry
			nameStr     string
			statusStr   string
			didStr      *string
			lastAction  *string
			lastSeen    *time.Time
			certX509PEM []byte
			publicKey   []byte
		)
		err := rows.Scan(
			&nameStr, &e.AgentID, &e.AgentType, &didStr, &certX509PEM, &publicKey,
			&e.AllowedActions, &e.AllowedAssets, &statusStr, &lastAction, &lastSeen,
			&e.RegisteredAt, &e.UpdatedAt, &e.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("scan ans: %w", err)
		}
		e.Name = AgentName(nameStr)
		e.Status = AgentStatus(statusStr)
		if didStr != nil {
			e.DID = *didStr
		}
		if lastAction != nil {
			e.LastAction = *lastAction
		}
		if lastSeen != nil {
			e.LastSeen = *lastSeen
		}
		e.CertX509PEM = certX509PEM
		e.PublicKey = publicKey
		out = append(out, &e)
	}

	return out, rows.Err()
}

// ListByStatus devuelve las entradas con un estado.
func (s *Store) ListByStatus(ctx context.Context, status AgentStatus) ([]*Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT name, agent_id, agent_type, did, cert_x509_pem, public_key,
		       allowed_actions, allowed_assets, status, last_action, last_seen,
		       registered_at, updated_at, expires_at
		FROM ans_entries
		WHERE status = $1
		ORDER BY name
	`, string(status))
	if err != nil {
		return nil, fmt.Errorf("list by status: %w", err)
	}
	defer rows.Close()

	var out []*Entry
	for rows.Next() {
		var (
			e           Entry
			nameStr     string
			statusStr   string
			didStr      *string
			lastAction  *string
			lastSeen    *time.Time
			certX509PEM []byte
			publicKey   []byte
		)
		err := rows.Scan(
			&nameStr, &e.AgentID, &e.AgentType, &didStr, &certX509PEM, &publicKey,
			&e.AllowedActions, &e.AllowedAssets, &statusStr, &lastAction, &lastSeen,
			&e.RegisteredAt, &e.UpdatedAt, &e.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("scan ans: %w", err)
		}
		e.Name = AgentName(nameStr)
		e.Status = AgentStatus(statusStr)
		if didStr != nil {
			e.DID = *didStr
		}
		if lastAction != nil {
			e.LastAction = *lastAction
		}
		if lastSeen != nil {
			e.LastSeen = *lastSeen
		}
		e.CertX509PEM = certX509PEM
		e.PublicKey = publicKey
		out = append(out, &e)
	}

	return out, rows.Err()
}

// DeleteExpired elimina las entradas expiradas.
// Devuelve el número de entradas eliminadas.
func (s *Store) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM ans_entries WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("delete expired ans: %w", err)
	}
	return tag.RowsAffected(), nil
}
