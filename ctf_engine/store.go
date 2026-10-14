// Nexus IA v6.0 — Persistencia del CTF Engine en PostgreSQL
//
// Guarda:
//   - Retos cargados
//   - Intentos (con timing y técnicas)
//   - Métricas calculadas
//   - Rankings de agentes
package ctf_engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persiste el estado del CTF Engine.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore construye el store desde un pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// schemaSQL define las tablas del CTF Engine.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS ctf_challenges (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    platform        TEXT NOT NULL,
    category        TEXT NOT NULL,
    difficulty      TEXT NOT NULL,
    description     TEXT,
    files           TEXT[] NOT NULL DEFAULT '{}',
    hints           JSONB,
    flag_format     TEXT,
    points          INT NOT NULL DEFAULT 0,
    solved          BOOLEAN NOT NULL DEFAULT FALSE,
    solved_by       TEXT,
    solved_at       TIMESTAMPTZ,
    flag            TEXT,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ctf_challenges_platform ON ctf_challenges(platform);
CREATE INDEX IF NOT EXISTS idx_ctf_challenges_category ON ctf_challenges(category);
CREATE INDEX IF NOT EXISTS idx_ctf_challenges_solved ON ctf_challenges(solved);

CREATE TABLE IF NOT EXISTS ctf_attempts (
    id              TEXT PRIMARY KEY,
    challenge_id    TEXT NOT NULL REFERENCES ctf_challenges(id) ON DELETE CASCADE,
    agent_id        TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    solved          BOOLEAN NOT NULL DEFAULT FALSE,
    flag            TEXT,
    time_spent_ns   BIGINT NOT NULL DEFAULT 0,
    hints_used      INT NOT NULL DEFAULT 0,
    techniques      TEXT[] NOT NULL DEFAULT '{}',
    mitre_ttps      TEXT[] NOT NULL DEFAULT '{}',
    trace_id        TEXT,
    session_id      TEXT,
    notes           TEXT
);

CREATE INDEX IF NOT EXISTS idx_ctf_attempts_challenge ON ctf_attempts(challenge_id);
CREATE INDEX IF NOT EXISTS idx_ctf_attempts_agent ON ctf_attempts(agent_id);
CREATE INDEX IF NOT EXISTS idx_ctf_attempts_solved ON ctf_attempts(solved);

CREATE TABLE IF NOT EXISTS ctf_metrics (
    challenge_id    TEXT PRIMARY KEY REFERENCES ctf_challenges(id) ON DELETE CASCADE,
    platform        TEXT NOT NULL,
    category        TEXT NOT NULL,
    difficulty      TEXT NOT NULL,
    time_spent_ns   BIGINT NOT NULL,
    hints_used      INT NOT NULL,
    techniques      TEXT[] NOT NULL DEFAULT '{}',
    human_avg_time_ns BIGINT NOT NULL DEFAULT 0,
    human_avg_hints DOUBLE PRECISION NOT NULL DEFAULT 0,
    percentile      DOUBLE PRECISION NOT NULL DEFAULT 0,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ctf_rankings (
    agent_id            TEXT PRIMARY KEY,
    total_solves        INT NOT NULL DEFAULT 0,
    total_attempts      INT NOT NULL DEFAULT 0,
    solve_rate          DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_time_spent_ns   BIGINT NOT NULL DEFAULT 0,
    avg_hints_used      DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_percentile      DOUBLE PRECISION NOT NULL DEFAULT 0,
    easy_solves         INT NOT NULL DEFAULT 0,
    medium_solves       INT NOT NULL DEFAULT 0,
    hard_solves         INT NOT NULL DEFAULT 0,
    insane_solves       INT NOT NULL DEFAULT 0,
    by_category         JSONB,
    first_solve_at      TIMESTAMPTZ,
    last_solve_at       TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ctf_rankings_solves ON ctf_rankings(total_solves DESC);
CREATE INDEX IF NOT EXISTS idx_ctf_rankings_pct ON ctf_rankings(avg_percentile DESC);
`

// Migrate crea el schema del CTF Engine.
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("migrate ctf schema: %w", err)
	}
	return nil
}

// SaveChallenge persiste un reto.
func (s *Store) SaveChallenge(ctx context.Context, ch *Challenge) error {
	hintsJSON, err := json.Marshal(ch.Hints)
	if err != nil {
		return fmt.Errorf("marshal hints: %w", err)
	}

	var metadataJSON []byte
	if ch.Metadata != nil {
		metadataJSON, err = json.Marshal(ch.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO ctf_challenges
			(id, name, platform, category, difficulty, description, files, hints,
			 flag_format, points, solved, solved_by, solved_at, flag, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			files = EXCLUDED.files,
			hints = EXCLUDED.hints,
			solved = EXCLUDED.solved,
			solved_by = EXCLUDED.solved_by,
			solved_at = EXCLUDED.solved_at,
			flag = EXCLUDED.flag,
			metadata = EXCLUDED.metadata
	`, ch.ID, ch.Name, string(ch.Platform), string(ch.Category), string(ch.Difficulty),
		ch.Description, ch.Files, hintsJSON, ch.FlagFormat, ch.Points,
		ch.Solved, ch.SolvedBy, ch.SolvedAt, ch.Flag, metadataJSON, ch.CreatedAt)

	if err != nil {
		return fmt.Errorf("save challenge: %w", err)
	}
	return nil
}

// GetChallenge recupera un reto por ID.
func (s *Store) GetChallenge(ctx context.Context, id string) (*Challenge, error) {
	var (
		ch          Challenge
		platformStr string
		catStr      string
		diffStr     string
		hintsRaw    []byte
		metadataRaw []byte
		desc        *string
		flagFormat  *string
		solvedBy    *string
		flag        *string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, platform, category, difficulty, description, files, hints,
		       flag_format, points, solved, solved_by, solved_at, flag, metadata, created_at
		FROM ctf_challenges WHERE id = $1
	`, id).Scan(
		&ch.ID, &ch.Name, &platformStr, &catStr, &diffStr, &desc, &ch.Files, &hintsRaw,
		&flagFormat, &ch.Points, &ch.Solved, &solvedBy, &ch.SolvedAt, &flag, &metadataRaw, &ch.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get challenge: %w", err)
	}

	ch.Platform = Platform(platformStr)
	ch.Category = Category(catStr)
	ch.Difficulty = Difficulty(diffStr)
	if desc != nil {
		ch.Description = *desc
	}
	if flagFormat != nil {
		ch.FlagFormat = *flagFormat
	}
	if solvedBy != nil {
		ch.SolvedBy = *solvedBy
	}
	if flag != nil {
		ch.Flag = *flag
	}
	if hintsRaw != nil {
		_ = json.Unmarshal(hintsRaw, &ch.Hints)
	}
	if metadataRaw != nil {
		_ = json.Unmarshal(metadataRaw, &ch.Metadata)
	}

	return &ch, nil
}

// SaveAttempt persiste un intento.
func (s *Store) SaveAttempt(ctx context.Context, a *Attempt) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ctf_attempts
			(id, challenge_id, agent_id, started_at, finished_at, solved, flag,
			 time_spent_ns, hints_used, techniques, mitre_ttps,
			 trace_id, session_id, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			finished_at = EXCLUDED.finished_at,
			solved = EXCLUDED.solved,
			flag = EXCLUDED.flag,
			time_spent_ns = EXCLUDED.time_spent_ns,
			hints_used = EXCLUDED.hints_used,
			techniques = EXCLUDED.techniques,
			notes = EXCLUDED.notes
	`, a.ID, a.ChallengeID, a.AgentID, a.StartedAt, a.FinishedAt, a.Solved, a.Flag,
		int64(a.TimeSpent), a.HintsUsed, a.Techniques, a.MITRE_TTPs,
		a.TraceID, a.SessionID, a.Notes)

	if err != nil {
		return fmt.Errorf("save attempt: %w", err)
	}
	return nil
}

// SaveMetrics persiste las métricas de un reto.
func (s *Store) SaveMetrics(ctx context.Context, m *SolveMetrics) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO ctf_metrics
			(challenge_id, platform, category, difficulty, time_spent_ns, hints_used,
			 techniques, human_avg_time_ns, human_avg_hints, percentile)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (challenge_id) DO UPDATE SET
			time_spent_ns = EXCLUDED.time_spent_ns,
			hints_used = EXCLUDED.hints_used,
			techniques = EXCLUDED.techniques,
			percentile = EXCLUDED.percentile,
			computed_at = NOW()
	`, m.ChallengeID, string(m.Platform), string(m.Category), string(m.Difficulty),
		int64(m.TimeSpent), m.HintsUsed, m.Techniques,
		int64(m.HumanAvgTime), m.HumanAvgHints, m.Percentile)

	if err != nil {
		return fmt.Errorf("save metrics: %w", err)
	}
	return nil
}

// SaveRanking persiste el ranking de un agente.
func (s *Store) SaveRanking(ctx context.Context, r *AgentRanking) error {
	byCategoryJSON, err := json.Marshal(r.ByCategory)
	if err != nil {
		return fmt.Errorf("marshal by_category: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO ctf_rankings
			(agent_id, total_solves, total_attempts, solve_rate, avg_time_spent_ns,
			 avg_hints_used, avg_percentile, easy_solves, medium_solves, hard_solves,
			 insane_solves, by_category, first_solve_at, last_solve_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (agent_id) DO UPDATE SET
			total_solves = EXCLUDED.total_solves,
			total_attempts = EXCLUDED.total_attempts,
			solve_rate = EXCLUDED.solve_rate,
			avg_time_spent_ns = EXCLUDED.avg_time_spent_ns,
			avg_hints_used = EXCLUDED.avg_hints_used,
			avg_percentile = EXCLUDED.avg_percentile,
			easy_solves = EXCLUDED.easy_solves,
			medium_solves = EXCLUDED.medium_solves,
			hard_solves = EXCLUDED.hard_solves,
			insane_solves = EXCLUDED.insane_solves,
			by_category = EXCLUDED.by_category,
			first_solve_at = EXCLUDED.first_solve_at,
			last_solve_at = EXCLUDED.last_solve_at,
			updated_at = NOW()
	`, r.AgentID, r.TotalSolves, r.TotalAttempts, r.SolveRate, int64(r.AvgTimeSpent),
		r.AvgHintsUsed, r.AvgPercentile, r.EasySolves, r.MediumSolves, r.HardSolves,
		r.InsaneSolves, byCategoryJSON, r.FirstSolveAt, r.LastSolveAt)

	if err != nil {
		return fmt.Errorf("save ranking: %w", err)
	}
	return nil
}

// ListAttemptsByChallenge recupera los intentos de un reto.
func (s *Store) ListAttemptsByChallenge(ctx context.Context, challengeID string) ([]*Attempt, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, challenge_id, agent_id, started_at, finished_at, solved, flag,
		       time_spent_ns, hints_used, techniques, mitre_ttps,
		       trace_id, session_id, notes
		FROM ctf_attempts
		WHERE challenge_id = $1
		ORDER BY started_at DESC
	`, challengeID)
	if err != nil {
		return nil, fmt.Errorf("list attempts: %w", err)
	}
	defer rows.Close()

	var out []*Attempt
	for rows.Next() {
		var (
			a           Attempt
			timeSpentNS int64
			flag        *string
			traceID     *string
			sessionID   *string
			notes       *string
		)
		err := rows.Scan(&a.ID, &a.ChallengeID, &a.AgentID, &a.StartedAt, &a.FinishedAt,
			&a.Solved, &flag, &timeSpentNS, &a.HintsUsed, &a.Techniques, &a.MITRE_TTPs,
			&traceID, &sessionID, &notes)
		if err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		a.TimeSpent = time.Duration(timeSpentNS)
		if flag != nil {
			a.Flag = *flag
		}
		if traceID != nil {
			a.TraceID = *traceID
		}
		if sessionID != nil {
			a.SessionID = *sessionID
		}
		if notes != nil {
			a.Notes = *notes
		}
		out = append(out, &a)
	}

	return out, rows.Err()
}

// Stats devuelve estadísticas globales desde la DB.
func (s *Store) Stats(ctx context.Context) (EngineStats, error) {
	stats := EngineStats{
		ByPlatform:   make(map[Platform]int),
		ByCategory:   make(map[Category]int),
		ByDifficulty: make(map[Difficulty]int),
	}

	// Total challenges + solved
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE solved)
		FROM ctf_challenges
	`).Scan(&stats.TotalChallenges, &stats.TotalSolved)
	if err != nil {
		return stats, fmt.Errorf("count challenges: %w", err)
	}

	// Total attempts
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ctf_attempts`).Scan(&stats.TotalAttempts)
	if err != nil {
		return stats, fmt.Errorf("count attempts: %w", err)
	}

	// Total agents
	err = s.pool.QueryRow(ctx, `SELECT COUNT(DISTINCT agent_id) FROM ctf_attempts`).Scan(&stats.TotalAgents)
	if err != nil {
		return stats, fmt.Errorf("count agents: %w", err)
	}

	return stats, nil
}
