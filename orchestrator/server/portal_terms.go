// Acceptance of the Terms of Service and Privacy Policy. Every account records
// which version it accepted, when and from which address. Each person accepts
// for themselves (an admin cannot accept for the team); until they accept the
// current version the portal only lets them read and accept it, so publishing
// a new version (bump portalTermsVersion) asks everyone again.
package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

// portalTermsVersion names the texts at /legal/terms and /legal/privacy.
// portalTermsDraft says they are still drafts pending legal review (shown as
// such); set it to false and give the approved texts a new version at launch.
const (
	portalTermsVersion = "2026-10-borrador"
	portalTermsDraft   = true
)

var errTermsRequired = "debes aceptar los Términos del Servicio y la Política de Privacidad"
var errTermsChanged = "los términos han cambiado; revisa la versión actual y acéptala de nuevo"

// termsGatePaths are the only calls allowed before accepting the current
// terms: who am I, read and accept them, language, and signing out.
var termsGatePaths = map[string]bool{
	"/api/portal/me":             true,
	"/api/portal/me/preferences": true,
	"/api/portal/terms":          true,
	"/api/portal/terms/accept":   true,
	"/api/portal/logout":         true,
}

// termsGate refuses everything but termsGatePaths to a user who has not
// accepted the current terms (403 "terms_acceptance_required").
func termsGate(w http.ResponseWriter, r *http.Request, accepted string) bool {
	if accepted == portalTermsVersion || termsGatePaths[r.URL.Path] {
		return true
	}
	writeErr(w, 403, "terms_acceptance_required")
	return false
}

// termsConsent validates the consent sent with a sign-up: it must be explicit
// and, when the page says which version it showed, be the current one.
func termsConsent(w http.ResponseWriter, accept bool, version string) bool {
	if !accept {
		writeErr(w, 400, errTermsRequired)
		return false
	}
	if v := strings.TrimSpace(version); v != "" && v != portalTermsVersion {
		writeErr(w, 409, errTermsChanged)
		return false
	}
	return true
}

// recordTermsAcceptance stores and audits an acceptance of the current terms.
func (s *Server) recordTermsAcceptance(r *http.Request, orgID, userID int64, email string) error {
	ip := clientIP(r)
	if _, err := s.m.pg.AcceptPortalTerms(userID, portalTermsVersion, ip); err != nil {
		log.Printf("[portal] recording terms acceptance for user %d failed: %v", userID, err)
		return err
	}
	s.audit(r, &orgID, email, "portal_user", "terms.accept",
		"user:"+strconv.FormatInt(userID, 10), "version="+portalTermsVersion)
	return nil
}

func termsInfo() map[string]any {
	return map[string]any{
		"version": portalTermsVersion,
		"draft":   portalTermsDraft,
		"terms":   "/legal/terms",
		"privacy": "/legal/privacy",
	}
}

// GET /api/portal/terms — public: the current version and where to read it.
func (s *Server) portalTerms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, termsInfo())
}

// POST /api/portal/terms/accept {version} — the signed-in user accepts the
// current terms. The version must be the one they were shown.
func (s *Server) portalAcceptTerms(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Version string `json:"version"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if strings.TrimSpace(req.Version) != portalTermsVersion {
		writeErr(w, 409, errTermsChanged)
		return
	}
	email := ""
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		email = u.Email
	}
	if err := s.recordTermsAcceptance(r, orgID, userID, email); err != nil {
		writeErr(w, 500, "no se pudo registrar la aceptación")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "version": portalTermsVersion})
}
