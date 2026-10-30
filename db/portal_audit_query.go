package db

import (
	"strconv"
	"strings"
	"time"
)

// AuditFilter narrows an organization's audit trail. Zero values mean "any".
type AuditFilter struct {
	Prefixes []string  // action prefixes, e.g. "auth.", "member."
	Query    string    // free text over actor, target, detail and IP
	From, To time.Time // created_at window [From, To)
	BeforeID int64     // keyset paging: entries older than this id
	Limit    int
}

// likeEscape makes user text literal inside a LIKE pattern.
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// ListOrgAudit returns one organization's audit entries, newest first, matching
// the filter. Every value is a bound parameter.
func (d *DB) ListOrgAudit(orgID int64, f AuditFilter) ([]AuditEntry, error) {
	where := []string{"org_id = $1"}
	args := []any{orgID}
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if len(f.Prefixes) > 0 {
		ors := make([]string, 0, len(f.Prefixes))
		for _, p := range f.Prefixes {
			ors = append(ors, "action LIKE "+arg(likeEscape(p)+"%"))
		}
		where = append(where, "("+strings.Join(ors, " OR ")+")")
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		p := arg("%" + likeEscape(q) + "%")
		where = append(where, "(actor ILIKE "+p+" OR target ILIKE "+p+" OR detail ILIKE "+p+" OR ip ILIKE "+p+")")
	}
	if !f.From.IsZero() {
		where = append(where, "created_at >= "+arg(f.From))
	}
	if !f.To.IsZero() {
		where = append(where, "created_at < "+arg(f.To))
	}
	if f.BeforeID > 0 {
		where = append(where, "id < "+arg(f.BeforeID))
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.Query(
		`SELECT id, org_id, actor, actor_type, action, target, detail, ip, prev_hash, hash, created_at
         FROM portal_audit WHERE `+strings.Join(where, " AND ")+` ORDER BY id DESC LIMIT `+arg(limit),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAudit(rows)
}
