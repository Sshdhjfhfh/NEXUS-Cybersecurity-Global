package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nexus-ia/nexus/orchestrator/level12"
)

// Level12Store persists reports and confirmed feedback independently of process memory.
type Level12Store struct{ db *DB }

func (d *DB) Level12() *Level12Store { return &Level12Store{db: d} }
func (s *Level12Store) SaveRun(ctx context.Context, run level12.Run) error {
	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}
	if len(raw) > 4<<20 {
		return level12.ErrInvalid
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO lab_analysis_runs(id,task_id,target,report,started_at,finished_at) VALUES($1,$2,$3,$4,$5,$6)`, run.ID, run.TaskID, run.Target, raw, run.StartedAt, run.FinishedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return level12.ErrConflict
	}
	return err
}
func decodeLabRun(raw []byte) (*level12.Run, error) {
	if len(raw) > 4<<20 {
		return nil, fmt.Errorf("laboratory report limit exceeded")
	}
	var run level12.Run
	if err := json.Unmarshal(raw, &run); err != nil {
		return nil, err
	}
	return &run, nil
}
func (s *Level12Store) GetRun(ctx context.Context, taskID int64, id string) (*level12.Run, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT report FROM lab_analysis_runs WHERE task_id=$1 AND id=$2`, taskID, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, level12.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeLabRun(raw)
}
func (s *Level12Store) ListRuns(ctx context.Context, taskID int64, limit int) ([]level12.Run, error) {
	if taskID <= 0 || limit < 1 || limit > 100 {
		return nil, level12.ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT report FROM lab_analysis_runs WHERE task_id=$1 ORDER BY started_at DESC,id DESC LIMIT $2`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := []level12.Run{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		run, err := decodeLabRun(raw)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	return runs, rows.Err()
}
func (s *Level12Store) AddFeedback(ctx context.Context, taskID int64, id string, f level12.Feedback) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT report FROM lab_analysis_runs WHERE task_id=$1 AND id=$2 FOR KEY SHARE`, taskID, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return level12.ErrNotFound
	}
	if err != nil {
		return err
	}
	run, err := decodeLabRun(raw)
	if err != nil {
		return err
	}
	valid := false
	for _, finding := range run.Findings {
		if finding.Rule == f.Class && finding.EvidenceID == f.EvidenceID {
			valid = true
			break
		}
	}
	if !valid {
		return level12.ErrInvalid
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO lab_prediction_feedback(run_id,task_id,class,evidence_id,confirmed) VALUES($1,$2,$3,$4,$5) ON CONFLICT(run_id,class) DO NOTHING`, id, taskID, f.Class, f.EvidenceID, f.Confirmed)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return level12.ErrConflict
	}
	return tx.Commit()
}
func (s *Level12Store) Outcomes(ctx context.Context, taskID int64) ([]level12.Outcome, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT run_id,class,confirmed FROM lab_prediction_feedback WHERE task_id=$1 ORDER BY run_id,class LIMIT 100001`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []level12.Outcome{}
	for rows.Next() {
		var o level12.Outcome
		if err := rows.Scan(&o.RunID, &o.Class, &o.Confirmed); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
