// Client-facing assessment results. This is the ONLY path by which a portal
// client reads engine output, and it is deliberately narrow for real-world
// safety:
//   - tenant isolation: every read is resolved from (request id + caller org),
//     so a client can never reach another org's task or findings;
//   - exposure control: only a curated subset of fields is returned (no raw
//     evidence, no internal worker/agent names, no internal asset/node ids), and
//     only findings in a client-safe status (triage noise and false positives
//     are excluded);
//   - operator gate: nothing is shown until an operator releases the results.
package db

import (
	"database/sql"
	"errors"
	"time"
)

// ClientFinding is the safe, curated view of a finding shown to a portal client.
type ClientFinding struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	VulnClass string    `json:"vulnclass"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// ClientFindingDetail adds the Markdown report for a single finding's detail view.
type ClientFindingDetail struct {
	ClientFinding
	Report string `json:"report"`
}

// clientVisibleStatuses are the finding states a client is allowed to see:
// validated/actionable results only — never pending triage, false positives,
// duplicates or ignored noise.
const clientVisibleStatuses = `('confirmed','in_progress','resolved','fixed','risk_accepted')`

// severity ordering so the client sees the most serious findings first.
const severityOrder = `CASE severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END`

// ResolveClientRequest returns the engine task and release flag for a request,
// scoped to the caller's org. exists is false when the request does not belong to
// that org (or does not exist) — the caller treats both as 404.
func (d *DB) ResolveClientRequest(requestID, orgID int64) (taskID int64, released bool, exists bool, err error) {
	var tid sql.NullInt64
	err = d.QueryRow(
		`SELECT engine_task_id, results_released FROM portal_requests WHERE id = $1 AND org_id = $2`,
		requestID, orgID,
	).Scan(&tid, &released)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, err
	}
	if tid.Valid {
		taskID = tid.Int64
	}
	return taskID, released, true, nil
}

// ListClientFindings returns the curated, client-visible findings for a task.
func (d *DB) ListClientFindings(taskID int64) ([]ClientFinding, error) {
	rows, err := d.Query(
		`SELECT id, name, vulnclass, severity, status, summary, created_at
         FROM findings
         WHERE task_id = $1 AND status IN `+clientVisibleStatuses+`
         ORDER BY `+severityOrder+`, created_at DESC LIMIT 1000`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ClientFinding{}
	for rows.Next() {
		var f ClientFinding
		if err := rows.Scan(&f.ID, &f.Name, &f.VulnClass, &f.Severity, &f.Status, &f.Summary, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetClientFinding returns one finding's detail, scoped to the task AND limited to
// client-visible statuses (so a client cannot read an internal-only finding by
// guessing its id).
func (d *DB) GetClientFinding(taskID, findingID int64) (*ClientFindingDetail, error) {
	var f ClientFindingDetail
	err := d.QueryRow(
		`SELECT id, name, vulnclass, severity, status, summary, report, created_at
         FROM findings
         WHERE id = $1 AND task_id = $2 AND status IN `+clientVisibleStatuses, findingID, taskID,
	).Scan(&f.ID, &f.Name, &f.VulnClass, &f.Severity, &f.Status, &f.Summary, &f.Report, &f.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// SetRequestResultsReleased toggles whether a request's results are visible to
// the client.
func (d *DB) SetRequestResultsReleased(requestID int64, released bool) error {
	res, err := d.Exec(`UPDATE portal_requests SET results_released = $1 WHERE id = $2`, released, requestID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
