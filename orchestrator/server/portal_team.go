// Portal team management: members, roles (RBAC) and invitations. Roles form a
// strict hierarchy owner > admin > member, plus "auditor": read access to what
// an admin can see (team, audit log, export) and no write access at all beyond
// their own account security. The caller's role is ALWAYS resolved
// from the database per request (never trusted from the JWT, which could be
// stale after a role change), and every read/write is scoped to the caller's org
// so one tenant can never touch another tenant's team.
package server

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/nexus-ia/nexus/db"
)

// roleRank orders the portal roles; a higher rank implies every lower privilege.
func roleRank(role string) int {
	switch role {
	case "owner":
		return 3
	case "admin":
		return 2
	case "member":
		return 1
	default:
		return 0
	}
}

// roleAuditor sits outside the rank ladder (rank 0): it may READ anything gated
// at admin or below, and write nothing.
const roleAuditor = "auditor"

func validInviteRole(role string) bool {
	return role == "admin" || role == "member" || role == roleAuditor
}
func validRole(role string) bool { return roleRank(role) > 0 || role == roleAuditor }

// canViewOrgAdmin reports whether a role may see admin-level org data
// (account security, activity, team, audit log).
func canViewOrgAdmin(role string) bool {
	return roleRank(role) >= roleRank("admin") || role == roleAuditor
}

func isReadMethod(m string) bool { return m == http.MethodGet || m == http.MethodHead }

// auditorSelfService are the only writes an auditor may make: their own
// preferences and account security.
var auditorSelfService = map[string]bool{
	"/api/portal/me/preferences":  true,
	"/api/portal/mfa/setup":       true,
	"/api/portal/mfa/enable":      true,
	"/api/portal/mfa/disable":     true,
	"/api/portal/password/change": true,
	"/api/portal/email/resend":    true,
}

// portalRequireRole authenticates the portal caller and resolves their current
// role from the DB, enforcing a minimum. Returns the org, user id and live role.
// An auditor passes read requests gated at admin or below and nothing else.
func (s *Server) portalRequireRole(w http.ResponseWriter, r *http.Request, minRole string) (orgID, userID int64, role string, ok bool) {
	orgID, userID, ok = s.portalAuth(w, r)
	if !ok {
		return 0, 0, "", false
	}
	pg := s.portalDB(w)
	if pg == nil {
		return 0, 0, "", false
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return 0, 0, "", false
	}
	if u == nil || u.OrgID != orgID {
		writeErr(w, 401, "no autorizado")
		return 0, 0, "", false
	}
	if u.Role == roleAuditor {
		if !isReadMethod(r.Method) || roleRank(minRole) > roleRank("admin") {
			writeErr(w, 403, "no tienes permisos para esta acción")
			return 0, 0, "", false
		}
		return orgID, userID, u.Role, true
	}
	if roleRank(u.Role) < roleRank(minRole) {
		writeErr(w, 403, "no tienes permisos para esta acción")
		return 0, 0, "", false
	}
	return orgID, userID, u.Role, true
}

func newInviteToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GET /api/portal/members — list this org's members (admin+).
func (s *Server) portalListMembers(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	members, err := pg.ListMembers(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar los miembros")
		return
	}
	writeJSON(w, 200, map[string]any{"members": members})
}

// PATCH /api/portal/members/{id} {role} — change a member's role (admin+).
// Guards against leaving the organization without an owner.
func (s *Server) portalUpdateMember(w http.ResponseWriter, r *http.Request) {
	orgID, actorID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Role = strings.TrimSpace(req.Role)
	if !validRole(req.Role) {
		writeErr(w, 400, "rol inválido")
		return
	}
	target, err := pg.GetPortalUserByID(targetID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if target == nil || target.OrgID != orgID {
		writeErr(w, 404, "miembro no encontrado")
		return
	}
	// Demoting the last owner would orphan the organization.
	if target.Role == "owner" && req.Role != "owner" {
		owners, err := pg.CountOwners(orgID)
		if err != nil {
			writeErr(w, 503, "almacenamiento no disponible")
			return
		}
		if owners <= 1 {
			writeErr(w, 409, "la organización debe conservar al menos un propietario")
			return
		}
	}
	if err := pg.UpdateMemberRole(orgID, targetID, req.Role); err != nil {
		writeErr(w, 500, "no se pudo actualizar el rol")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(actorID, 10), "portal_user", "member.role_change",
		"user:"+strconv.FormatInt(targetID, 10), target.Role+"->"+req.Role)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// DELETE /api/portal/members/{id} — remove a member (admin+). Cannot remove the
// last owner.
func (s *Server) portalDeleteMember(w http.ResponseWriter, r *http.Request) {
	orgID, actorID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	target, err := pg.GetPortalUserByID(targetID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if target == nil || target.OrgID != orgID {
		writeErr(w, 404, "miembro no encontrado")
		return
	}
	if target.Role == "owner" {
		owners, err := pg.CountOwners(orgID)
		if err != nil {
			writeErr(w, 503, "almacenamiento no disponible")
			return
		}
		if owners <= 1 {
			writeErr(w, 409, "la organización debe conservar al menos un propietario")
			return
		}
	}
	if err := pg.DeleteMember(orgID, targetID); err != nil {
		writeErr(w, 500, "no se pudo eliminar el miembro")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(actorID, 10), "portal_user", "member.remove",
		"user:"+strconv.FormatInt(targetID, 10), "email="+target.Email)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/members/{id}/disable {disabled} — deactivate or reactivate a
// member (admin+). A disabled user is blocked at login and on every request.
// Guards: you cannot disable yourself, nor the last active owner.
func (s *Server) portalSetMemberDisabled(w http.ResponseWriter, r *http.Request) {
	orgID, actorID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if req.Disabled && targetID == actorID {
		writeErr(w, 400, "no puedes desactivar tu propia cuenta")
		return
	}
	target, err := pg.GetPortalUserByID(targetID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if target == nil || target.OrgID != orgID {
		writeErr(w, 404, "miembro no encontrado")
		return
	}
	if req.Disabled && target.Role == "owner" {
		owners, err := pg.CountActiveOwners(orgID)
		if err != nil {
			writeErr(w, 503, "almacenamiento no disponible")
			return
		}
		if owners <= 1 {
			writeErr(w, 409, "la organización debe conservar al menos un propietario activo")
			return
		}
	}
	if err := pg.SetPortalUserDisabled(orgID, targetID, req.Disabled); err != nil {
		writeErr(w, 500, "no se pudo actualizar la cuenta")
		return
	}
	action := "member.enable"
	if req.Disabled {
		action = "member.disable"
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(actorID, 10), "portal_user", action,
		"user:"+strconv.FormatInt(targetID, 10), "email="+target.Email)
	writeJSON(w, 200, map[string]any{"ok": true, "disabled": req.Disabled})
}

// POST /api/portal/members/{id}/revoke-sessions — invalidate a member's active
// tokens (admin+), forcing them to sign in again.
func (s *Server) portalRevokeMemberSessions(w http.ResponseWriter, r *http.Request) {
	orgID, actorID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	target, err := pg.GetPortalUserByID(targetID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if target == nil || target.OrgID != orgID {
		writeErr(w, 404, "miembro no encontrado")
		return
	}
	if err := pg.RevokePortalUserSessions(orgID, targetID); err != nil {
		writeErr(w, 500, "no se pudieron revocar las sesiones")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(actorID, 10), "portal_user", "member.revoke_sessions",
		"user:"+strconv.FormatInt(targetID, 10), "email="+target.Email)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/invites {email, role} — create an invitation (admin+).
// Returns a one-time token the admin shares with the invitee (no email yet).
func (s *Server) portalCreateInvite(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Role = strings.TrimSpace(req.Role)
	if req.Role == "" {
		req.Role = "member"
	}
	if !emailRe.MatchString(req.Email) || len(req.Email) > 200 {
		writeErr(w, 400, "email inválido")
		return
	}
	if !validInviteRole(req.Role) {
		writeErr(w, 400, "rol inválido (admin, member o auditor)")
		return
	}
	// Reject inviting someone who already has an account anywhere.
	existing, err := pg.GetPortalUserByEmail(req.Email)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if existing != nil {
		writeErr(w, 409, "ese email ya tiene una cuenta")
		return
	}
	token, err := newInviteToken()
	if err != nil {
		writeErr(w, 500, "no se pudo generar la invitación")
		return
	}
	id, err := pg.CreateInvite(orgID, userID, req.Email, req.Role, token)
	if err != nil {
		writeErr(w, 500, "no se pudo crear la invitación")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(userID, 10), "portal_user", "invite.create",
		"invite:"+strconv.FormatInt(id, 10), "email="+req.Email+" role="+req.Role)
	writeJSON(w, 201, map[string]any{
		"ok":    true,
		"id":    id,
		"token": token,
		"email": req.Email,
		"role":  req.Role,
	})
}

// GET /api/portal/invites — list this org's pending invitations (admin+).
func (s *Server) portalListInvites(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	invites, err := pg.ListInvites(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las invitaciones")
		return
	}
	writeJSON(w, 200, map[string]any{"invites": invites})
}

// DELETE /api/portal/invites/{id} — revoke a pending invitation (admin+).
func (s *Server) portalDeleteInvite(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	if err := pg.DeleteInvite(orgID, id); err != nil {
		writeErr(w, 500, "no se pudo revocar la invitación")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/portal/invites/lookup?token=... — public: show the invite's org +
// email so the accept page can display them (no token value echoed back).
func (s *Server) portalLookupInvite(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		writeErr(w, 400, "token requerido")
		return
	}
	inv, err := pg.GetInviteByToken(token)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if inv == nil || inv.Accepted {
		writeErr(w, 404, "invitación no válida o ya usada")
		return
	}
	org, err := pg.GetOrg(inv.OrgID)
	if err != nil || org == nil {
		writeErr(w, 404, "organización no encontrada")
		return
	}
	writeJSON(w, 200, map[string]any{"email": inv.Email, "role": inv.Role, "org": org.Name})
}

// POST /api/portal/invites/accept {token, password} — public: create the user in
// the invited org with the invited role, then issue a portal token.
func (s *Server) portalAcceptInvite(w http.ResponseWriter, r *http.Request) {
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
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		writeErr(w, 400, "token requerido")
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		writeErr(w, 400, "la contraseña debe tener entre 8 y 72 caracteres")
		return
	}
	inv, err := pg.GetInviteByToken(req.Token)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if inv == nil || inv.Accepted {
		writeErr(w, 404, "invitación no válida o ya usada")
		return
	}
	if !validInviteRole(inv.Role) {
		writeErr(w, 400, "rol de invitación inválido")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, 500, "no se pudo crear la cuenta")
		return
	}
	userID, err := pg.AcceptInvite(inv.ID, inv.OrgID, inv.Email, inv.Role, string(hash))
	if err != nil {
		if errors.Is(err, db.ErrEmailTaken) {
			writeErr(w, 409, "ese email ya tiene una cuenta")
			return
		}
		writeErr(w, 500, "no se pudo aceptar la invitación")
		return
	}
	tok, err := signPortalJWT(s.jwtKey, userID, inv.OrgID)
	if err != nil {
		writeErr(w, 500, "no se pudo emitir el token")
		return
	}
	s.audit(r, &inv.OrgID, inv.Email, "portal_user", "invite.accept",
		"user:"+strconv.FormatInt(userID, 10), "role="+inv.Role)
	s.sendVerification(userID, inv.Email)
	writeJSON(w, 201, map[string]any{"token": tok})
}
