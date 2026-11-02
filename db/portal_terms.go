package db

import (
	"database/sql"
	"time"
)

// AcceptPortalTerms records that a user accepted a version of the Terms of
// Service and Privacy Policy, with the time and the address it came from.
func (d *DB) AcceptPortalTerms(userID int64, version, ip string) (time.Time, error) {
	var at time.Time
	err := d.QueryRow(
		`UPDATE portal_users SET terms_version = $2, terms_accepted_at = now(), terms_accepted_ip = $3
         WHERE id = $1 RETURNING terms_accepted_at`, userID, version, ip,
	).Scan(&at)
	return at, err
}

// GetPortalTerms returns the version a user accepted ("" = none) and when.
func (d *DB) GetPortalTerms(userID int64) (version string, acceptedAt *time.Time, err error) {
	var at sql.NullTime
	err = d.QueryRow(
		`SELECT terms_version, terms_accepted_at FROM portal_users WHERE id = $1`, userID,
	).Scan(&version, &at)
	if at.Valid {
		acceptedAt = &at.Time
	}
	return version, acceptedAt, err
}
