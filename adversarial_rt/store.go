// Nexus IA v6.0 — Persistencia del Adversarial RT en PostgreSQL
//
// Guarda:
//   - Ciclos de testing adversarial
//   - Ataques individuales
//   - Debilidades encontradas
//   - Patches aplicados
package adversarial_rt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persiste el estado del Adversarial RT.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye el store desde un pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del Adversarial RT.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS adv_cycles (
    id              TEXT PRIMARY KEY,
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    weaknesses      INT NOT NULL DEFAULT 0,
    patches         INT NOT NULL DEFAULT 0,
    summary         TEXT
);

CREATE INDEX IF NOT EXISTS idx_adv_cycles_started ON adv_cycles(started_at DESC);

CREATE TABLE IF NOT EXISTS adv_attacks (
    id              TEXT PRIMARY KEY,
    cycle_id        TEXT REFERENCES adv_cycles(id) ON DELETE CASCADE,
    target          TEXT NOT NULL,
    type            TEXT NOT NULL,
    technique       TEXT,
    payload         TEXT,
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    attempts        INT NOT NULL DEFAULT 0,
    result          TEXT NOT NULL,
    severity        TEXT NOT NULL,
    evidence        TEXT,
    details         TEXT,
    patch_suggested TEXT,
    patch_applied   BOOLEAN NOT NULL DEFAULT FALSE,
    patch_applied_at TIMESTAMPTZ,
    metadata        JSONB
);

CREATE INDEX IF NOT EXISTS idx_adv_attacks_target ON adv_attacks(target);
CREATE INDEX IF NOT EXISTS idx_adv_attacks_type ON adv_attacks(type);
CREATE INDEX IF NOT EXISTS idx_adv_attacks_result ON adv_attacks(result);
CREATE INDEX IF NOT EXISTS idx_adv_attacks_severity ON adv_attacks(severity);
CREATE INDEX IF NOT EXISTS idx_adv_attacks_started ON adv_attacks(started_at DESC);
`

// Migrate crea el schema del Adversarial RT.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate adv schema: %w", err)
	}
	return nil
}

// SaveCycle persiste un ciclo.
func (s *Store) SaveCycle(ctx context.Context, cycle *Cycle) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO adv_cycles
			(id, started_at, finished_at, weaknesses, patches, summary)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			finished_at = EXCLUDED.finished_at,
			weaknesses = EXCLUDED.weaknesses,
			patches = EXCLUDED.patches,
			summary = EXCLUDED.summary
	`, cycle.ID, cycle.StartedAt, cycle.FinishedAt, cycle.Weaknesses, cycle.Patches, cycle.Summary)

	if err != nil {
		return fmt.Errorf("save cycle: %w", err)
	}
	return nil
}

// SaveAttack persiste un ataque.
func (s *Store) SaveAttack(ctx context.Context, cycleID string, attack *Attack) error {
	var metadataJSON []byte
	if attack.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(attack.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO adv_attacks
			(id, cycle_id, target, type, technique, payload,
			 started_at, finished_at, attempts, result, severity,
			 evidence, details, patch_suggested, patch_applied, patch_applied_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			finished_at = EXCLUDED.finished_at,
			attempts = EXCLUDED.attempts,
			result = EXCLUDED.result,
			severity = EXCLUDED.severity,
			evidence = EXCLUDED.evidence,
			details = EXCLUDED.details,
			patch_suggested = EXCLUDED.patch_suggested,
			patch_applied = EXCLUDED.patch_applied,
			patch_applied_at = EXCLUDED.patch_applied_at
	`, attack.ID, cycleID, string(attack.Target), string(attack.Type),
		attack.Technique, attack.Payload, attack.StartedAt, attack.FinishedAt,
		attack.Attempts, string(attack.Result), string(attack.Severity),
		attack.Evidence, attack.Details, attack.PatchSuggested,
		attack.PatchApplied, attack.PatchAppliedAt, metadataJSON)

	if err != nil {
		return fmt.Errorf("save attack: %w", err)
	}
	return nil
}

// ListCycles devuelve los ciclos ordenados por fecha descendente.
func (s *Store) ListCycles(ctx context.Context, limit int) ([]*Cycle, error) {
	query := `
		SELECT id, started_at, finished_at, weaknesses, patches, summary
		FROM adv_cycles
		ORDER BY started_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list cycles: %w", err)
	}
	defer rows.Close()

	var out []*Cycle
	for rows.Next() {
		var (
			c       Cycle
			summary *string
		)
		err := rows.Scan(&c.ID, &c.StartedAt, &c.FinishedAt, &c.Weaknesses, &c.Patches, &summary)
		if err != nil {
			return nil, fmt.Errorf("scan cycle: %w", err)
		}
		if summary != nil {
			c.Summary = *summary
		}
		out = append(out, &c)
	}

	return out, rows.Err()
}

// ListAttacksByTarget devuelve los ataques a un target.
func (s *Store) ListAttacksByTarget(ctx context.Context, target Target, limit int) ([]*Attack, error) {
	query := `
		SELECT id, target, type, technique, payload, started_at, finished_at,
		       attempts, result, severity, evidence, details,
		       patch_suggested, patch_applied, patch_applied_at, metadata
		FROM adv_attacks
		WHERE target = $1
		ORDER BY started_at DESC
	`
	args := []interface{}{string(target)}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list attacks: %w", err)
	}
	defer rows.Close()

	var out []*Attack
	for rows.Next() {
		var (
			a           Attack
			targetStr   string
			typeStr     string
			resultStr   string
			sevStr      string
			technique   *string
			payload     *string
			evidence    *string
			details     *string
			patchSug    *string
			metadataRaw []byte
		)
		err := rows.Scan(
			&a.ID, &targetStr, &typeStr, &technique, &payload,
			&a.StartedAt, &a.FinishedAt, &a.Attempts,
			&resultStr, &sevStr, &evidence, &details,
			&patchSug, &a.PatchApplied, &a.PatchAppliedAt, &metadataRaw)
		if err != nil {
			return nil, fmt.Errorf("scan attack: %w", err)
		}
		a.Target = Target(targetStr)
		a.Type = AttackType(typeStr)
		a.Result = Result(resultStr)
		a.Severity = Severity(sevStr)
		if technique != nil {
			a.Technique = *technique
		}
		if payload != nil {
			a.Payload = *payload
		}
		if evidence != nil {
			a.Evidence = *evidence
		}
		if details != nil {
			a.Details = *details
		}
		if patchSug != nil {
			a.PatchSuggested = *patchSug
		}
		if metadataRaw != nil {
			_ = json.Unmarshal(metadataRaw, &a.Metadata)
		}
		out = append(out, &a)
	}

	return out, rows.Err()
}

// ListWeaknesses devuelve los ataques que encontraron debilidad.
func (s *Store) ListWeaknesses(ctx context.Context, limit int) ([]*Attack, error) {
	query := `
		SELECT id, target, type, technique, payload, started_at, finished_at,
		       attempts, result, severity, evidence, details,
		       patch_suggested, patch_applied, patch_applied_at, metadata
		FROM adv_attacks
		WHERE result IN ('WEAKNESS_FOUND', 'PARTIAL_BYPASS')
		ORDER BY severity DESC, started_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list weaknesses: %w", err)
	}
	defer rows.Close()

	var out []*Attack
	for rows.Next() {
		var (
			a           Attack
			targetStr   string
			typeStr     string
			resultStr   string
			sevStr      string
			technique   *string
			payload     *string
			evidence    *string
			details     *string
			patchSug    *string
			metadataRaw []byte
		)
		err := rows.Scan(
			&a.ID, &targetStr, &typeStr, &technique, &payload,
			&a.StartedAt, &a.FinishedAt, &a.Attempts,
			&resultStr, &sevStr, &evidence, &details,
			&patchSug, &a.PatchApplied, &a.PatchAppliedAt, &metadataRaw)
		if err != nil {
			return nil, fmt.Errorf("scan weakness: %w", err)
		}
		a.Target = Target(targetStr)
		a.Type = AttackType(typeStr)
		a.Result = Result(resultStr)
		a.Severity = Severity(sevStr)
		if technique != nil {
			a.Technique = *technique
		}
		if payload != nil {
			a.Payload = *payload
		}
		if evidence != nil {
			a.Evidence = *evidence
		}
		if details != nil {
			a.Details = *details
		}
		if patchSug != nil {
			a.PatchSuggested = *patchSug
		}
		if metadataRaw != nil {
			_ = json.Unmarshal(metadataRaw, &a.Metadata)
		}
		out = append(out, &a)
	}

	return out, rows.Err()
}

// CountWeaknesses cuenta las debilidades encontradas.
func (s *Store) CountWeaknesses(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM adv_attacks
		WHERE result IN ('WEAKNESS_FOUND', 'PARTIAL_BYPASS') AND started_at >= $1
	`, since).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("count weaknesses: %w", err)
	}
	return count, nil
}
