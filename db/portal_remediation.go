// Client remediation workspace. A client works on the findings NEXUS released to
// them: assigning owners, setting target dates, reporting progress, requesting a
// verification (retest) and — for org admins — formally accepting a risk for a
// bounded period. Real-world rules encoded here:
//   - the engine's finding row is never written from the portal; a finding is
//     only ever "fixed" when a NEXUS retest verifies it (finding_retests);
//   - every item is resolved through the same released + client-visible scope as
//     the results view, so an org can only ever touch its own published findings;
//   - risk acceptance always expires; once expired the finding counts as open again;
//   - the effective state shown to the client is computed in ONE place
//     (effectiveState) and reused by the overview, so numbers never disagree.
package db

import (
	"database/sql"
	"errors"
	"time"
)

// Effective (client-facing) remediation states.
const (
	RemOpen            = "open"
	RemInProgress      = "in_progress"
	RemRetestRequested = "retest_requested"
	RemRetestRunning   = "retest_running"
	RemStillVulnerable = "still_vulnerable"
	RemVerifiedFixed   = "verified_fixed"
	RemRiskAccepted    = "risk_accepted"
)

// RemediationSLADays is NEXUS's default remediation-time policy, counted from the
// detection date. Unrated findings get no SLA rather than a guessed one.
var RemediationSLADays = map[string]int{"critical": 7, "high": 30, "medium": 90, "low": 180}

// ClientRetest is the client-safe view of the latest NEXUS verification: status
// and verdict only (the retester's evidence stays internal).
type ClientRetest struct {
	Status     string     `json:"status"`  // pending | running | completed | failed | stopped
	Verdict    string     `json:"verdict"` // '' | reproduced | fixed | inconclusive
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// RemediationItem is one released finding with its remediation tracking.
type RemediationItem struct {
	FindingID         int64         `json:"finding_id"`
	RequestID         int64         `json:"request_id"`
	RequestTitle      string        `json:"request_title"`
	Name              string        `json:"name"`
	VulnClass         string        `json:"vulnclass"`
	Severity          string        `json:"severity"`
	Summary           string        `json:"summary"`
	DetectedAt        time.Time     `json:"detected_at"`
	State             string        `json:"state"`  // effective, see Rem* constants
	Bucket            string        `json:"bucket"` // open | in_progress | remediated | risk_accepted
	AssigneeID        *int64        `json:"assignee_id"`
	AssigneeEmail     string        `json:"assignee_email,omitempty"`
	DueDate           string        `json:"due_date,omitempty"`   // YYYY-MM-DD (custom or SLA)
	DueSource         string        `json:"due_source,omitempty"` // custom | sla
	SLADays           int           `json:"sla_days,omitempty"`
	Overdue           bool          `json:"overdue"`
	RiskReason        string        `json:"risk_reason,omitempty"`
	RiskExpiresAt     string        `json:"risk_expires_at,omitempty"`
	RiskExpired       bool          `json:"risk_expired,omitempty"`
	RiskAcceptedBy    string        `json:"risk_accepted_by,omitempty"`
	RetestRequestedAt *time.Time    `json:"retest_requested_at,omitempty"`
	RetestNote        string        `json:"retest_note,omitempty"`
	LastRetest        *ClientRetest `json:"last_retest,omitempty"`

	engineStatus string
	clientState  string
}

// RemediationEvent is one entry of a finding's remediation timeline.
type RemediationEvent struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`
	Kind      string    `json:"kind"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

const dateLayout = "2006-01-02"

// utcToday is the calendar day used for due/expiry comparisons.
func utcToday(now time.Time) time.Time {
	u := now.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// effectiveState derives the single client-facing state from the engine status,
// the client's own tracking and the latest NEXUS retest.
func effectiveState(it *RemediationItem, riskExpires *time.Time, now time.Time) {
	today := utcToday(now)
	riskActive := it.clientState == "risk_accepted" && riskExpires != nil && !riskExpires.Before(today)
	it.RiskExpired = it.clientState == "risk_accepted" && !riskActive
	retestActive := it.LastRetest != nil && (it.LastRetest.Status == "pending" || it.LastRetest.Status == "running")
	retestAfterRequest := it.LastRetest != nil && it.RetestRequestedAt != nil && it.LastRetest.FinishedAt != nil &&
		it.LastRetest.FinishedAt.After(*it.RetestRequestedAt)

	switch {
	case it.engineStatus == "fixed" || it.engineStatus == "resolved":
		it.State = RemVerifiedFixed
	case it.engineStatus == "risk_accepted" || riskActive:
		it.State = RemRiskAccepted
	case retestActive:
		it.State = RemRetestRunning
	case it.clientState == "ready_for_retest" && retestAfterRequest && it.LastRetest.Verdict == "reproduced":
		it.State = RemStillVulnerable
	case it.clientState == "ready_for_retest":
		it.State = RemRetestRequested
	case it.clientState == "in_progress" || it.engineStatus == "in_progress":
		it.State = RemInProgress
	default:
		it.State = RemOpen
	}

	switch it.State {
	case RemVerifiedFixed:
		it.Bucket = "remediated"
	case RemRiskAccepted:
		it.Bucket = "risk_accepted"
	case RemInProgress, RemRetestRequested, RemRetestRunning:
		it.Bucket = "in_progress"
	default:
		it.Bucket = "open"
	}

	if it.DueSource == "" {
		if days, ok := RemediationSLADays[it.Severity]; ok {
			it.SLADays = days
			it.DueDate = utcToday(it.DetectedAt).AddDate(0, 0, days).Format(dateLayout)
			it.DueSource = "sla"
		}
	}
	if it.DueDate != "" && (it.Bucket == "open" || it.Bucket == "in_progress") {
		if due, err := time.Parse(dateLayout, it.DueDate); err == nil && due.Before(today) {
			it.Overdue = true
		}
	}
}

const remediationSelect = releasedTasksCTE + remediationBody

// requestTaskCTE scopes to ONE request of the org whether or not its results are
// released — only for the operator's draft report preview ($1 org, $2 request).
const requestTaskCTE = `WITH rel AS (
    SELECT engine_task_id AS task_id, id AS request_id, title
    FROM portal_requests
    WHERE org_id = $1 AND id = $2 AND engine_task_id IS NOT NULL
) `

const remediationBody = `
SELECT f.id, rel.request_id, rel.title, f.name, f.vulnclass, f.severity, f.status, f.summary, f.created_at,
       COALESCE(pr.client_state, 'open'), pr.assignee_id, COALESCE(u.email, ''), pr.due_date,
       COALESCE(pr.risk_reason, ''), pr.risk_expires_at, COALESCE(ru.email, ''),
       pr.retest_requested_at, COALESCE(pr.retest_note, ''),
       lr.status, lr.verdict, lr.created_at, lr.finished_at
FROM findings f
JOIN rel ON f.task_id = rel.task_id
LEFT JOIN portal_remediation pr ON pr.org_id = $1 AND pr.finding_id = f.id
LEFT JOIN portal_users u ON u.id = pr.assignee_id
LEFT JOIN portal_users ru ON ru.id = pr.risk_accepted_by
LEFT JOIN LATERAL (
    SELECT r.status, r.verdict, r.created_at, r.finished_at
    FROM finding_retests r WHERE r.finding_id = f.id ORDER BY r.id DESC LIMIT 1
) lr ON true
WHERE f.status IN ` + clientVisibleStatuses

func scanRemediation(rows interface{ Scan(...any) error }, now time.Time) (*RemediationItem, error) {
	var it RemediationItem
	var assignee sql.NullInt64
	var due, riskExp sql.NullTime
	var reqAt sql.NullTime
	var rStatus, rVerdict sql.NullString
	var rCreated, rFinished sql.NullTime
	if err := rows.Scan(&it.FindingID, &it.RequestID, &it.RequestTitle, &it.Name, &it.VulnClass, &it.Severity,
		&it.engineStatus, &it.Summary, &it.DetectedAt,
		&it.clientState, &assignee, &it.AssigneeEmail, &due,
		&it.RiskReason, &riskExp, &it.RiskAcceptedBy,
		&reqAt, &it.RetestNote,
		&rStatus, &rVerdict, &rCreated, &rFinished); err != nil {
		return nil, err
	}
	if assignee.Valid {
		it.AssigneeID = &assignee.Int64
	}
	if due.Valid {
		it.DueDate = due.Time.Format(dateLayout)
		it.DueSource = "custom"
	}
	var riskExpires *time.Time
	if riskExp.Valid {
		t := riskExp.Time
		riskExpires = &t
		it.RiskExpiresAt = t.Format(dateLayout)
	}
	if reqAt.Valid {
		t := reqAt.Time
		it.RetestRequestedAt = &t
	}
	if rStatus.Valid {
		lr := &ClientRetest{Status: rStatus.String, Verdict: rVerdict.String, CreatedAt: rCreated.Time}
		if rFinished.Valid {
			t := rFinished.Time
			lr.FinishedAt = &t
		}
		it.LastRetest = lr
	}
	if it.clientState != "risk_accepted" {
		it.RiskReason, it.RiskExpiresAt, it.RiskAcceptedBy = "", "", ""
	}
	effectiveState(&it, riskExpires, now)
	return &it, nil
}

// ListRemediation returns every released, client-visible finding of an org with
// its remediation tracking and effective state. Most severe first, then oldest.
func (d *DB) ListRemediation(orgID int64, now time.Time) ([]*RemediationItem, error) {
	rows, err := d.Query(remediationSelect+` ORDER BY `+severityOrder+`, f.created_at ASC LIMIT 5000`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*RemediationItem{}
	for rows.Next() {
		it, err := scanRemediation(rows, now)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ListRequestReportItems returns the client-visible findings of one request (of
// orgID) with their remediation state, plus each finding's Markdown report, for
// the PDF reports. It does not check the release flag: callers decide (clients
// only get released requests; the operator may preview a draft).
func (d *DB) ListRequestReportItems(orgID, requestID int64, now time.Time) ([]*RemediationItem, map[int64]string, error) {
	rows, err := d.Query(requestTaskCTE+remediationBody+` ORDER BY `+severityOrder+`, f.created_at ASC LIMIT 2000`,
		orgID, requestID)
	if err != nil {
		return nil, nil, err
	}
	items := []*RemediationItem{}
	ids := []int64{}
	for rows.Next() {
		it, err := scanRemediation(rows, now)
		if err != nil {
			rows.Close()
			return nil, nil, err
		}
		items = append(items, it)
		ids = append(ids, it.FindingID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	reports := map[int64]string{}
	if len(ids) == 0 {
		return items, reports, nil
	}
	rrows, err := d.Query(`SELECT id, report FROM findings WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, nil, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var id int64
		var rep string
		if err := rrows.Scan(&id, &rep); err != nil {
			return nil, nil, err
		}
		reports[id] = rep
	}
	return items, reports, rrows.Err()
}

// GetRemediation returns one finding's remediation item, or nil when the finding
// is not a released, client-visible finding of this org (callers answer 404).
func (d *DB) GetRemediation(orgID, findingID int64, now time.Time) (*RemediationItem, error) {
	it, err := scanRemediation(d.QueryRow(remediationSelect+` AND f.id = $2`, orgID, findingID), now)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return it, err
}

// GetRemediationReport returns the finding's Markdown report (already scoped by
// the caller through GetRemediation).
func (d *DB) GetRemediationReport(findingID int64) (string, error) {
	var report string
	err := d.QueryRow(`SELECT report FROM findings WHERE id = $1`, findingID).Scan(&report)
	return report, err
}

// ensureRemediation creates the tracking row on first touch.
func (d *DB) ensureRemediation(orgID, findingID int64) error {
	_, err := d.Exec(`INSERT INTO portal_remediation(org_id, finding_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		orgID, findingID)
	return err
}

func (d *DB) updateRemediation(orgID, findingID int64, set string, args ...any) error {
	if err := d.ensureRemediation(orgID, findingID); err != nil {
		return err
	}
	_, err := d.Exec(`UPDATE portal_remediation SET `+set+`, updated_at = now() WHERE org_id = $1 AND finding_id = $2`,
		append([]any{orgID, findingID}, args...)...)
	return err
}

// SetRemediationAssignee sets (or clears, with nil) the owner of a finding.
func (d *DB) SetRemediationAssignee(orgID, findingID int64, assignee *int64) error {
	return d.updateRemediation(orgID, findingID, `assignee_id = $3`, assignee)
}

// SetRemediationDueDate sets (or clears, with nil, falling back to the SLA) the
// target date.
func (d *DB) SetRemediationDueDate(orgID, findingID int64, due *time.Time) error {
	return d.updateRemediation(orgID, findingID, `due_date = $3`, due)
}

// SetRemediationWorkState records the team's own progress ('open'|'in_progress').
// It also clears a pending retest request or risk acceptance it supersedes.
func (d *DB) SetRemediationWorkState(orgID, findingID int64, state string) error {
	return d.updateRemediation(orgID, findingID,
		`client_state = $3, risk_reason = '', risk_expires_at = NULL, risk_accepted_by = NULL, risk_accepted_at = NULL`,
		state)
}

// RequestRemediationRetest marks the finding ready for a NEXUS verification.
func (d *DB) RequestRemediationRetest(orgID, findingID, userID int64, note string) error {
	return d.updateRemediation(orgID, findingID,
		`client_state = 'ready_for_retest', retest_requested_at = now(), retest_requested_by = $3, retest_note = $4`,
		userID, note)
}

// AcceptRemediationRisk records a time-bounded risk acceptance.
func (d *DB) AcceptRemediationRisk(orgID, findingID, userID int64, reason string, expires time.Time) error {
	return d.updateRemediation(orgID, findingID,
		`client_state = 'risk_accepted', risk_reason = $3, risk_expires_at = $4, risk_accepted_by = $5, risk_accepted_at = now()`,
		reason, expires, userID)
}

// AddRemediationEvent appends to a finding's remediation timeline.
func (d *DB) AddRemediationEvent(orgID, findingID int64, actor, kind, detail string) error {
	_, err := d.Exec(`INSERT INTO portal_remediation_events(org_id, finding_id, actor, kind, detail) VALUES ($1,$2,$3,$4,$5)`,
		orgID, findingID, actor, kind, detail)
	return err
}

// ListRemediationEvents returns a finding's timeline, oldest first.
func (d *DB) ListRemediationEvents(orgID, findingID int64) ([]RemediationEvent, error) {
	rows, err := d.Query(`SELECT id, actor, kind, detail, created_at FROM portal_remediation_events
         WHERE org_id = $1 AND finding_id = $2 ORDER BY id ASC LIMIT 500`, orgID, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RemediationEvent{}
	for rows.Next() {
		var e RemediationEvent
		if err := rows.Scan(&e.ID, &e.Actor, &e.Kind, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// OrgRemediationEvent is a timeline entry with its finding, for the org export.
type OrgRemediationEvent struct {
	FindingID int64 `json:"finding_id"`
	RemediationEvent
}

// ListOrgRemediationEvents returns every remediation timeline entry of an org.
func (d *DB) ListOrgRemediationEvents(orgID int64) ([]OrgRemediationEvent, error) {
	rows, err := d.Query(`SELECT finding_id, id, actor, kind, detail, created_at FROM portal_remediation_events
         WHERE org_id = $1 ORDER BY finding_id, id LIMIT 50000`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrgRemediationEvent{}
	for rows.Next() {
		var e OrgRemediationEvent
		if err := rows.Scan(&e.FindingID, &e.ID, &e.Actor, &e.Kind, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ActiveOrgMember reports whether userID is an active member of orgID (valid
// assignee). Auditors are read-only, so they can never own remediation work.
func (d *DB) ActiveOrgMember(orgID, userID int64) (bool, error) {
	var ok bool
	err := d.QueryRow(`SELECT EXISTS(SELECT 1 FROM portal_users
	        WHERE id = $1 AND org_id = $2 AND NOT disabled AND role <> 'auditor')`,
		userID, orgID).Scan(&ok)
	return ok, err
}

// AdminRetestRequest is a pending client verification request, for the operator.
type AdminRetestRequest struct {
	OrgID        int64         `json:"org_id"`
	OrgName      string        `json:"org_name"`
	FindingID    int64         `json:"finding_id"`
	FindingName  string        `json:"finding_name"`
	Severity     string        `json:"severity"`
	RequestedAt  time.Time     `json:"requested_at"`
	RequestedBy  string        `json:"requested_by"`
	Note         string        `json:"note"`
	LatestRetest *ClientRetest `json:"latest_retest,omitempty"`
}

// AdminListRetestRequests lists client verification requests still awaiting a
// NEXUS retest outcome (no retest finished since the request), across orgs.
func (d *DB) AdminListRetestRequests() ([]AdminRetestRequest, error) {
	rows, err := d.Query(`
SELECT pr.org_id, o.name, f.id, COALESCE(NULLIF(f.name, ''), f.vulnclass), f.severity,
       pr.retest_requested_at, COALESCE(u.email, ''), pr.retest_note,
       lr.status, lr.verdict, lr.created_at, lr.finished_at
FROM portal_remediation pr
JOIN orgs o ON o.id = pr.org_id
JOIN findings f ON f.id = pr.finding_id
LEFT JOIN portal_users u ON u.id = pr.retest_requested_by
LEFT JOIN LATERAL (
    SELECT r.status, r.verdict, r.created_at, r.finished_at
    FROM finding_retests r WHERE r.finding_id = f.id ORDER BY r.id DESC LIMIT 1
) lr ON true
WHERE pr.client_state = 'ready_for_retest'
  AND f.status NOT IN ('fixed', 'resolved')
  -- Done for the operator only once a retest after the request reproduced the
  -- issue (a "fixed" verdict flips the finding itself). Inconclusive, failed or
  -- stopped retests keep the request in the queue so it is not silently dropped.
  AND NOT COALESCE(lr.finished_at >= pr.retest_requested_at AND lr.verdict = 'reproduced', false)
ORDER BY pr.retest_requested_at ASC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdminRetestRequest{}
	for rows.Next() {
		var a AdminRetestRequest
		var rStatus, rVerdict sql.NullString
		var rCreated, rFinished sql.NullTime
		if err := rows.Scan(&a.OrgID, &a.OrgName, &a.FindingID, &a.FindingName, &a.Severity,
			&a.RequestedAt, &a.RequestedBy, &a.Note, &rStatus, &rVerdict, &rCreated, &rFinished); err != nil {
			return nil, err
		}
		if rStatus.Valid {
			lr := &ClientRetest{Status: rStatus.String, Verdict: rVerdict.String, CreatedAt: rCreated.Time}
			if rFinished.Valid {
				t := rFinished.Time
				lr.FinishedAt = &t
			}
			a.LatestRetest = lr
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
