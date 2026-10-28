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
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": strconv.FormatInt(userID, 10),
		"org": orgID,
		"typ": "portal",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(jwtTTL).Unix(),
	}).SignedString(key)
}

func parsePortalJWT(tokenStr string, key []byte) (orgID, userID int64, ok bool) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	})
	if err != nil || !t.Valid {
		return 0, 0, false
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return 0, 0, false
	}
	if typ, _ := claims["typ"].(string); typ != "portal" {
		return 0, 0, false // a console token must not be accepted as a portal token
	}
	orgF, _ := claims["org"].(float64)
	uid, _ := strconv.ParseInt(fmt.Sprint(claims["sub"]), 10, 64)
	if orgF <= 0 || uid <= 0 {
		return 0, 0, false
	}
	return int64(orgF), uid, true
}

func (s *Server) portalAuth(w http.ResponseWriter, r *http.Request) (orgID, userID int64, ok bool) {
	orgID, userID, ok = parsePortalJWT(extractToken(r), s.jwtKey)
	if !ok {
		writeErr(w, 401, "no autorizado")
		return 0, 0, false
	}
	// A suspended organization is locked out immediately, even with a valid token.
	if s.orgSuspended(orgID) {
		writeErr(w, 403, "organización suspendida")
		return 0, 0, false
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
	if s.orgSuspended(u.OrgID) {
		s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login_blocked", "", "org suspendida")
		writeErr(w, 403, "organización suspendida; contacta con soporte")
		return
	}
	// Second factor: if the user has 2FA on, a valid TOTP code is required. The
	// "mfa_required" error is the frontend's signal to prompt for the code.
	if u.MFAEnabled {
		secret, derr := s.decryptSecret(u.MFASecret)
		if derr != nil || !verifyTOTP(secret, strings.TrimSpace(req.Code)) {
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
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		role = u.Role
		email = u.Email
	}
	writeJSON(w, 200, map[string]any{
		"user": map[string]any{"id": userID, "role": role, "email": email},
		"org":  org,
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
		Title string `json:"title"`
		Scope string `json:"scope"`
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
	id, err := pg.CreatePortalRequest(orgID, userID, req.Title, req.Scope)
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
	writeJSON(w, 200, map[string]any{"requests": list})
}
