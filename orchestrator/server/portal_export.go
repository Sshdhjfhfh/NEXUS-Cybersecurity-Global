// GDPR-style data rights for portal tenants: export (right of access /
// portability) and account deletion (right to erasure). Export is org-scoped and
// curated (no secrets, no internal engine fields). Deletion is irreversible, so
// the owner must re-enter their password; an operator can also delete an org on a
// verified request.
package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// GET /api/portal/export — download this org's data as JSON (admin+).
func (s *Server) portalExport(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}

	org, err := pg.GetOrg(orgID)
	if err != nil || org == nil {
		writeErr(w, 404, "organización no encontrada")
		return
	}
	members, err := pg.ListMembers(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudo exportar")
		return
	}
	reqs, err := pg.ListOrgRequestsForExport(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudo exportar")
		return
	}
	// Per request, include the curated findings the client is entitled to (only
	// when results have been released) — same view as the portal shows.
	type reqOut struct {
		ID              int64  `json:"id"`
		Title           string `json:"title"`
		Scope           string `json:"scope"`
		Status          string `json:"status"`
		ResultsReleased bool   `json:"results_released"`
		CreatedAt       string `json:"created_at"`
		Findings        any    `json:"findings,omitempty"`
	}
	reqsOut := make([]reqOut, 0, len(reqs))
	for _, rq := range reqs {
		ro := reqOut{
			ID: rq.ID, Title: rq.Title, Scope: rq.Scope, Status: rq.Status,
			ResultsReleased: rq.ResultsReleased, CreatedAt: rq.CreatedAt.Format(time.RFC3339),
		}
		if rq.ResultsReleased && rq.EngineTaskID != nil {
			if fs, err := pg.ListClientFindings(*rq.EngineTaskID); err == nil && len(fs) > 0 {
				ro.Findings = fs
			}
		}
		reqsOut = append(reqsOut, ro)
	}
	audit, _ := pg.ListAudit(&orgID, 2000)

	bundle := map[string]any{
		"exported_at":  time.Now().UTC().Format(time.RFC3339),
		"organization": org,
		"members":      members, // Member DTO carries no password hash or secrets
		"requests":     reqsOut,
		"audit":        audit,
	}
	s.audit(r, &orgID, org.Name, "portal_user", "data.export", "org:"+strconv.FormatInt(orgID, 10), "")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"nexus-export-org-"+strconv.FormatInt(orgID, 10)+".json\"")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(bundle)
}

// POST /api/portal/account/delete {password} — the owner permanently deletes their
// organization and all its data (right to erasure). Requires the owner's password.
func (s *Server) portalDeleteAccount(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "owner")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		writeErr(w, 404, "usuario no encontrado")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		writeErr(w, 400, "contraseña incorrecta")
		return
	}
	// Record the erasure in the GLOBAL audit (org_id nil) so a compliance trace
	// survives the deletion without retaining the org's own data.
	s.audit(r, nil, u.Email, "portal_user", "org.delete", "org:"+strconv.FormatInt(orgID, 10), "right-to-erasure")
	if err := pg.DeleteOrg(orgID); err != nil {
		writeErr(w, 500, "no se pudo eliminar la cuenta")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// DELETE /api/admin/portal/orgs/{id} — operator deletes an org (on a verified
// request). Console-only (requireAuth).
func (s *Server) adminDeleteOrg(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	orgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orgID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	s.audit(r, nil, "operator", "operator", "org.delete", "org:"+strconv.FormatInt(orgID, 10), "operator-initiated")
	if err := pg.DeleteOrg(orgID); err != nil {
		writeErr(w, 500, "no se pudo eliminar la organización")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
