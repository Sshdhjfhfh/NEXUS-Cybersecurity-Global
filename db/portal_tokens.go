// One-time tokens for password reset and email verification. Only the hash of a
// token is persisted; validating consumes it (single use) inside a transaction.
package db

import (
	"database/sql"
	"time"
)

// CreatePortalToken stores a hashed one-time token of the given kind.
func (d *DB) CreatePortalToken(userID int64, kind, tokenHash string, ttl time.Duration) error {
	_, err := d.Exec(
		`INSERT INTO portal_tokens(user_id, kind, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		userID, kind, tokenHash, time.Now().UTC().Add(ttl),
	)
	return err
}

// PeekPortalToken returns the user a still-valid token belongs to without using
// it up, so a request can be validated before the token is spent.
func (d *DB) PeekPortalToken(kind, tokenHash string) (userID int64, ok bool, err error) {
	err = d.QueryRow(
		`SELECT user_id FROM portal_tokens
         WHERE token_hash = $1 AND kind = $2 AND used = false AND expires_at > now()`, tokenHash, kind,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return userID, err == nil, err
}

// ConsumePortalToken validates a token of the given kind and marks it used in one
// transaction (so it cannot be replayed). Returns the owning user id. ok is false
// when the token is unknown, already used, or expired.
func (d *DB) ConsumePortalToken(kind, tokenHash string) (userID int64, ok bool, err error) {
	tx, err := d.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback() //nolint:errcheck

	var id int64
	err = tx.QueryRow(
		`SELECT id, user_id FROM portal_tokens
         WHERE token_hash = $1 AND kind = $2 AND used = false AND expires_at > now()
         FOR UPDATE`, tokenHash, kind,
	).Scan(&id, &userID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if _, err = tx.Exec(`UPDATE portal_tokens SET used = true WHERE id = $1`, id); err != nil {
		return 0, false, err
	}
	return userID, true, tx.Commit()
}

// SetPortalPassword updates a user's password hash.
func (d *DB) SetPortalPassword(userID int64, passwordHash string) error {
	res, err := d.Exec(`UPDATE portal_users SET password_hash = $1 WHERE id = $2`, passwordHash, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkPortalEmailVerified flags a user's email as verified.
func (d *DB) MarkPortalEmailVerified(userID int64) error {
	_, err := d.Exec(`UPDATE portal_users SET email_verified = true WHERE id = $1`, userID)
	return err
}

// InvalidatePortalTokens marks every still-valid token of a kind for a user as
// used (e.g. after a successful password reset, to retire other outstanding
// reset links).
func (d *DB) InvalidatePortalTokens(userID int64, kind string) error {
	_, err := d.Exec(
		`UPDATE portal_tokens SET used = true WHERE user_id = $1 AND kind = $2 AND used = false`, userID, kind,
	)
	return err
}

// PortalHousekeeping removes identity records that can no longer be used:
// spent or expired one-time tokens (after a week), sessions that expired more
// than 90 days ago (kept until then to recognise known devices), and pending
// invitations 30 days past their expiry. Returns rows removed per kind.
func (d *DB) PortalHousekeeping() (tokens, sessions, invites int64, err error) {
	n := func(q string, args ...any) int64 {
		if err != nil {
			return 0
		}
		res, e := d.Exec(q, args...)
		if e != nil {
			err = e
			return 0
		}
		c, _ := res.RowsAffected()
		return c
	}
	tokens = n(`DELETE FROM portal_tokens WHERE (used OR expires_at < now()) AND created_at < now() - interval '7 days'`)
	sessions = n(`DELETE FROM portal_sessions WHERE expires_at < now() - interval '` + sessionMemory + `'`)
	invites = n(`DELETE FROM portal_invites WHERE accepted = false AND created_at < $1`,
		time.Now().Add(-InviteTTL-30*24*time.Hour))
	return tokens, sessions, invites, err
}
