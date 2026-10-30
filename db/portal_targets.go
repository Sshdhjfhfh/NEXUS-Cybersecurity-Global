// Target ownership verification and Rules of Engagement (RoE). Testing a system
// without its owner's authorization is illegal in most jurisdictions, so NEXUS
// only tests targets an org has proven it controls (or an operator approved on
// documentary evidence), within a window an org admin accepted in writing.
package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// TargetVerificationValidity is how long a verification counts; after that the
// org must prove control again before a new evaluation can start.
const TargetVerificationValidity = 365 * 24 * time.Hour

// ErrTargetExists means the org already registered that target.
var ErrTargetExists = errors.New("target already registered")

// PortalTarget is a domain or IP an org asks NEXUS to test.
type PortalTarget struct {
	ID          int64      `json:"id"`
	OrgID       int64      `json:"org_id"`
	OrgName     string     `json:"org_name,omitempty"` // operator listings only
	Kind        string     `json:"kind"`               // domain | ip
	Value       string     `json:"value"`
	Token       string     `json:"token,omitempty"` // shown to the org for DNS/HTTP proof
	Status      string     `json:"status"`          // pending | verified | rejected | expired (derived)
	Method      string     `json:"method,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	VerifiedBy  string     `json:"verified_by,omitempty"`
	Note        string     `json:"note,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Usable reports whether the target may be tested right now.
func (t *PortalTarget) Usable() bool { return t.Status == "verified" }

const targetCols = `t.id, t.org_id, o.name, t.kind, t.value, t.token, t.status, t.method, t.verified_at,
       t.verified_by, t.note, t.last_check_at, t.last_error, t.created_at`

func scanTarget(row interface{ Scan(...any) error }, now time.Time) (*PortalTarget, error) {
	var t PortalTarget
	var verifiedAt, lastCheck sql.NullTime
	if err := row.Scan(&t.ID, &t.OrgID, &t.OrgName, &t.Kind, &t.Value, &t.Token, &t.Status, &t.Method, &verifiedAt,
		&t.VerifiedBy, &t.Note, &lastCheck, &t.LastError, &t.CreatedAt); err != nil {
		return nil, err
	}
	if verifiedAt.Valid {
		v := verifiedAt.Time
		exp := v.Add(TargetVerificationValidity)
		t.VerifiedAt, t.ExpiresAt = &v, &exp
		if t.Status == "verified" && now.After(exp) {
			t.Status = "expired"
		}
	}
	if lastCheck.Valid {
		lc := lastCheck.Time
		t.LastCheckAt = &lc
	}
	return &t, nil
}

func (d *DB) queryTargets(where string, args ...any) ([]*PortalTarget, error) {
	rows, err := d.Query(`SELECT `+targetCols+` FROM portal_targets t JOIN orgs o ON o.id = t.org_id `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now()
	out := []*PortalTarget{}
	for rows.Next() {
		t, err := scanTarget(rows, now)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreatePortalTarget registers a target (pending) with its proof token.
func (d *DB) CreatePortalTarget(orgID, userID int64, kind, value, token string) (int64, error) {
	var id int64
	err := d.QueryRow(`INSERT INTO portal_targets(org_id, kind, value, token, created_by)
VALUES ($1,$2,$3,$4,$5) RETURNING id`, orgID, kind, value, token, userID).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return 0, ErrTargetExists
	}
	return id, err
}

// ListPortalTargets returns an org's targets, newest first.
func (d *DB) ListPortalTargets(orgID int64) ([]*PortalTarget, error) {
	return d.queryTargets(`WHERE t.org_id = $1 ORDER BY t.created_at DESC LIMIT 1000`, orgID)
}

// GetPortalTarget returns one target of an org (nil if none — also for another
// org's target, so ids never leak across tenants).
func (d *DB) GetPortalTarget(orgID, id int64) (*PortalTarget, error) {
	list, err := d.queryTargets(`WHERE t.org_id = $1 AND t.id = $2`, orgID, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

// AdminGetPortalTarget returns any target (operator).
func (d *DB) AdminGetPortalTarget(id int64) (*PortalTarget, error) {
	list, err := d.queryTargets(`WHERE t.id = $1`, id)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

// AdminListPortalTargets lists targets across orgs; status "" = all.
func (d *DB) AdminListPortalTargets(status string) ([]*PortalTarget, error) {
	if status == "" {
		return d.queryTargets(`ORDER BY t.created_at DESC LIMIT 1000`)
	}
	return d.queryTargets(`WHERE t.status = $1 ORDER BY t.created_at ASC LIMIT 1000`, status)
}

// RecordTargetCheck stores the outcome of an automatic (DNS/HTTP) check.
func (d *DB) RecordTargetCheck(id int64, ok bool, method, errText string) error {
	if ok {
		_, err := d.Exec(`UPDATE portal_targets SET status = 'verified', method = $2, verified_at = now(),
verified_by = 'auto:' || $2, last_check_at = now(), last_error = '' WHERE id = $1`, id, method)
		return err
	}
	_, err := d.Exec(`UPDATE portal_targets SET last_check_at = now(), last_error = $2 WHERE id = $1`, id, errText)
	return err
}

// SetTargetDecision records an operator's manual approval or rejection.
func (d *DB) SetTargetDecision(id int64, approve bool, note string) error {
	if approve {
		_, err := d.Exec(`UPDATE portal_targets SET status = 'verified', method = 'manual', verified_at = now(),
verified_by = 'operator', note = $2, last_error = '' WHERE id = $1`, id, note)
		return err
	}
	_, err := d.Exec(`UPDATE portal_targets SET status = 'rejected', method = '', verified_at = NULL,
verified_by = 'operator', note = $2 WHERE id = $1`, id, note)
	return err
}

// TargetInActiveRequest reports whether a target belongs to a request that was
// already promoted to an engine task (it cannot be removed then).
func (d *DB) TargetInActiveRequest(targetID int64) (bool, error) {
	var ok bool
	err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM portal_request_targets rt
JOIN portal_requests r ON r.id = rt.request_id
WHERE rt.target_id = $1 AND r.engine_task_id IS NOT NULL AND r.status <> 'done')`, targetID).Scan(&ok)
	return ok, err
}

// DeletePortalTarget removes an org's target.
func (d *DB) DeletePortalTarget(orgID, id int64) error {
	_, err := d.Exec(`DELETE FROM portal_targets WHERE org_id = $1 AND id = $2`, orgID, id)
	return err
}

// CreatePortalRequestWithTargets creates a request and links its targets in one
// transaction. Target ids must already be checked to belong to the org.
func (d *DB) CreatePortalRequestWithTargets(orgID, userID int64, title, scope string, targetIDs []int64) (int64, error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck
	var id int64
	if err := tx.QueryRow(`INSERT INTO portal_requests(org_id, title, scope, created_by) VALUES ($1,$2,$3,$4) RETURNING id`,
		orgID, title, scope, userID).Scan(&id); err != nil {
		return 0, err
	}
	for _, tid := range targetIDs {
		if _, err := tx.Exec(`INSERT INTO portal_request_targets(request_id, target_id)
SELECT $1, id FROM portal_targets WHERE id = $2 AND org_id = $3 ON CONFLICT DO NOTHING`, id, tid, orgID); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// RoE is an accepted Rules of Engagement record.
type RoE struct {
	RequestID        int64     `json:"request_id"`
	AcceptedEmail    string    `json:"accepted_email"`
	SignerName       string    `json:"signer_name"`
	SignerTitle      string    `json:"signer_title"`
	WindowStart      string    `json:"window_start"`
	WindowEnd        string    `json:"window_end"`
	EmergencyContact string    `json:"emergency_contact"`
	Exclusions       string    `json:"exclusions"`
	Version          string    `json:"roe_version"`
	Hash             string    `json:"roe_hash"`
	IP               string    `json:"-"`
	AcceptedAt       time.Time `json:"accepted_at"`
}

// AcceptRoE stores an RoE acceptance. Returns false if one already exists.
func (d *DB) AcceptRoE(orgID, userID int64, r *RoE) (bool, error) {
	res, err := d.Exec(`INSERT INTO portal_roe(request_id, org_id, accepted_by, accepted_email, signer_name, signer_title,
window_start, window_end, emergency_contact, exclusions, roe_version, roe_hash, ip)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (request_id) DO NOTHING`,
		r.RequestID, orgID, userID, r.AcceptedEmail, r.SignerName, r.SignerTitle, r.WindowStart, r.WindowEnd,
		r.EmergencyContact, r.Exclusions, r.Version, r.Hash, r.IP)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// RequestAuth is the authorization status of one request: its targets and RoE.
type RequestAuth struct {
	Targets []*PortalTarget `json:"targets"`
	RoE     *RoE            `json:"roe,omitempty"`
	// Ready is true when there is at least one target, every target is usable
	// and the RoE is accepted with a window that has not ended.
	Ready bool `json:"ready"`
}

// RequestAuthorizations returns the authorization status of the given requests.
func (d *DB) RequestAuthorizations(requestIDs []int64) (map[int64]*RequestAuth, error) {
	out := map[int64]*RequestAuth{}
	if len(requestIDs) == 0 {
		return out, nil
	}
	for _, id := range requestIDs {
		out[id] = &RequestAuth{Targets: []*PortalTarget{}}
	}
	now := time.Now()
	rows, err := d.Query(`SELECT rt.request_id, `+targetCols+` FROM portal_request_targets rt
JOIN portal_targets t ON t.id = rt.target_id JOIN orgs o ON o.id = t.org_id
WHERE rt.request_id = ANY($1) ORDER BY t.value`, requestIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rid int64
		var t PortalTarget
		var verifiedAt, lastCheck sql.NullTime
		if err := rows.Scan(&rid, &t.ID, &t.OrgID, &t.OrgName, &t.Kind, &t.Value, &t.Token, &t.Status, &t.Method,
			&verifiedAt, &t.VerifiedBy, &t.Note, &lastCheck, &t.LastError, &t.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		if verifiedAt.Valid {
			v := verifiedAt.Time
			exp := v.Add(TargetVerificationValidity)
			t.VerifiedAt, t.ExpiresAt = &v, &exp
			if t.Status == "verified" && now.After(exp) {
				t.Status = "expired"
			}
		}
		t.Token = "" // the proof token is only shown on the targets view
		if a := out[rid]; a != nil {
			a.Targets = append(a.Targets, &t)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	roes, err := d.Query(`SELECT request_id, accepted_email, signer_name, signer_title, window_start, window_end,
emergency_contact, exclusions, roe_version, roe_hash, ip, accepted_at FROM portal_roe WHERE request_id = ANY($1)`, requestIDs)
	if err != nil {
		return nil, err
	}
	defer roes.Close()
	for roes.Next() {
		var r RoE
		var ws, we time.Time
		if err := roes.Scan(&r.RequestID, &r.AcceptedEmail, &r.SignerName, &r.SignerTitle, &ws, &we,
			&r.EmergencyContact, &r.Exclusions, &r.Version, &r.Hash, &r.IP, &r.AcceptedAt); err != nil {
			return nil, err
		}
		r.WindowStart, r.WindowEnd = ws.Format(dateLayout), we.Format(dateLayout)
		if a := out[r.RequestID]; a != nil {
			a.RoE = &r
		}
	}
	if err := roes.Err(); err != nil {
		return nil, err
	}

	today := utcToday(now).Format(dateLayout)
	for _, a := range out {
		ready := len(a.Targets) > 0 && a.RoE != nil && a.RoE.WindowEnd >= today
		for _, t := range a.Targets {
			if !t.Usable() {
				ready = false
			}
		}
		a.Ready = ready
	}
	return out, nil
}
