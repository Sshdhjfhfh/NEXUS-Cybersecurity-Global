package db

import (
	"time"
)

// PortalSession is one signed-in device of a portal user.
type PortalSession struct {
	ID         string    `json:"id"`
	AMR        string    `json:"amr"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// sessionMemory is how long ended sessions are kept to recognise known devices.
const sessionMemory = "90 days"

// CreatePortalSession opens a session and reports how the device compares with
// the user's recent history: first is true when the user has no session at all
// in the last 90 days (a brand-new account), known when one of those sessions
// came from the same browser or the same address.
func (d *DB) CreatePortalSession(id string, userID, orgID int64, amr, ip, ua string, expires time.Time) (first, known bool, err error) {
	if _, err = d.Exec(
		`DELETE FROM portal_sessions WHERE user_id = $1 AND expires_at < now() - interval '`+sessionMemory+`'`,
		userID,
	); err != nil {
		return false, false, err
	}
	var n, same int
	if err = d.QueryRow(
		`SELECT COUNT(*), COUNT(*) FILTER (WHERE user_agent = $2 OR ip = $3)
         FROM portal_sessions WHERE user_id = $1 AND created_at > now() - interval '`+sessionMemory+`'`,
		userID, ua, ip,
	).Scan(&n, &same); err != nil {
		return false, false, err
	}
	_, err = d.Exec(
		`INSERT INTO portal_sessions (id, user_id, org_id, amr, ip, user_agent, expires_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, userID, orgID, amr, ip, ua, expires,
	)
	return n == 0, same > 0, err
}

// TouchPortalSession records activity on a session. Callers only invoke it when
// the stored timestamp is stale, so an active session costs one write a minute.
func (d *DB) TouchPortalSession(id, ip string) error {
	_, err := d.Exec(
		`UPDATE portal_sessions SET last_seen_at = now(), ip = $2
         WHERE id = $1 AND revoked_at IS NULL AND last_seen_at < now() - interval '1 minute'`,
		id, ip,
	)
	return err
}

// ListPortalSessions returns a user's open sessions, most recently used first.
func (d *DB) ListPortalSessions(userID int64) ([]PortalSession, error) {
	rows, err := d.Query(
		`SELECT id, amr, ip, user_agent, created_at, last_seen_at, expires_at
         FROM portal_sessions
         WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
         ORDER BY last_seen_at DESC LIMIT 100`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PortalSession{}
	for rows.Next() {
		var s PortalSession
		if err := rows.Scan(&s.ID, &s.AMR, &s.IP, &s.UserAgent, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// RevokePortalSession signs out one of the user's own sessions. It reports
// false when no open session with that id belongs to the user.
func (d *DB) RevokePortalSession(userID int64, id string) (bool, error) {
	res, err := d.Exec(
		`UPDATE portal_sessions SET revoked_at = now()
         WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > now()`,
		id, userID,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// RevokeOtherPortalSessions signs out every session of the user except keep,
// and returns how many were open.
func (d *DB) RevokeOtherPortalSessions(userID int64, keep string) (int64, error) {
	res, err := d.Exec(
		`UPDATE portal_sessions SET revoked_at = now()
         WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL AND expires_at > now()`,
		userID, keep,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (d *DB) endAllPortalSessions(userID int64) error {
	_, err := d.Exec(
		`UPDATE portal_sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID,
	)
	return err
}
