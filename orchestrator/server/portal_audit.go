// Audit wiring for the portal/operator domain. s.audit appends a tamper-evident
// entry (best-effort: a failed append is logged but never blocks the action, so a
// DB blip can't take the portal down). Operators read the full chain and verify
// its integrity; an org admin reads only their own org's entries.
package server

import (
	"log"
	"net/http"
	"strconv"

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

// GET /api/portal/audit — this org's own audit trail (admin+ of the org).
func (s *Server) portalListAudit(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := pg.ListAudit(&orgID, limit)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la auditoría")
		return
	}
	writeJSON(w, 200, map[string]any{"entries": entries})
}
