// Server-side portal sessions. Every portal token names a row in
// portal_sessions (sid claim), checked on every request by portalAuth, so a
// user can sign out one device, see where their account is open, and close the
// rest; and an admin revoking a member or a password reset ends them all.
package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

// issuePortalSession opens a server-side session and returns its token. When a
// sign-in comes from a browser and address the user has not used in the last
// 90 days, they get a security email (not on a brand-new account's first one).
func (s *Server) issuePortalSession(r *http.Request, pg *db.DB, userID, orgID int64, email, amr string) (string, error) {
	sid := randB64(18)
	exp := time.Now().Add(jwtTTL)
	ip := clientIP(r)
	ua := trimLimit(r.UserAgent(), 400)
	first, known, err := pg.CreatePortalSession(sid, userID, orgID, amr, ip, ua, exp)
	if err != nil {
		return "", err
	}
	if !first && !known {
		s.sendNewSignInAlert(userID, email, ip, ua, amr)
	}
	return signPortalJWT(s.jwtKey, userID, orgID, amr, sid, exp)
}

// sendNewSignInAlert tells the user their account was opened on a new device.
func (s *Server) sendNewSignInAlert(userID int64, email, ip, ua, amr string) {
	t := notifyTextFor(userLang(s.m.pg, userID))
	browser, osName := describeUserAgent(ua)
	method := t.SignInMethodPassword
	if amr == "sso" {
		method = t.SignInMethodSSO
	}
	s.sendSecurityMailAsync(email, mailContent{
		Lang: t.Lang, Subject: t.SignInSubject, Preheader: t.SignInIntro, Title: t.SignInTitle,
		Intro: []string{
			t.SignInIntro,
			t.SignInWhen + ": " + time.Now().UTC().Format("2006-01-02 15:04") + " UTC",
			t.SignInDevice + ": " + t.SignInDeviceText(browser, osName),
			t.SignInIP + ": " + ip,
			t.SignInMethod + ": " + method,
		},
		CTA: t.SignInCTA, CTALink: portalLink("/portal?tab=security"),
		Note: t.SignInNote, Security: true,
	})
}

// describeUserAgent reduces a User-Agent header to a browser and an operating
// system name for people to recognise their devices ("" when unknown). Order
// matters: Edge and Opera also say Chrome, Chrome also says Safari, Android
// also says Linux.
func describeUserAgent(ua string) (browser, osName string) {
	switch {
	case strings.Contains(ua, "Edg/"):
		browser = "Edge"
	case strings.Contains(ua, "OPR/"):
		browser = "Opera"
	case strings.Contains(ua, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "Chrome/") || strings.Contains(ua, "CriOS/"):
		browser = "Chrome"
	case strings.Contains(ua, "Safari/") && strings.Contains(ua, "Version/"):
		browser = "Safari"
	case strings.HasPrefix(ua, "curl/"):
		browser = "curl"
	case strings.HasPrefix(ua, "python-requests/"), strings.HasPrefix(ua, "Python-urllib/"):
		browser = "Python"
	}
	switch {
	case strings.Contains(ua, "Windows"):
		osName = "Windows"
	case strings.Contains(ua, "Android"):
		osName = "Android"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"):
		osName = "iOS"
	case strings.Contains(ua, "Mac OS X"):
		osName = "macOS"
	case strings.Contains(ua, "CrOS"):
		osName = "ChromeOS"
	case strings.Contains(ua, "Linux"):
		osName = "Linux"
	}
	return browser, osName
}

// portalSessionID is the server-side session of the request's token.
func (s *Server) portalSessionID(r *http.Request) string {
	c, _ := parsePortalJWT(extractToken(r), s.jwtKey)
	return c.SID
}

// POST /api/portal/logout — ends the caller's own session. It needs only a
// valid token, not the full gate, so a member held back by a policy (network,
// enrollment) can still sign out.
func (s *Server) portalLogout(w http.ResponseWriter, r *http.Request) {
	c, ok := parsePortalJWT(extractToken(r), s.jwtKey)
	if !ok {
		writeErr(w, 401, "no autorizado")
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	ended, err := pg.RevokePortalSession(c.UserID, c.SID)
	if err != nil {
		writeErr(w, 500, "no se pudo cerrar la sesión")
		return
	}
	if ended {
		email := ""
		if u, err := pg.GetPortalUserByID(c.UserID); err == nil && u != nil {
			email = u.Email
		}
		s.audit(r, &c.OrgID, email, "portal_user", "auth.logout", "user:"+strconv.FormatInt(c.UserID, 10), "")
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/portal/sessions — the caller's open sessions.
func (s *Server) portalListSessions(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	list, err := pg.ListPortalSessions(userID)
	if err != nil {
		writeErr(w, 500, "no se pudieron cargar las sesiones")
		return
	}
	current := s.portalSessionID(r)
	out := make([]map[string]any, 0, len(list))
	for _, x := range list {
		browser, osName := describeUserAgent(x.UserAgent)
		out = append(out, map[string]any{
			"id": x.ID, "amr": x.AMR, "ip": x.IP, "browser": browser, "os": osName,
			"created_at": x.CreatedAt, "last_seen_at": x.LastSeenAt, "expires_at": x.ExpiresAt,
			"current": x.ID == current,
		})
	}
	writeJSON(w, 200, map[string]any{"sessions": out})
}

// DELETE /api/portal/sessions/{id} — sign out one of the caller's own sessions.
func (s *Server) portalRevokeSession(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	id := r.PathValue("id")
	ended, err := pg.RevokePortalSession(userID, id)
	if err != nil {
		writeErr(w, 500, "no se pudo cerrar la sesión")
		return
	}
	if !ended {
		writeErr(w, 404, "sesión no encontrada")
		return
	}
	s.audit(r, &orgID, s.portalActorEmail(pg, userID), "portal_user", "auth.session_revoked",
		"user:"+strconv.FormatInt(userID, 10), "current="+strconv.FormatBool(id == s.portalSessionID(r)))
	writeJSON(w, 200, map[string]any{"ok": true})
}

// POST /api/portal/sessions/revoke-others — sign out every other device.
func (s *Server) portalRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	orgID, userID, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	n, err := pg.RevokeOtherPortalSessions(userID, s.portalSessionID(r))
	if err != nil {
		writeErr(w, 500, "no se pudieron cerrar las sesiones")
		return
	}
	s.audit(r, &orgID, s.portalActorEmail(pg, userID), "portal_user", "auth.sessions_revoked",
		"user:"+strconv.FormatInt(userID, 10), "count="+strconv.FormatInt(n, 10))
	writeJSON(w, 200, map[string]any{"revoked": n})
}

func (s *Server) portalActorEmail(pg *db.DB, userID int64) string {
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		log.Printf("[portal] actor lookup user=%d: %v", userID, err)
		return "user:" + strconv.FormatInt(userID, 10)
	}
	return u.Email
}
