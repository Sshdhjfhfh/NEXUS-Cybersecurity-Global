// Per-user TOTP two-factor authentication for portal accounts. Reuses the TOTP
// (RFC 6238) and AES-256-GCM helpers from mfa.go; each portal user has their own
// secret, stored encrypted at rest on portal_users. Enforced at login.
package server

import (
	"net/http"
	"strconv"
	"strings"
)

// GET /api/portal/mfa/status — whether the caller has 2FA enabled.
func (s *Server) portalMFAStatus(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		writeErr(w, 404, "usuario no encontrado")
		return
	}
	writeJSON(w, 200, map[string]any{"enabled": u.MFAEnabled})
}

// POST /api/portal/mfa/setup — generate a new secret (pending until enabled) and
// return it plus the otpauth URI for the authenticator app.
func (s *Server) portalMFASetup(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		writeErr(w, 404, "usuario no encontrado")
		return
	}
	secret := newTOTPSecret()
	enc, err := s.encryptSecret(secret)
	if err != nil {
		writeErr(w, 500, "no se pudo generar el secreto")
		return
	}
	if err := pg.SetPortalMFASecret(userID, enc); err != nil {
		writeErr(w, 500, "no se pudo guardar el secreto")
		return
	}
	writeJSON(w, 200, map[string]any{"secret": secret, "otpauth": otpauthURIFor(u.Email, secret)})
}

// POST /api/portal/mfa/enable {code} — verify the pending secret and turn 2FA on.
func (s *Server) portalMFAEnable(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Code string `json:"code"`
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
	if u.MFASecret == "" {
		writeErr(w, 400, "primero ejecuta la configuración")
		return
	}
	secret, err := s.decryptSecret(u.MFASecret)
	if err != nil {
		writeErr(w, 500, "secreto inválido")
		return
	}
	if !verifyTOTP(secret, strings.TrimSpace(req.Code)) {
		writeErr(w, 400, "código inválido")
		return
	}
	if err := pg.EnablePortalMFA(userID); err != nil {
		writeErr(w, 500, "no se pudo activar")
		return
	}
	s.audit(r, &orgID, u.Email, "portal_user", "mfa.enable", "user:"+strconv.FormatInt(userID, 10), "")
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/mfa/disable {code} — turn 2FA off (requires a valid code).
func (s *Server) portalMFADisable(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Code string `json:"code"`
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
	if p, err := pg.GetOrgSecurityPolicy(orgID); err == nil && p != nil && p.RequireMFA {
		writeErr(w, 403, "la política de la organización exige la verificación en dos pasos")
		return
	}
	if u.MFAEnabled {
		secret, err := s.decryptSecret(u.MFASecret)
		if err != nil || !verifyTOTP(secret, strings.TrimSpace(req.Code)) {
			writeErr(w, 400, "código inválido")
			return
		}
	}
	if err := pg.DisablePortalMFA(userID); err != nil {
		writeErr(w, 500, "no se pudo desactivar")
		return
	}
	s.audit(r, &orgID, u.Email, "portal_user", "mfa.disable", "user:"+strconv.FormatInt(userID, 10), "")
	writeJSON(w, 200, map[string]any{"ok": true})
}
