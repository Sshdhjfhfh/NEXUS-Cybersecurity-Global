// Organization overview for the client portal dashboard. Every number here is
// computed from the org's own data and follows the same client-safety rules as
// the results view (portal_results.go): only findings of requests whose results
// an operator RELEASED, and only client-visible statuses. Nothing is estimated
// or synthesized — an org with no released results gets zeros, and the UI shows
// an empty state instead of a made-up score.
package db

import (
	"time"
)

// SeverityStatusCounts buckets a severity's client-visible findings by where they
// are in the remediation lifecycle.
type SeverityStatusCounts struct {
	Open         int `json:"open"`          // confirmed, nobody working on it yet
	InProgress   int `json:"in_progress"`   // remediation under way
	Remediated   int `json:"remediated"`    // resolved / fixed
	RiskAccepted int `json:"risk_accepted"` // consciously accepted by the client
}

// OverviewFinding is a prioritized open finding, with the request it belongs to
// so the UI can drill down into that evaluation's results.
type OverviewFinding struct {
	ID           int64     `json:"id"`
	RequestID    int64     `json:"request_id"`
	RequestTitle string    `json:"request_title"`
	Name         string    `json:"name"`
	VulnClass    string    `json:"vulnclass"`
	Severity     string    `json:"severity"`
	Status       string    `json:"status"` // effective remediation state
	CreatedAt    time.Time `json:"created_at"`
	DueDate      string    `json:"due_date,omitempty"`
	Overdue      bool      `json:"overdue"`
}

// MonthCount is the number of published findings detected in a calendar month.
type MonthCount struct {
	Month string `json:"month"` // YYYY-MM
	Count int    `json:"count"`
}

// OrgOverview is the dashboard payload shared by every portal role.
type OrgOverview struct {
	Requests         map[string]int                   `json:"requests"` // by status
	RequestsTotal    int                              `json:"requests_total"`
	ReleasedRequests int                              `json:"released_requests"`
	BySeverity       map[string]*SeverityStatusCounts `json:"by_severity"`
	TopOpen          []OverviewFinding                `json:"top_open"`
	Overdue          int                              `json:"overdue"` // open items past their target date
	Monthly          []MonthCount                     `json:"monthly"`
}

// AccountSecurity summarizes the org's account hygiene (admin view).
type AccountSecurity struct {
	ActiveMembers  int `json:"active_members"`
	MFAEnabled     int `json:"mfa_enabled"` // among active members
	Unverified     int `json:"unverified"`  // active members with unverified email
	Disabled       int `json:"disabled"`
	PendingInvites int `json:"pending_invites"`
}

// releasedTasksCTE resolves the engine tasks whose results this org may see. One
// task could in theory back two requests; DISTINCT ON keeps each finding counted
// once (attributed to the oldest request).
const releasedTasksCTE = `WITH rel AS (
    SELECT DISTINCT ON (engine_task_id) engine_task_id AS task_id, id AS request_id, title
    FROM portal_requests
    WHERE org_id = $1 AND results_released AND engine_task_id IS NOT NULL
    ORDER BY engine_task_id, id
) `

var overviewSeverities = []string{"critical", "high", "medium", "low"}

// overviewMonths is how many calendar months the detection trend covers.
const overviewMonths = 6

// GetOrgOverview computes the dashboard overview for one org.
func (d *DB) GetOrgOverview(orgID int64, now time.Time) (*OrgOverview, error) {
	ov := &OrgOverview{
		Requests:   map[string]int{"new": 0, "scheduled": 0, "in_progress": 0, "done": 0},
		BySeverity: map[string]*SeverityStatusCounts{},
		TopOpen:    []OverviewFinding{},
	}
	for _, s := range overviewSeverities {
		ov.BySeverity[s] = &SeverityStatusCounts{}
	}

	// Evaluations by status.
	rows, err := d.Query(
		`SELECT status, count(*), count(*) FILTER (WHERE results_released)
         FROM portal_requests WHERE org_id = $1 GROUP BY status`, orgID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var n, released int
		if err := rows.Scan(&status, &n, &released); err != nil {
			rows.Close()
			return nil, err
		}
		ov.Requests[status] += n
		ov.RequestsTotal += n
		ov.ReleasedRequests += released
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Findings by severity × lifecycle bucket, from the SAME effective state the
	// remediation workspace shows (client progress, retests, risk acceptance and
	// its expiry included), so the two views can never disagree.
	items, err := d.ListRemediation(orgID, now)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		b, ok := ov.BySeverity[it.Severity]
		if !ok {
			// Findings without a recognized severity are still real: count them in
			// their own "unrated" bucket instead of guessing a severity or
			// silently dropping them from the totals.
			if ov.BySeverity["unrated"] == nil {
				ov.BySeverity["unrated"] = &SeverityStatusCounts{}
			}
			b = ov.BySeverity["unrated"]
		}
		switch it.Bucket {
		case "open":
			b.Open++
		case "in_progress":
			b.InProgress++
		case "remediated":
			b.Remediated++
		case "risk_accepted":
			b.RiskAccepted++
		}
		// Highest-priority open findings: items are already most severe first,
		// then oldest first (the longer a serious issue stays open, the more
		// urgent it is).
		if (it.Bucket == "open" || it.Bucket == "in_progress") && len(ov.TopOpen) < 10 {
			ov.TopOpen = append(ov.TopOpen, OverviewFinding{
				ID: it.FindingID, RequestID: it.RequestID, RequestTitle: it.RequestTitle, Name: it.Name,
				VulnClass: it.VulnClass, Severity: it.Severity, Status: it.State, CreatedAt: it.DetectedAt,
				DueDate: it.DueDate, Overdue: it.Overdue,
			})
		}
		if it.Overdue {
			ov.Overdue++
		}
	}

	// Detection trend: published findings per calendar month (UTC), zero-filled so
	// the chart never implies a gap is missing data.
	start := time.Date(now.UTC().Year(), now.UTC().Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -(overviewMonths - 1), 0)
	counts := map[string]int{}
	rows, err = d.Query(releasedTasksCTE+
		`SELECT to_char(date_trunc('month', f.created_at AT TIME ZONE 'UTC'), 'YYYY-MM'), count(*)
         FROM findings f JOIN rel ON f.task_id = rel.task_id
         WHERE f.status IN `+clientVisibleStatuses+` AND f.created_at >= $2
         GROUP BY 1`, orgID, start)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m string
		var n int
		if err := rows.Scan(&m, &n); err != nil {
			rows.Close()
			return nil, err
		}
		counts[m] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := 0; i < overviewMonths; i++ {
		m := start.AddDate(0, i, 0).Format("2006-01")
		ov.Monthly = append(ov.Monthly, MonthCount{Month: m, Count: counts[m]})
	}
	return ov, nil
}

// GetAccountSecurity summarizes member account hygiene for an org.
func (d *DB) GetAccountSecurity(orgID int64) (*AccountSecurity, error) {
	var a AccountSecurity
	err := d.QueryRow(
		`SELECT count(*) FILTER (WHERE NOT disabled),
                count(*) FILTER (WHERE NOT disabled AND mfa_enabled),
                count(*) FILTER (WHERE NOT disabled AND NOT email_verified),
                count(*) FILTER (WHERE disabled)
         FROM portal_users WHERE org_id = $1`, orgID,
	).Scan(&a.ActiveMembers, &a.MFAEnabled, &a.Unverified, &a.Disabled)
	if err != nil {
		return nil, err
	}
	if err := d.QueryRow(
		`SELECT count(*) FROM portal_invites WHERE org_id = $1 AND NOT accepted`, orgID,
	).Scan(&a.PendingInvites); err != nil {
		return nil, err
	}
	return &a, nil
}

// GetPortalUserPrefs returns a user's UI preferences.
func (d *DB) GetPortalUserPrefs(userID int64) (viewProfile, lang string, err error) {
	err = d.QueryRow(`SELECT view_profile, lang FROM portal_users WHERE id = $1`, userID).Scan(&viewProfile, &lang)
	return viewProfile, lang, err
}

// SetPortalUserPrefs updates a user's UI preferences. Values are validated by the
// caller; nil leaves a field unchanged.
func (d *DB) SetPortalUserPrefs(userID int64, viewProfile, lang *string) error {
	_, err := d.Exec(
		`UPDATE portal_users SET view_profile = COALESCE($2, view_profile), lang = COALESCE($3, lang) WHERE id = $1`,
		userID, viewProfile, lang,
	)
	return err
}
