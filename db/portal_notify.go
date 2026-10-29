// Email notifications for the client portal: who may receive them, a claim
// ledger so each event is mailed at most once, and the outbox the delivery
// worker drains (with retries) so mail never blocks a request.
package db

import (
	"database/sql"
	"time"
)

// NotifyRecipient is a portal user who may receive notification mail: active,
// with a verified address and not opted out.
type NotifyRecipient struct {
	ID    int64
	Email string
	Role  string
	Lang  string
}

const notifyEligible = `NOT disabled AND email_verified AND notify_email`

// NotifyRecipients lists the eligible recipients of an org.
func (d *DB) NotifyRecipients(orgID int64) ([]NotifyRecipient, error) {
	rows, err := d.Query(`SELECT id, email, role, lang FROM portal_users
WHERE org_id = $1 AND `+notifyEligible+` ORDER BY id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NotifyRecipient{}
	for rows.Next() {
		var r NotifyRecipient
		if err := rows.Scan(&r.ID, &r.Email, &r.Role, &r.Lang); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// NotifyRecipientByID returns one user of the org if eligible, else nil.
func (d *DB) NotifyRecipientByID(orgID, userID int64) (*NotifyRecipient, error) {
	var r NotifyRecipient
	err := d.QueryRow(`SELECT id, email, role, lang FROM portal_users
WHERE id = $1 AND org_id = $2 AND `+notifyEligible, userID, orgID).Scan(&r.ID, &r.Email, &r.Role, &r.Lang)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetPortalUserNotifyEmail reports whether the user receives notification mail.
func (d *DB) GetPortalUserNotifyEmail(userID int64) (bool, error) {
	var v bool
	err := d.QueryRow(`SELECT notify_email FROM portal_users WHERE id = $1`, userID).Scan(&v)
	return v, err
}

// ClaimNotification records that (org, kind, ref) is being notified. It returns
// false when it was already claimed, so the caller must not send it again.
func (d *DB) ClaimNotification(orgID int64, kind, ref string) (bool, error) {
	var id int64
	err := d.QueryRow(`INSERT INTO portal_notifications(org_id, kind, ref) VALUES ($1,$2,$3)
ON CONFLICT (org_id, kind, ref) DO NOTHING RETURNING id`, orgID, kind, ref).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// NotifyOrgs lists the non-suspended orgs with at least one released evaluation
// — the only ones that can have remediation events.
func (d *DB) NotifyOrgs() ([]int64, error) {
	rows, err := d.Query(`SELECT o.id FROM orgs o WHERE NOT o.suspended AND EXISTS (
    SELECT 1 FROM portal_requests r WHERE r.org_id = o.id AND r.results_released AND r.engine_task_id IS NOT NULL)
ORDER BY o.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// RemediationActors returns, per finding of the org, who requested the latest
// verification and who accepted the risk (0 when unknown).
func (d *DB) RemediationActors(orgID int64) (requester, accepter map[int64]int64, err error) {
	requester, accepter = map[int64]int64{}, map[int64]int64{}
	rows, err := d.Query(`SELECT finding_id, COALESCE(retest_requested_by, 0), COALESCE(risk_accepted_by, 0)
FROM portal_remediation WHERE org_id = $1`, orgID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var fid, req, acc int64
		if err := rows.Scan(&fid, &req, &acc); err != nil {
			return nil, nil, err
		}
		requester[fid], accepter[fid] = req, acc
	}
	return requester, accepter, rows.Err()
}

// OutboxMail is one queued notification email.
type OutboxMail struct {
	ID        int64      `json:"id"`
	OrgID     *int64     `json:"org_id"`
	OrgName   string     `json:"org_name"`
	Kind      string     `json:"kind"`
	To        string     `json:"to"`
	Subject   string     `json:"subject"`
	Status    string     `json:"status"`
	Attempts  int        `json:"attempts"`
	LastError string     `json:"last_error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	Text      string     `json:"-"`
	HTML      string     `json:"-"`
}

// EnqueueMail queues a notification email for the delivery worker.
func (d *DB) EnqueueMail(orgID *int64, kind, to, subject, text, html string) error {
	_, err := d.Exec(`INSERT INTO portal_mail_outbox(org_id, kind, to_addr, subject, text_body, html_body)
VALUES ($1,$2,$3,$4,$5,$6)`, orgID, kind, to, subject, text, html)
	return err
}

// DueMail returns pending mail whose next attempt is due, oldest first.
func (d *DB) DueMail(limit int) ([]*OutboxMail, error) {
	rows, err := d.Query(`SELECT id, org_id, kind, to_addr, subject, text_body, html_body, attempts
FROM portal_mail_outbox WHERE status = 'pending' AND next_at <= now() ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*OutboxMail{}
	for rows.Next() {
		m := &OutboxMail{Status: "pending"}
		var org sql.NullInt64
		if err := rows.Scan(&m.ID, &org, &m.Kind, &m.To, &m.Subject, &m.Text, &m.HTML, &m.Attempts); err != nil {
			return nil, err
		}
		if org.Valid {
			m.OrgID = &org.Int64
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MarkMailSent records a successful delivery.
func (d *DB) MarkMailSent(id int64) error {
	_, err := d.Exec(`UPDATE portal_mail_outbox SET status = 'sent', attempts = attempts + 1,
last_error = '', sent_at = now() WHERE id = $1`, id)
	return err
}

// MarkMailFailed records a failed attempt: retried at next, or given up (final).
func (d *DB) MarkMailFailed(id int64, errMsg string, next time.Time, final bool) error {
	status := "pending"
	if final {
		status = "failed"
	}
	_, err := d.Exec(`UPDATE portal_mail_outbox SET status = $2, attempts = attempts + 1, last_error = $3,
next_at = $4 WHERE id = $1`, id, status, errMsg, next)
	return err
}

// PurgeOldMail deletes delivered or abandoned mail older than the cutoff.
func (d *DB) PurgeOldMail(before time.Time) (int64, error) {
	res, err := d.Exec(`DELETE FROM portal_mail_outbox WHERE status IN ('sent','failed') AND created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// AdminListOutbox lists recent outbox mail (newest first, no bodies) for the
// operator.
func (d *DB) AdminListOutbox(limit int) ([]*OutboxMail, error) {
	rows, err := d.Query(`SELECT m.id, m.org_id, COALESCE(o.name, ''), m.kind, m.to_addr, m.subject, m.status,
       m.attempts, m.last_error, m.created_at, m.sent_at
FROM portal_mail_outbox m LEFT JOIN orgs o ON o.id = m.org_id ORDER BY m.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*OutboxMail{}
	for rows.Next() {
		m := &OutboxMail{}
		var org sql.NullInt64
		var sent sql.NullTime
		if err := rows.Scan(&m.ID, &org, &m.OrgName, &m.Kind, &m.To, &m.Subject, &m.Status,
			&m.Attempts, &m.LastError, &m.CreatedAt, &sent); err != nil {
			return nil, err
		}
		if org.Valid {
			m.OrgID = &org.Int64
		}
		if sent.Valid {
			t := sent.Time
			m.SentAt = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetOutboxMail returns one outbox mail with its bodies (nil if missing).
func (d *DB) GetOutboxMail(id int64) (*OutboxMail, error) {
	m := &OutboxMail{}
	var org sql.NullInt64
	err := d.QueryRow(`SELECT id, org_id, kind, to_addr, subject, status, text_body, html_body
FROM portal_mail_outbox WHERE id = $1`, id).Scan(&m.ID, &org, &m.Kind, &m.To, &m.Subject, &m.Status, &m.Text, &m.HTML)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if org.Valid {
		m.OrgID = &org.Int64
	}
	return m, nil
}
