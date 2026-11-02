// Organization security policy: required 2FA, network allowlist and maximum
// session lifetime. Set by the org owner; enforced centrally in portalAuth (so
// no endpoint can skip it) and at login (for a clear message + audit trail).
package server

import (
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nexus-ia/nexus/db"
)

const (
	maxSessionHours   = int(jwtTTL / time.Hour) // a policy can only shorten sessions
	maxAllowlistItems = 50
)

// mfaEnrollmentPaths are the only calls a member may make while the org
// requires 2FA and they have not enrolled yet: see who they are, enroll, and
// manage their own account basics.
var mfaEnrollmentPaths = map[string]bool{
	"/api/portal/me":              true,
	"/api/portal/me/preferences":  true,
	"/api/portal/mfa/status":      true,
	"/api/portal/mfa/setup":       true,
	"/api/portal/mfa/enable":      true,
	"/api/portal/password/change": true,
	"/api/portal/email/resend":    true,
	"/api/portal/terms/accept":    true,
	"/api/portal/logout":          true,
}

// normalizeAllowlist validates CIDRs or single addresses and returns them in
// canonical, de-duplicated form ("10.0.0.7/24" -> "10.0.0.0/24", "1.2.3.4" ->
// "1.2.3.4/32").
func normalizeAllowlist(in []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		var p netip.Prefix
		if strings.Contains(v, "/") {
			pp, err := netip.ParsePrefix(v)
			if err != nil {
				return nil, fmt.Errorf("red inválida: %s", v)
			}
			p = pp.Masked()
		} else {
			a, err := netip.ParseAddr(v)
			if err != nil {
				return nil, fmt.Errorf("red inválida: %s", v)
			}
			a = a.Unmap()
			p = netip.PrefixFrom(a, a.BitLen())
		}
		if p.Addr().Is4In6() && p.Bits() >= 96 {
			p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
		}
		if s := p.String(); !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) > maxAllowlistItems {
		return nil, fmt.Errorf("máximo %d redes", maxAllowlistItems)
	}
	return out, nil
}

// ipAllowed reports whether ip falls in any allowlisted network. An empty list
// allows any address; an unparseable address is never allowed.
func ipAllowed(ip string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return true
	}
	a, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	a = a.Unmap()
	for _, c := range allowlist {
		if p, err := netip.ParsePrefix(c); err == nil && p.Contains(a) {
			return true
		}
	}
	return false
}

// ipBlockAudit throttles auth.ip_blocked audit entries to one per user and
// address every 10 minutes, so a client polling from a blocked network cannot
// flood the org's audit log.
var ipBlockAudit = struct {
	sync.Mutex
	last map[string]time.Time
}{last: map[string]time.Time{}}

func shouldAuditIPBlock(userID int64, ip string) bool {
	k := strconv.FormatInt(userID, 10) + "|" + ip
	now := time.Now()
	ipBlockAudit.Lock()
	defer ipBlockAudit.Unlock()
	if t, ok := ipBlockAudit.last[k]; ok && now.Sub(t) < 10*time.Minute {
		return false
	}
	if len(ipBlockAudit.last) > 10000 {
		ipBlockAudit.last = map[string]time.Time{}
	}
	ipBlockAudit.last[k] = now
	return true
}

// enforceOrgPolicy applies the org security policy to an authenticated portal
// request (called from portalAuth). It writes the error and returns false when
// the request must be refused.
func (s *Server) enforceOrgPolicy(w http.ResponseWriter, r *http.Request, a *db.PortalAuthInfo, userID, iat int64) bool {
	if a.SessionMaxHours > 0 && a.SessionMaxHours < maxSessionHours &&
		time.Now().Unix()-iat > int64(a.SessionMaxHours)*3600 {
		writeErr(w, 401, "sesión expirada por la política de la organización; inicia sesión de nuevo")
		return false
	}
	if ip := clientIP(r); !ipAllowed(ip, a.IPAllowlist) {
		if shouldAuditIPBlock(userID, ip) {
			actor := "user:" + strconv.FormatInt(userID, 10)
			if u, err := s.m.pg.GetPortalUserByID(userID); err == nil && u != nil {
				actor = u.Email
			}
			s.audit(r, &a.OrgID, actor, "portal_user", "auth.ip_blocked", "user:"+strconv.FormatInt(userID, 10), "ip="+ip)
		}
		writeErr(w, 403, "acceso no permitido desde esta red por la política de la organización")
		return false
	}
	// An SSO session's second factor is enforced by the org's identity provider.
	if a.RequireMFA && !a.MFAEnabled && !mfaEnrollmentPaths[r.URL.Path] && !portalSessionIsSSO(r, s.jwtKey) {
		writeErr(w, 403, "mfa_enrollment_required")
		return false
	}
	return true
}

// GET /api/portal/security-policy — admins and auditors read the policy.
func (s *Server) portalGetSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	s.writeSecurityPolicy(w, r, pg, orgID)
}

func (s *Server) writeSecurityPolicy(w http.ResponseWriter, r *http.Request, pg *db.DB, orgID int64) {
	p, err := pg.GetOrgSecurityPolicy(orgID)
	if err != nil || p == nil {
		writeErr(w, 500, "no se pudo leer la política")
		return
	}
	without, err := pg.MembersWithoutMFA(orgID)
	if err != nil {
		without = []string{}
	}
	writeJSON(w, 200, map[string]any{
		"policy":              p,
		"your_ip":             clientIP(r),
		"max_session_hours":   maxSessionHours,
		"members_without_mfa": without,
	})
}

// PUT /api/portal/security-policy {require_mfa, ip_allowlist, session_max_hours}
// — owner only. Refuses changes that would lock the caller out.
func (s *Server) portalSetSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "owner")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		RequireMFA      bool     `json:"require_mfa"`
		IPAllowlist     []string `json:"ip_allowlist"`
		SessionMaxHours int      `json:"session_max_hours"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if req.SessionMaxHours < 1 || req.SessionMaxHours > maxSessionHours {
		writeErr(w, 400, fmt.Sprintf("la duración de sesión debe estar entre 1 y %d horas", maxSessionHours))
		return
	}
	allow, err := normalizeAllowlist(req.IPAllowlist)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if req.RequireMFA && !u.MFAEnabled {
		writeErr(w, 400, "activa tu propia verificación en dos pasos antes de exigirla a la organización")
		return
	}
	if ip := clientIP(r); !ipAllowed(ip, allow) {
		writeErr(w, 400, fmt.Sprintf("tu IP actual (%s) no está en la lista; añádela para no bloquear tu propio acceso", ip))
		return
	}
	prev, _ := pg.GetOrgSecurityPolicy(orgID)
	if err := pg.SetOrgSecurityPolicy(orgID, req.RequireMFA, allow, req.SessionMaxHours, u.Email); err != nil {
		writeErr(w, 500, "no se pudo guardar la política")
		return
	}
	detail := fmt.Sprintf("require_mfa=%t ip_allowlist=[%s] session_max_hours=%d",
		req.RequireMFA, strings.Join(allow, " "), req.SessionMaxHours)
	if prev != nil {
		detail += fmt.Sprintf(" (antes: require_mfa=%t ip_allowlist=[%s] session_max_hours=%d)",
			prev.RequireMFA, strings.Join(prev.IPAllowlist, " "), prev.SessionMaxHours)
	}
	s.audit(r, &orgID, u.Email, "portal_user", "policy.update", "org:"+strconv.FormatInt(orgID, 10), detail)
	s.writeSecurityPolicy(w, r, pg, orgID)
}

// DELETE /api/admin/portal/orgs/{id}/security-policy — operator recovery: an
// owner who locked the whole org out (e.g. allowlisted networks they no longer
// use) gets the default policy back. Console-authenticated, audited.
func (s *Server) adminResetSecurityPolicy(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	orgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orgID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	if err := pg.SetOrgSecurityPolicy(orgID, false, nil, maxSessionHours, "operator"); err != nil {
		writeErr(w, 404, "organización no encontrada")
		return
	}
	s.audit(r, &orgID, "operator", "operator", "policy.reset", "org:"+strconv.FormatInt(orgID, 10), "")
	writeJSON(w, 200, map[string]any{"ok": true})
}
