// Audit wiring for the portal/operator domain. s.audit appends a tamper-evident
// entry (best-effort: a failed append is logged but never blocks the action, so a
// DB blip can't take the portal down). Operators read the full chain and verify
// its integrity; an org admin reads only their own org's entries.
package server

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

// audit records one security-relevant action. orgID may be nil for anonymous or
// cross-org operator events.
func (s *Server) audit(r *http.Request, orgID *int64, actor, actorType, action, target, detail string) {
	if s.m == nil || s.m.pg == nil {
		return
	}
	if _, err := s.m.pg.AppendAudit(db.AuditEntry{
		OrgID:     orgID,
		Actor:     actor,
		ActorType: actorType,
		Action:    action,
		Target:    target,
		Detail:    trimLimit(detail, 500),
		IP:        clientIP(r),
	}); err != nil {
		log.Printf("[audit] append failed action=%s: %v", action, err)
	}
}

// GET /api/admin/portal/audit?org_id=&limit= — full audit trail (operator).
func (s *Server) adminListAudit(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var orgID *int64
	if raw := r.URL.Query().Get("org_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil && id > 0 {
			orgID = &id
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := pg.ListAudit(orgID, limit)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la auditoría")
		return
	}
	writeJSON(w, 200, map[string]any{"entries": entries})
}

// GET /api/admin/portal/audit/verify — recompute the whole chain (operator).
func (s *Server) adminVerifyAudit(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	res, err := pg.VerifyAuditChain()
	if err != nil {
		writeErr(w, 500, "no se pudo verificar la auditoría")
		return
	}
	writeJSON(w, 200, res)
}

// auditCategories group the org's audit actions for filtering.
var auditCategories = map[string][]string{
	"access": {"auth.", "password.", "mfa.", "email.", "terms."},
	"team":   {"invite.", "member."},
	"config": {"org.", "policy.", "sso.", "target.", "roe.", "audit."},
	"work":   {"request.", "results.", "remediation.", "report.", "data."},
}

const (
	auditPageSize  = 100
	auditExportMax = 50000
)

// auditFilterFrom reads ?category=&q=&from=&to= (dates YYYY-MM-DD in UTC, both
// inclusive).
func auditFilterFrom(r *http.Request) (db.AuditFilter, error) {
	q := r.URL.Query()
	var f db.AuditFilter
	if c := q.Get("category"); c != "" {
		p, ok := auditCategories[c]
		if !ok {
			return f, errors.New("categoría de auditoría inválida")
		}
		f.Prefixes = p
	}
	f.Query = trimLimit(q.Get("q"), 200)
	day := func(key string) (time.Time, error) {
		v := q.Get(key)
		if v == "" {
			return time.Time{}, nil
		}
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return time.Time{}, errors.New("fecha inválida (AAAA-MM-DD)")
		}
		return t, nil
	}
	var err error
	if f.From, err = day("from"); err != nil {
		return f, err
	}
	if f.To, err = day("to"); err != nil {
		return f, err
	}
	if !f.To.IsZero() {
		f.To = f.To.AddDate(0, 0, 1)
	}
	if !f.From.IsZero() && !f.To.IsZero() && !f.From.Before(f.To) {
		return f, errors.New("rango de fechas inválido")
	}
	return f, nil
}

// GET /api/portal/audit — this org's own audit trail (admin+ and auditors),
// newest first, filtered, a page at a time (?before=<id> for the next page).
func (s *Server) portalListAudit(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	f, err := auditFilterFrom(r)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	f.BeforeID, _ = strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	page := auditPageSize
	if n, _ := strconv.Atoi(r.URL.Query().Get("limit")); n > 0 && n <= 500 {
		page = n
	}
	f.Limit = page + 1
	entries, err := pg.ListOrgAudit(orgID, f)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la auditoría")
		return
	}
	more := len(entries) > page
	if more {
		entries = entries[:page]
	}
	writeJSON(w, 200, map[string]any{"entries": entries, "has_more": more})
}

// GET /api/portal/audit/export — the filtered trail as CSV (at most 50 000
// rows, newest first) with each entry's chain hashes. Exporting is itself
// recorded in the trail.
func (s *Server) portalExportAudit(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	f, err := auditFilterFrom(r)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	f.Limit = auditExportMax + 1
	entries, err := pg.ListOrgAudit(orgID, f)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la auditoría")
		return
	}
	truncated := len(entries) > auditExportMax
	if truncated {
		entries = entries[:auditExportMax]
	}
	q := r.URL.Query()
	s.audit(r, &orgID, s.portalActorEmail(pg, userID), "portal_user", "audit.export", "",
		fmt.Sprintf("rows=%d category=%s from=%s to=%s q=%s", len(entries), q.Get("category"), q.Get("from"), q.Get("to"), f.Query))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="nexus-audit-%d-%s.csv"`, orgID, time.Now().UTC().Format("20060102")))
	w.Header().Set("Cache-Control", "no-store")
	if truncated {
		w.Header().Set("X-Nexus-Truncated", "true")
	}
	_, _ = w.Write([]byte("\xEF\xBB\xBF")) // BOM: spreadsheets then read the file as UTF-8
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "created_at_utc", "action", "actor", "actor_type", "target", "detail", "ip", "prev_hash", "hash"})
	for _, e := range entries {
		_ = cw.Write([]string{
			strconv.FormatInt(e.ID, 10), e.CreatedAt.UTC().Format(time.RFC3339),
			csvSafe(e.Action), csvSafe(e.Actor), csvSafe(e.ActorType), csvSafe(e.Target),
			csvSafe(e.Detail), csvSafe(e.IP), e.PrevHash, e.Hash,
		})
	}
	cw.Flush()
}

// csvSafe keeps a spreadsheet from evaluating a cell as a formula (CSV
// injection): audit text holds user-supplied values such as emails and names,
// so a cell starting with = + - @ or a control character is prefixed with '.
func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r\n", rune(v[0])) {
		return "'" + v
	}
	return v
}
