// Password reset, change-password and email verification for portal accounts.
// Tokens are one-time, hashed at rest, and time-limited; delivery goes through the
// pluggable mailer (portal_mail.go). Forgot-password never reveals whether an
// email exists (anti-enumeration): it always returns ok.
package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	resetTokenTTL  = time.Hour
	verifyTokenTTL = 48 * time.Hour
)

// sendVerification issues an email-verification token and mails the link.
func (s *Server) sendVerification(userID int64, email string) {
	pg := s.m.pg
	if pg == nil {
		return
	}
	tok, err := newInviteToken()
	if err != nil {
		return
	}
	if err := pg.CreatePortalToken(userID, "verify", hashToken(tok), verifyTokenTTL); err != nil {
		return
	}
	link := portalLink("/portal/verify?token=" + tok)
	_ = sendPortalMail(email, "Verifica tu email · NEXUS",
		"Confirma tu dirección de email para tu cuenta de NEXUS:\n\n"+link+
			"\n\nSi no creaste esta cuenta, ignora este mensaje.")
}

// sendPasswordReset issues a reset token and mails the link.
func (s *Server) sendPasswordReset(userID int64, email string) {
	pg := s.m.pg
	if pg == nil {
		return
	}
	tok, err := newInviteToken()
	if err != nil {
		return
	}
	if err := pg.CreatePortalToken(userID, "reset", hashToken(tok), resetTokenTTL); err != nil {
		return
	}
	link := portalLink("/portal/reset?token=" + tok)
	_ = sendPortalMail(email, "Restablece tu contraseña · NEXUS",
		"Has solicitado restablecer tu contraseña de NEXUS. Abre este enlace (válido 1 hora):\n\n"+link+
			"\n\nSi no fuiste tú, ignora este mensaje; tu contraseña no cambiará.")
}

// POST /api/portal/password/forgot {email} — public; always 200 (no enumeration).
func (s *Server) portalForgotPassword(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if u, err := pg.GetPortalUserByEmail(req.Email); err == nil && u != nil {
		s.sendPasswordReset(u.ID, u.Email)
		s.audit(r, &u.OrgID, u.Email, "portal_user", "password.forgot", "", "")
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/password/reset {token, password} — public.
func (s *Server) portalResetPassword(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		writeErr(w, 400, "la contraseña debe tener entre 8 y 72 caracteres")
		return
	}
	userID, ok, err := pg.ConsumePortalToken("reset", hashToken(strings.TrimSpace(req.Token)))
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if !ok {
		writeErr(w, 400, "enlace inválido o expirado")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, "no se pudo actualizar la contraseña")
		return
	}
	if err := pg.SetPortalPassword(userID, string(hash)); err != nil {
		writeErr(w, 500, "no se pudo actualizar la contraseña")
		return
	}
	_ = pg.InvalidatePortalTokens(userID, "reset") // retire any other outstanding links
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		s.audit(r, &u.OrgID, u.Email, "portal_user", "password.reset", "user:"+strconv.FormatInt(userID, 10), "")
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/password/change {old_password, new_password} — authenticated.
func (s *Server) portalChangePassword(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		writeErr(w, 400, "la contraseña debe tener entre 8 y 72 caracteres")
		return
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		writeErr(w, 404, "usuario no encontrado")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)) != nil {
		writeErr(w, 400, "la contraseña actual es incorrecta")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, "no se pudo actualizar la contraseña")
		return
	}
	if err := pg.SetPortalPassword(userID, string(hash)); err != nil {
		writeErr(w, 500, "no se pudo actualizar la contraseña")
		return
	}
	s.audit(r, &orgID, u.Email, "portal_user", "password.change", "user:"+strconv.FormatInt(userID, 10), "")
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/email/verify {token} — public.
func (s *Server) portalVerifyEmail(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	userID, ok, err := pg.ConsumePortalToken("verify", hashToken(strings.TrimSpace(req.Token)))
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if !ok {
		writeErr(w, 400, "enlace inválido o expirado")
		return
	}
	if err := pg.MarkPortalEmailVerified(userID); err != nil {
		writeErr(w, 500, "no se pudo verificar el email")
		return
	}
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		s.audit(r, &u.OrgID, u.Email, "portal_user", "email.verify", "user:"+strconv.FormatInt(userID, 10), "")
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/email/resend — authenticated; resend the verification email.
func (s *Server) portalResendVerification(w http.ResponseWriter, r *http.Request) {
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
	if u.EmailVerified {
		writeJSON(w, 200, map[string]any{"ok": true, "already_verified": true})
		return
	}
	s.sendVerification(u.ID, u.Email)
	writeJSON(w, 200, map[string]any{"ok": true})
}
