package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// OrgSecurityPolicy is an organization's portal access policy.
type OrgSecurityPolicy struct {
	RequireMFA      bool       `json:"require_mfa"`
	IPAllowlist     []string   `json:"ip_allowlist"` // canonical CIDRs; empty = any network
	SessionMaxHours int        `json:"session_max_hours"`
	UpdatedAt       *time.Time `json:"updated_at"`
	UpdatedBy       string     `json:"updated_by"`
}

func splitAllowlist(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// GetOrgSecurityPolicy devuelve la política de seguridad de la org (nil si no existe).
func (d *DB) GetOrgSecurityPolicy(orgID int64) (*OrgSecurityPolicy, error) {
	var p OrgSecurityPolicy
	var allow string
	var upd sql.NullTime
	err := d.QueryRow(
		`SELECT require_mfa, ip_allowlist, session_max_hours, policy_updated_at, policy_updated_by
         FROM orgs WHERE id = $1`, orgID,
	).Scan(&p.RequireMFA, &allow, &p.SessionMaxHours, &upd, &p.UpdatedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IPAllowlist = splitAllowlist(allow)
	if upd.Valid {
		t := upd.Time
		p.UpdatedAt = &t
	}
	return &p, nil
}

// SetOrgSecurityPolicy guarda la política (los CIDR ya validados y canónicos).
func (d *DB) SetOrgSecurityPolicy(orgID int64, requireMFA bool, allowlist []string, sessionMaxHours int, by string) error {
	res, err := d.Exec(
		`UPDATE orgs SET require_mfa = $1, ip_allowlist = $2, session_max_hours = $3,
                policy_updated_at = now(), policy_updated_by = $4
         WHERE id = $5`,
		requireMFA, strings.Join(allowlist, ","), sessionMaxHours, by, orgID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MembersWithoutMFA lista los emails de los miembros activos sin 2FA.
func (d *DB) MembersWithoutMFA(orgID int64) ([]string, error) {
	rows, err := d.Query(
		`SELECT email FROM portal_users
         WHERE org_id = $1 AND NOT disabled AND NOT mfa_enabled ORDER BY email`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
