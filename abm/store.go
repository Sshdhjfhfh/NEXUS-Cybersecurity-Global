// Nexus IA v6.0 — Persistencia del ABM en PostgreSQL
//
// Guarda:
//   - Violaciones detectadas (inmutables)
//   - Señales de comportamiento
//   - Perfiles de baseline
//   - Estado de agentes detenidos/cuarentenados
//   - Alertas emitidas
package abm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persiste el estado del ABM.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye el store desde un pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del ABM.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS abm_violations (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL,
    severity        INT NOT NULL,
    agent_id        TEXT NOT NULL,
    agent_type      TEXT NOT NULL,
    action          TEXT NOT NULL,
    target          TEXT NOT NULL,
    evidence        TEXT,
    session_id      TEXT,
    trace_id        TEXT,
    timestamp       TIMESTAMPTZ NOT NULL,
    metadata        JSONB,
    action_taken    TEXT
);

CREATE INDEX IF NOT EXISTS idx_abm_violations_agent ON abm_violations(agent_id);
CREATE INDEX IF NOT EXISTS idx_abm_violations_type ON abm_violations(type);
CREATE INDEX IF NOT EXISTS idx_abm_violations_ts ON abm_violations(timestamp DESC);

CREATE TABLE IF NOT EXISTS abm_signals (
    id          BIGSERIAL PRIMARY KEY,
    agent_id    TEXT NOT NULL,
    type        TEXT NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    baseline    DOUBLE PRECISION NOT NULL,
    threshold   DOUBLE PRECISION NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL,
    raw         TEXT
);

CREATE INDEX IF NOT EXISTS idx_abm_signals_agent ON abm_signals(agent_id);
CREATE INDEX IF NOT EXISTS idx_abm_signals_ts ON abm_signals(timestamp DESC);

CREATE TABLE IF NOT EXISTS abm_profiles (
    agent_id                TEXT PRIMARY KEY,
    avg_actions_per_minute  DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_actions_per_minute  DOUBLE PRECISION NOT NULL DEFAULT 0,
    std_actions_per_minute  DOUBLE PRECISION NOT NULL DEFAULT 0,
    known_targets           TEXT[] NOT NULL DEFAULT '{}',
    known_ports             INT[] NOT NULL DEFAULT '{}',
    known_protocols         TEXT[] NOT NULL DEFAULT '{}',
    avg_cpu_usage           DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_memory_mb           DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_file_ops            DOUBLE PRECISION NOT NULL DEFAULT 0,
    active_hours            INT[] NOT NULL DEFAULT '{}',
    observed_since          TIMESTAMPTZ,
    observed_until          TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS abm_alerts (
    id          TEXT PRIMARY KEY,
    severity    TEXT NOT NULL,
    agent_id    TEXT NOT NULL,
    message     TEXT NOT NULL,
    violation_id TEXT,
    timestamp   TIMESTAMPTZ NOT NULL,
    actions     TEXT[] NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_abm_alerts_agent ON abm_alerts(agent_id);
CREATE INDEX IF NOT EXISTS idx_abm_alerts_sev ON abm_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_abm_alerts_ts ON abm_alerts(timestamp DESC);

CREATE TABLE IF NOT EXISTS abm_agent_state (
    agent_id        TEXT PRIMARY KEY,
    stopped         BOOLEAN NOT NULL DEFAULT FALSE,
    quarantined     BOOLEAN NOT NULL DEFAULT FALSE,
    stop_reason     TEXT,
    stop_time       TIMESTAMPTZ,
    last_update     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// Migrate crea el schema del ABM.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate abm schema: %w", err)
	}
	return nil
}

// SaveViolation persiste una violación.
func (s *Store) SaveViolation(ctx context.Context, v *Violation) error {
	var metadataJSON []byte
	if v.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(v.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO abm_violations
			(id, type, severity, agent_id, agent_type, action, target,
			 evidence, session_id, trace_id, timestamp, metadata, action_taken)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO NOTHING
	`, v.ID, string(v.Type), int(v.Severity), v.AgentID, v.AgentType,
		v.Action, v.Target, v.Evidence, v.SessionID, v.TraceID,
		v.Timestamp, metadataJSON, v.ActionTaken)

	if err != nil {
		return fmt.Errorf("save violation: %w", err)
	}
	return nil
}

// ListViolations devuelve las violaciones de un agente.
func (s *Store) ListViolations(ctx context.Context, agentID string, limit int) ([]*Violation, error) {
	query := `
		SELECT id, type, severity, agent_id, agent_type, action, target,
		       evidence, session_id, trace_id, timestamp, metadata, action_taken
		FROM abm_violations
		WHERE agent_id = $1
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
		return nil, fmt.Errorf("list violations: %w", err)
	}
	defer rows.Close()

	var out []*Violation
	for rows.Next() {
		var (
			v           Violation
			typeStr     string
			sevInt      int
			metadataRaw []byte
			evidence    *string
			sessionID   *string
			traceID     *string
			actionTaken *string
		)
		err := rows.Scan(
			&v.ID, &typeStr, &sevInt, &v.AgentID, &v.AgentType, &v.Action, &v.Target,
			&evidence, &sessionID, &traceID, &v.Timestamp, &metadataRaw, &actionTaken)
		if err != nil {
			return nil, fmt.Errorf("scan violation: %w", err)
		}
		v.Type = ViolationType(typeStr)
		v.Severity = Severity(sevInt)
		if evidence != nil {
			v.Evidence = *evidence
		}
		if sessionID != nil {
			v.SessionID = *sessionID
		}
		if traceID != nil {
			v.TraceID = *traceID
		}
		if actionTaken != nil {
			v.ActionTaken = *actionTaken
		}
		if metadataRaw != nil {
			_ = json.Unmarshal(metadataRaw, &v.Metadata)
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

// SaveProfile persiste un perfil de baseline.
func (s *Store) SaveProfile(ctx context.Context, p *BehavioralProfile) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO abm_profiles
			(agent_id, avg_actions_per_minute, max_actions_per_minute, std_actions_per_minute,
			 known_targets, known_ports, known_protocols,
			 avg_cpu_usage, avg_memory_mb, avg_file_ops, active_hours,
			 observed_since, observed_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (agent_id) DO UPDATE SET
			avg_actions_per_minute = EXCLUDED.avg_actions_per_minute,
			max_actions_per_minute = EXCLUDED.max_actions_per_minute,
			std_actions_per_minute = EXCLUDED.std_actions_per_minute,
			known_targets = EXCLUDED.known_targets,
			known_ports = EXCLUDED.known_ports,
			known_protocols = EXCLUDED.known_protocols,
			avg_cpu_usage = EXCLUDED.avg_cpu_usage,
			avg_memory_mb = EXCLUDED.avg_memory_mb,
			avg_file_ops = EXCLUDED.avg_file_ops,
			active_hours = EXCLUDED.active_hours,
			observed_since = EXCLUDED.observed_since,
			observed_until = EXCLUDED.observed_until
	`, p.AgentID, p.AvgActionsPerMinute, p.MaxActionsPerMinute, p.StdActionsPerMinute,
		p.KnownTargets, p.KnownPorts, p.KnownProtocols,
		p.AvgCPUUsage, p.AvgMemoryMB, p.AvgFileOps, p.ActiveHours,
		p.ObservedSince, p.ObservedUntil)

	if err != nil {
		return fmt.Errorf("save profile: %w", err)
	}
	return nil
}

// GetProfile recupera un perfil.
func (s *Store) GetProfile(ctx context.Context, agentID string) (*BehavioralProfile, error) {
	var p BehavioralProfile
	err := s.pool.QueryRow(ctx, `
		SELECT agent_id, avg_actions_per_minute, max_actions_per_minute, std_actions_per_minute,
		       known_targets, known_ports, known_protocols,
		       avg_cpu_usage, avg_memory_mb, avg_file_ops, active_hours,
		       observed_since, observed_until
		FROM abm_profiles
		WHERE agent_id = $1
	`, agentID).Scan(
		&p.AgentID, &p.AvgActionsPerMinute, &p.MaxActionsPerMinute, &p.StdActionsPerMinute,
		&p.KnownTargets, &p.KnownPorts, &p.KnownProtocols,
		&p.AvgCPUUsage, &p.AvgMemoryMB, &p.AvgFileOps, &p.ActiveHours,
		&p.ObservedSince, &p.ObservedUntil)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	return &p, nil
}

// SaveAlert persiste una alerta.
func (s *Store) SaveAlert(ctx context.Context, a *Alert) error {
	var violationID *string
	if a.Violation != nil {
		violationID = &a.Violation.ID
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO abm_alerts
			(id, severity, agent_id, message, violation_id, timestamp, actions)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, a.ID, string(a.Severity), a.AgentID, a.Message, violationID, a.Timestamp, a.Actions)

	if err != nil {
		return fmt.Errorf("save alert: %w", err)
	}
	return nil
}

// ListAlerts devuelve las alertas de un agente.
func (s *Store) ListAlerts(ctx context.Context, agentID string, limit int) ([]*Alert, error) {
	query := `
		SELECT id, severity, agent_id, message, violation_id, timestamp, actions
		FROM abm_alerts
		WHERE agent_id = $1
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
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var out []*Alert
	for rows.Next() {
		var (
			a         Alert
			sevStr    string
			violID    *string
		)
		err := rows.Scan(&a.ID, &sevStr, &a.AgentID, &a.Message, &violID, &a.Timestamp, &a.Actions)
		if err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Severity = AlertSeverity(sevStr)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// SaveAgentState persiste el estado detenido/cuarentenado de un agente.
func (s *Store) SaveAgentState(ctx context.Context, agentID string, stopped, quarantined bool, reason string) error {
	var stopTime *time.Time
	if stopped || quarantined {
		now := time.Now()
		stopTime = &now
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO abm_agent_state
			(agent_id, stopped, quarantined, stop_reason, stop_time, last_update)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (agent_id) DO UPDATE SET
			stopped = EXCLUDED.stopped,
			quarantined = EXCLUDED.quarantined,
			stop_reason = EXCLUDED.stop_reason,
			stop_time = COALESCE(EXCLUDED.stop_time, abm_agent_state.stop_time),
			last_update = NOW()
	`, agentID, stopped, quarantined, reason, stopTime)

	if err != nil {
		return fmt.Errorf("save agent state: %w", err)
	}
	return nil
}

// GetAgentState recupera el estado de un agente.
func (s *Store) GetAgentState(ctx context.Context, agentID string) (stopped, quarantined bool, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT stopped, quarantined FROM abm_agent_state WHERE agent_id = $1
	`, agentID).Scan(&stopped, &quarantined)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	return stopped, quarantined, err
}
