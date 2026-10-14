// Nexus IA v6.0 — Persistencia del Governance en PostgreSQL
//
// Guarda:
//   - Reportes de gobernanza generados
//   - Firmas Dilithium
//   - Rutas de archivos JSON/STIX/PDF
package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persiste el estado del Governance.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye el store desde un pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del Governance.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS gov_reports (
    id                  TEXT PRIMARY KEY,
    period              TEXT NOT NULL,
    period_start        TIMESTAMPTZ NOT NULL,
    period_end          TIMESTAMPTZ NOT NULL,
    generated_at        TIMESTAMPTZ NOT NULL,
    generated_by        TEXT NOT NULL,
    total_actions       INT NOT NULL DEFAULT 0,
    total_blocked       INT NOT NULL DEFAULT 0,
    total_rogue         INT NOT NULL DEFAULT 0,
    overall_risk        TEXT NOT NULL,
    summary             TEXT,
    rsi_level           TEXT,
    content_hash        TEXT NOT NULL,
    signature           TEXT NOT NULL,
    signature_alg       TEXT NOT NULL,
    signing_key_id      TEXT NOT NULL,
    json_path           TEXT,
    pdf_path            TEXT,
    stix_path           TEXT,
    full_report         JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gov_reports_ts ON gov_reports(generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_gov_reports_period ON gov_reports(period);
CREATE INDEX IF NOT EXISTS idx_gov_reports_risk ON gov_reports(overall_risk);
`

// Migrate crea el schema del Governance.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate gov schema: %w", err)
	}
	return nil
}

// SaveReport persiste un reporte.
func (s *Store) SaveReport(ctx context.Context, r *Report) error {
	if r == nil {
		return fmt.Errorf("nil report")
	}

	fullJSON, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	var rsiLevel *string
	if r.RSIEvaluation != nil {
		lvl := r.RSIEvaluation.Level
		rsiLevel = &lvl
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO gov_reports
			(id, period, period_start, period_end, generated_at, generated_by,
			 total_actions, total_blocked, total_rogue, overall_risk, summary,
			 rsi_level, content_hash, signature, signature_alg, signing_key_id,
			 json_path, pdf_path, stix_path, full_report)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (id) DO UPDATE SET
			json_path = EXCLUDED.json_path,
			pdf_path = EXCLUDED.pdf_path,
			stix_path = EXCLUDED.stix_path,
			full_report = EXCLUDED.full_report
	`, r.ID, string(r.Period), r.PeriodStart, r.PeriodEnd, r.GeneratedAt, r.GeneratedBy,
		r.TotalActions, r.TotalBlockedActions, r.TotalRogueIncidents, r.OverallRiskLevel, r.Summary,
		rsiLevel, r.ContentHash, r.Signature, r.SignatureAlg, r.SigningKeyID,
		r.JSONPath, r.PDFPath, r.STIXPath, fullJSON)

	if err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}

// GetReport recupera un reporte por ID.
func (s *Store) GetReport(ctx context.Context, id string) (*Report, error) {
	var fullJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT full_report FROM gov_reports WHERE id = $1
	`, id).Scan(&fullJSON)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get report: %w", err)
	}

	var r Report
	if err := json.Unmarshal(fullJSON, &r); err != nil {
		return nil, fmt.Errorf("unmarshal report: %w", err)
	}
	return &r, nil
}

// ListReports devuelve los últimos reportes.
func (s *Store) ListReports(ctx context.Context, limit int) ([]*Report, error) {
	query := `
		SELECT full_report FROM gov_reports
		ORDER BY generated_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var out []*Report
	for rows.Next() {
		var fullJSON []byte
		if err := rows.Scan(&fullJSON); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		var r Report
		if err := json.Unmarshal(fullJSON, &r); err != nil {
			return nil, fmt.Errorf("unmarshal report: %w", err)
		}
		out = append(out, &r)
	}

	return out, rows.Err()
}

// ListReportsByPeriod devuelve los reportes de un período.
func (s *Store) ListReportsByPeriod(ctx context.Context, period ReportPeriod, since time.Time) ([]*Report, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT full_report FROM gov_reports
		WHERE period = $1 AND generated_at >= $2
		ORDER BY generated_at DESC
	`, string(period), since)
	if err != nil {
		return nil, fmt.Errorf("list reports by period: %w", err)
	}
	defer rows.Close()

	var out []*Report
	for rows.Next() {
		var fullJSON []byte
		if err := rows.Scan(&fullJSON); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		var r Report
		if err := json.Unmarshal(fullJSON, &r); err != nil {
			return nil, fmt.Errorf("unmarshal report: %w", err)
		}
		out = append(out, &r)
	}

	return out, rows.Err()
}

// CountByRisk cuenta reportes por nivel de riesgo.
func (s *Store) CountByRisk(ctx context.Context) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT overall_risk, COUNT(*) FROM gov_reports GROUP BY overall_risk
	`)
	if err != nil {
		return nil, fmt.Errorf("count by risk: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var risk string
		var count int
		if err := rows.Scan(&risk, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		out[risk] = count
	}
	return out, rows.Err()
}

// DeleteReport elimina un reporte.
func (s *Store) DeleteReport(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM gov_reports WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete report: %w", err)
	}
	return nil
}
