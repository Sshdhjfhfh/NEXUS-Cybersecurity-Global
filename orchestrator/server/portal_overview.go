package server

import (
	"net/http"
	"time"

	"github.com/nexus-ia/nexus/db"
)

// GET /api/portal/overview — the dashboard summary for the caller's org. Every
// member sees the risk/evaluation overview (computed only from released, client-
// visible results); account-security hygiene and recent activity are added for
// admins and owners, mirroring who can act on them (team + audit).
func (s *Server) portalOverview(w http.ResponseWriter, r *http.Request) {
	orgID, _, role, ok := s.portalRequireRole(w, r, "member")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	ov, err := pg.GetOrgOverview(orgID, time.Now())
	if err != nil {
		writeErr(w, 500, "no se pudo calcular el resumen")
		return
	}
	resp := map[string]any{
		"overview":     ov,
		"generated_at": time.Now().UTC(),
	}
	if roleRank(role) >= roleRank("admin") {
		if acct, err := pg.GetAccountSecurity(orgID); err == nil {
			resp["account_security"] = acct
		}
		if entries, err := pg.ListAudit(&orgID, 8); err == nil {
			resp["activity"] = entries
		} else {
			resp["activity"] = []db.AuditEntry{}
		}
	}
	writeJSON(w, 200, resp)
}

func validViewProfile(v string) bool { return v == "" || v == "executive" || v == "technical" }
func validPortalLang(v string) bool  { return v == "" || v == "es" || v == "en" }

// PATCH /api/portal/me/preferences {view_profile?, lang?, notify_email?} — the
// caller's own preferences. The view profile is purely presentational: it
// changes what is shown first, never what the user may see (that is the role,
// server-side). notify_email opts out of notification mail only; security mail
// (password reset, email verification) is always sent.
func (s *Server) portalSetPreferences(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		ViewProfile *string `json:"view_profile"`
		Lang        *string `json:"lang"`
		NotifyEmail *bool   `json:"notify_email"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if req.ViewProfile != nil && !validViewProfile(*req.ViewProfile) {
		writeErr(w, 400, "perfil de vista inválido")
		return
	}
	if req.Lang != nil && !validPortalLang(*req.Lang) {
		writeErr(w, 400, "idioma no soportado")
		return
	}
	if err := pg.SetPortalUserPrefs(userID, req.ViewProfile, req.Lang, req.NotifyEmail); err != nil {
		writeErr(w, 500, "no se pudieron guardar las preferencias")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
