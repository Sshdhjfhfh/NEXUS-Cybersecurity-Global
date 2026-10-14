// Nexus IA v6.0 — Persistencia del RSI Detector en PostgreSQL
//
// Guarda:
//   - Muestras de los 4 indicadores (para análisis histórico)
//   - Estados calculados
//   - Alertas RSI emitidas
//   - Acks de alertas por investigadores
package rsi_detector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persiste el estado del RSI Detector.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye el store desde un pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del RSI Detector.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS rsi_samples (
    id          BIGSERIAL PRIMARY KEY,
    indicator   TEXT NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    window_ns   BIGINT NOT NULL DEFAULT 0,
    metadata    JSONB
);

CREATE INDEX IF NOT EXISTS idx_rsi_samples_ind ON rsi_samples(indicator);
CREATE INDEX IF NOT EXISTS idx_rsi_samples_ts ON rsi_samples(timestamp DESC);

CREATE TABLE IF NOT EXISTS rsi_reports (
    id              BIGSERIAL PRIMARY KEY,
    level           TEXT NOT NULL,
    summary         TEXT NOT NULL,
    recommendation  TEXT NOT NULL,
    data_points     INT NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL,
    statuses_json   JSONB
);

CREATE INDEX IF NOT EXISTS idx_rsi_reports_ts ON rsi_reports(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_rsi_reports_level ON rsi_reports(level);

CREATE TABLE IF NOT EXISTS rsi_alerts (
    id              TEXT PRIMARY KEY,
    level           TEXT NOT NULL,
    message         TEXT NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL,
    acknowledged    BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_by TEXT,
    acknowledged_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_rsi_alerts_ts ON rsi_alerts(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_rsi_alerts_level ON rsi_alerts(level);
CREATE INDEX IF NOT EXISTS idx_rsi_alerts_ack ON rsi_alerts(acknowledged);
`

// Migrate crea el schema.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate rsi schema: %w", err)
	}
	return nil
}

// SaveSample persiste una muestra.
func (s *Store) SaveSample(ctx context.Context, sample *Sample) error {
	var metadataJSON []byte
	if sample.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(sample.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO rsi_samples (indicator, value, timestamp, window_ns, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, string(sample.Indicator), sample.Value, sample.Timestamp, int64(sample.Window), metadataJSON)

	if err != nil {
		return fmt.Errorf("save sample: %w", err)
	}
	return nil
}

// LoadSamples recupera las muestras de un indicador.
func (s *Store) LoadSamples(ctx context.Context, ind Indicator, since time.Time, limit int) ([]*Sample, error) {
	query := `
		SELECT indicator, value, timestamp, window_ns, metadata
		FROM rsi_samples
		WHERE indicator = $1 AND timestamp >= $2
		ORDER BY timestamp DESC
	`
	if limit > 0 {
		query += " LIMIT $3"
	}

	var rows pgx.Rows
	var err error
	if limit > 0 {
		rows, err = s.pool.Query(ctx, query, string(ind), since, limit)
	} else {
		rows, err = s.pool.Query(ctx, query, string(ind), since)
	}
	if err != nil {
		return nil, fmt.Errorf("load samples: %w", err)
	}
	defer rows.Close()

	var out []*Sample
	for rows.Next() {
		var (
			sample      Sample
			indStr      string
			windowNS    int64
			metadataRaw []byte
		)
		err := rows.Scan(&indStr, &sample.Value, &sample.Timestamp, &windowNS, &metadataRaw)
		if err != nil {
			return nil, fmt.Errorf("scan sample: %w", err)
		}
		sample.Indicator = Indicator(indStr)
		sample.Window = time.Duration(windowNS)
		if metadataRaw != nil {
			_ = json.Unmarshal(metadataRaw, &sample.Metadata)
		}
		out = append(out, &sample)
	}

	return out, rows.Err()
}

// SaveReport persiste un reporte.
func (s *Store) SaveReport(ctx context.Context, r *RSIReport) error {
	statusesJSON, err := json.Marshal(r.Statuses)
	if err != nil {
		return fmt.Errorf("marshal statuses: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO rsi_reports (level, summary, recommendation, data_points, timestamp, statuses_json)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, string(r.Level), r.Summary, r.Recommendation, r.DataPoints, r.Timestamp, statusesJSON)

	if err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}

// SaveAlert persiste una alerta.
func (s *Store) SaveAlert(ctx context.Context, a *RSIAlert) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rsi_alerts (id, level, message, timestamp, acknowledged)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, a.ID, string(a.Level), a.Message, a.Timestamp, a.Acknowledged)

	if err != nil {
		return fmt.Errorf("save alert: %w", err)
	}
	return nil
}

// AcknowledgeAlert marca una alerta como reconocida.
func (s *Store) AcknowledgeAlert(ctx context.Context, alertID, by string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE rsi_alerts
		SET acknowledged = TRUE, acknowledged_by = $2, acknowledged_at = NOW()
		WHERE id = $1
	`, alertID, by)

	if err != nil {
		return fmt.Errorf("acknowledge alert: %w", err)
	}
	return nil
}

// ListAlerts devuelve las alertas recientes.
func (s *Store) ListAlerts(ctx context.Context, limit int) ([]*RSIAlert, error) {
	query := `
		SELECT id, level, message, timestamp, acknowledged, acknowledged_by, acknowledged_at
		FROM rsi_alerts
		ORDER BY timestamp DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var out []*RSIAlert
	for rows.Next() {
		var (
			a          RSIAlert
			levelStr   string
			ackBy      *string
			ackAt      *time.Time
		)
		err := rows.Scan(&a.ID, &levelStr, &a.Message, &a.Timestamp, &a.Acknowledged, &ackBy, &ackAt)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Level = AlertLevel(levelStr)
		if ackBy != nil {
			a.AcknowledgedBy = *ackBy
		}
		a.AcknowledgedAt = ackAt
		out = append(out, &a)
	}

	return out, rows.Err()
}

// CountSamplesByIndicator cuenta las muestras de un indicador.
func (s *Store) CountSamplesByIndicator(ctx context.Context, ind Indicator) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM rsi_samples WHERE indicator = $1
	`, string(ind)).Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("count samples: %w", err)
	}
	return count, nil
}
