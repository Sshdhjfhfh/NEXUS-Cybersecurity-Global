// Multi-tenant client portal. Each organization is isolated: portal JWTs carry
// the org id, and every data access is scoped to the caller's org server-side.
// This is separate from the single-admin console (requireAuth); /api/portal/* do
// their own portal authentication.
package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/nexus-ia/nexus/db"
)

// portalBrute throttles portal logins per email (5 fails / 15 min -> 15 min lock).
var portalBrute = &bruteGuard{
	byIP:     make(map[string]*loginAttempts),
	maxFails: 5,
	window:   15 * time.Minute,
	lockout:  15 * time.Minute,
}

// --- portal JWT (HS256, with org + typ claims) ---

func signPortalJWT(key []byte, userID, orgID int64) (string, error) {
	return signPortalJWTWith(key, userID, orgID, "pwd")
}

// signPortalJWTWith issues a portal session recording how the user signed in
// ("pwd" or "sso"): an SSO session's second factor is the identity provider's.
func signPortalJWTWith(key []byte, userID, orgID int64, amr string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": strconv.FormatInt(userID, 10),
		"org": orgID,
		"typ": "portal",
		"amr": amr,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(jwtTTL).Unix(),
	}).SignedString(key)
}

// portalSessionIsSSO reports whether the request's portal session came from SSO.
func portalSessionIsSSO(r *http.Request, key []byte) bool {
	t, err := jwt.Parse(extractToken(r), func(t *jwt.Token) (any, error) { return key, nil },
		jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !t.Valid {
		return false
	}
	c, _ := t.Claims.(jwt.MapClaims)
	amr, _ := c["amr"].(string)
	return amr == "sso"
}

func parsePortalJWT(tokenStr string, key []byte) (orgID, userID, iat int64, ok bool) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	})
	if err != nil || !t.Valid {
		return 0, 0, 0, false
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return 0, 0, 0, false
	}
	if typ, _ := claims["typ"].(string); typ != "portal" {
		return 0, 0, 0, false // a console token must not be accepted as a portal token
	}
	orgF, _ := claims["org"].(float64)
	uid, _ := strconv.ParseInt(fmt.Sprint(claims["sub"]), 10, 64)
	iatF, _ := claims["iat"].(float64)
	if orgF <= 0 || uid <= 0 {
		return 0, 0, 0, false
	}
	return int64(orgF), uid, int64(iatF), true
}

func (s *Server) portalAuth(w http.ResponseWriter, r *http.Request) (orgID, userID int64, ok bool) {
	orgID, userID, iat, ok := parsePortalJWT(extractToken(r), s.jwtKey)
	if !ok {
		writeErr(w, 401, "no autorizado")
		return 0, 0, false
	}
	// One query resolves everything that can revoke access mid-session: the org
	// being suspended, the user being deactivated, or their sessions being revoked
	// (token issued before sessions_revoked_at). Fails open only on a DB error
	// (any write then fails on the same DB anyway).
	if s.m != nil && s.m.pg != nil && s.m.pg.EnsurePortalTables() == nil {
		a, err := s.m.pg.PortalAuthState(userID)
		if err == nil {
			if a == nil {
				writeErr(w, 401, "no autorizado")
				return 0, 0, false
			}
			if a.Suspended {
				writeErr(w, 403, "organización suspendida")
				return 0, 0, false
			}
			if a.Disabled {
				writeErr(w, 403, "cuenta desactivada")
				return 0, 0, false
			}
			if a.RevokedAt != nil && iat < a.RevokedAt.Unix() {
				writeErr(w, 401, "sesión revocada; inicia sesión de nuevo")
				return 0, 0, false
			}
			// The auditor role is read-only everywhere — enforced here, once, so
			// a write endpoint can never forget it — except for the auditor's own
			// account security.
			if a.Role == roleAuditor && !isReadMethod(r.Method) && !auditorSelfService[r.URL.Path] {
				writeErr(w, 403, "el rol auditor es de solo lectura")
				return 0, 0, false
			}
			if !s.enforceOrgPolicy(w, r, a, userID, iat) {
				return 0, 0, false
			}
		}
	}
	return orgID, userID, true
}

// orgSuspended reports whether an org is suspended. It reads the raw DB handle
// directly (no error written to the client) and fails open on a DB error so a
// transient blip never locks every tenant out; login + this gate still block a
// definitively-suspended org.
func (s *Server) orgSuspended(orgID int64) bool {
	if s.m == nil || s.m.pg == nil {
		return false
	}
	suspended, err := s.m.pg.OrgSuspended(orgID)
	return err == nil && suspended
}

func (s *Server) portalDB(w http.ResponseWriter) *db.DB {
	pg := s.pg(w)
	if pg == nil {
		return nil
	}
	if err := pg.EnsurePortalTables(); err != nil {
		writeErr(w, 503, "almacenamiento del portal no disponible")
		return nil
	}
	return pg
}

// POST /api/portal/register {org_name, email, password}
func (s *Server) portalRegister(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		OrgName  string `json:"org_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.OrgName = strings.TrimSpace(req.OrgName)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.OrgName == "" || len(req.OrgName) > 200 {
		writeErr(w, 400, "nombre de organización inválido")
		return
	}
	if !emailRe.MatchString(req.Email) || len(req.Email) > 200 {
		writeErr(w, 400, "email inválido")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		writeErr(w, 400, "la contraseña debe tener entre 8 y 72 caracteres")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, "no se pudo crear la cuenta")
		return
	}
	orgID, userID, err := pg.CreateOrgWithOwner(req.OrgName, req.Email, string(hash))
	if err != nil {
		if errors.Is(err, db.ErrEmailTaken) {
			writeErr(w, 409, "ese email ya está registrado")
			return
		}
		writeErr(w, 500, "no se pudo crear la cuenta")
		return
	}
	tok, err := signPortalJWT(s.jwtKey, userID, orgID)
	if err != nil {
		writeErr(w, 500, "no se pudo emitir el token")
		return
	}
	s.audit(r, &orgID, req.Email, "portal_user", "org.register",
		"org:"+strconv.FormatInt(orgID, 10), "org="+req.OrgName)
	s.sendVerification(userID, req.Email)
	writeJSON(w, 201, map[string]any{"token": tok, "org": map[string]any{"id": orgID, "name": req.OrgName}})
}

// POST /api/portal/login {email, password}
func (s *Server) portalLogin(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	key := "portal:" + req.Email
	if blocked, retry := portalBrute.blocked(key); blocked {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())+1))
		writeErr(w, 429, "demasiados intentos fallidos; inténtalo más tarde")
		return
	}
	u, err := pg.GetPortalUserByEmail(req.Email)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if u == nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		portalBrute.fail(key)
		s.audit(r, nil, req.Email, "anonymous", "auth.login_failed", "", "")
		writeErr(w, 401, "email o contraseña incorrectos")
		return
	}
	portalBrute.success(key)
	if u.Disabled {
		s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login_blocked", "", "cuenta desactivada")
		writeErr(w, 403, "tu cuenta ha sido desactivada; contacta con un administrador")
		return
	}
	if s.orgSuspended(u.OrgID) {
		s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login_blocked", "", "org suspendida")
		writeErr(w, 403, "organización suspendida; contacta con soporte")
		return
	}
	// SSO enforced: members sign in through the identity provider only. Owners
	// keep their password (and 2FA) as break-glass access if the IdP is down.
	if u.Role != "owner" {
		if on, err := pg.PortalSSOEnforced(u.OrgID); err == nil && on {
			s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login_blocked", "", "sso enforced")
			writeErr(w, 403, "sso_required")
			return
		}
	}
	// Network allowlist, checked before the second factor so codes cannot be
	// tried from a network the org does not allow.
	if p, perr := pg.GetOrgSecurityPolicy(u.OrgID); perr == nil && p != nil {
		if ip := policyClientIP(r); !ipAllowed(ip, p.IPAllowlist) {
			s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.ip_blocked", "user:"+strconv.FormatInt(u.ID, 10), "ip="+ip)
			writeErr(w, 403, "acceso no permitido desde esta red por la política de la organización")
			return
		}
	}
	// Second factor: if the user has 2FA on, a valid TOTP code is required. The
	// "mfa_required" error is the frontend's signal to prompt for the code.
	if u.MFAEnabled {
		secret, derr := s.decryptSecret(u.MFASecret)
		if derr != nil || !verifyPortalTOTPOnce(pg, u.ID, secret, req.Code) {
			s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.mfa_failed", "", "")
			writeErr(w, 401, "mfa_required")
			return
		}
	}
	s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login", "user:"+strconv.FormatInt(u.ID, 10), "")
	tok, err := signPortalJWT(s.jwtKey, u.ID, u.OrgID)
	if err != nil {
		writeErr(w, 500, "no se pudo emitir el token")
		return
	}
	writeJSON(w, 200, map[string]any{"token": tok})
}

// GET /api/portal/me
func (s *Server) portalMe(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
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
	role := ""
	email := ""
	verified := false
	mfaEnabled := false
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		role = u.Role
		email = u.Email
		verified = u.EmailVerified
		mfaEnabled = u.MFAEnabled
	}
	// The org requires 2FA and this member has not enrolled: the portal shows
	// only the enrollment screen (the server refuses everything else anyway).
	requireMFA := false
	if p, err := pg.GetOrgSecurityPolicy(orgID); err == nil && p != nil {
		requireMFA = p.RequireMFA
	}
	sso := portalSessionIsSSO(r, s.jwtKey)
	viewProfile, lang, _ := pg.GetPortalUserPrefs(userID)
	notifyEmail, err := pg.GetPortalUserNotifyEmail(userID)
	if err != nil {
		notifyEmail = true
	}
	writeJSON(w, 200, map[string]any{
		"user": map[string]any{
			"id": userID, "role": role, "email": email, "email_verified": verified,
			"view_profile": viewProfile, "lang": lang, "notify_email": notifyEmail,
			"mfa_enabled": mfaEnabled, "mfa_enrollment_required": requireMFA && !mfaEnabled && !sso,
			"org_requires_mfa": requireMFA, "signed_in_with_sso": sso,
		},
		"org": org,
	})
}

// POST /api/portal/requests {title, scope}
func (s *Server) portalCreateRequest(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Title     string  `json:"title"`
		Scope     string  `json:"scope"` // free-text notes for NEXUS; never put into engine scope
		TargetIDs []int64 `json:"target_ids"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Scope = trimLimit(req.Scope, 4000)
	if req.Title == "" || len(req.Title) > 200 {
		writeErr(w, 400, "título inválido")
		return
	}
	if len(req.TargetIDs) > 100 {
		writeErr(w, 400, "demasiados objetivos en una solicitud")
		return
	}
	// Every target must be this org's own and not rejected (verification may
	// still be pending: the request cannot start until it is verified).
	for _, tid := range req.TargetIDs {
		t, err := pg.GetPortalTarget(orgID, tid)
		if err != nil {
			writeErr(w, 500, "no se pudo crear la solicitud")
			return
		}
		if t == nil || t.Status == "rejected" {
			writeErr(w, 400, "objetivo no válido para esta solicitud")
			return
		}
	}
	id, err := pg.CreatePortalRequestWithTargets(orgID, userID, req.Title, req.Scope, req.TargetIDs)
	if err != nil {
		writeErr(w, 500, "no se pudo crear la solicitud")
		return
	}
	writeJSON(w, 201, map[string]any{"ok": true, "id": id})
}

// GET /api/portal/requests — only this org's requests (tenant isolation).
func (s *Server) portalListRequests(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	list, err := pg.ListPortalRequests(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las solicitudes")
		return
	}
	ids := make([]int64, len(list))
	for i, rq := range list {
		ids[i] = rq.ID
	}
	auths, err := pg.RequestAuthorizations(ids)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las solicitudes")
		return
	}
	type withAuth struct {
		db.PortalRequest
		Authorization *db.RequestAuth `json:"authorization"`
	}
	out := make([]withAuth, len(list))
	for i, rq := range list {
		out[i] = withAuth{rq, auths[rq.ID]}
	}
	writeJSON(w, 200, map[string]any{"requests": out})
}
