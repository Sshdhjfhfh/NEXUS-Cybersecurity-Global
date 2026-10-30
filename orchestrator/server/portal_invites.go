// Invitation mail and resend. The invitation link carries a one-time token, so
// like password-reset links it is mailed right away and never queued or stored.
package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/nexus-ia/nexus/db"
)

// sendInviteMail mails an invitation link to its recipient, in the inviting
// admin's language (the invitee has no account and so no preference yet).
func (s *Server) sendInviteMail(orgID, inviterID int64, to, role, token string, expires time.Time) {
	pg := s.m.pg
	t := notifyTextFor(userLang(pg, inviterID))
	inviter := ""
	if u, err := pg.GetPortalUserByID(inviterID); err == nil && u != nil {
		inviter = u.Email
	}
	org := ""
	if o, err := pg.GetOrg(orgID); err == nil && o != nil {
		org = headerSafe(o.Name)
	}
	roleName := t.RoleName[role]
	if roleName == "" {
		roleName = role
	}
	intro := t.InviteIntro(inviter, org, roleName)
	c := mailContent{
		Lang: t.Lang, Subject: t.InviteSubject(org), Preheader: intro, Title: t.InviteTitle, Org: org,
		Intro: []string{intro, t.InviteExpires(expires.UTC().Format("2006-01-02 15:04") + " UTC")},
		CTA:   t.InviteCTA, CTALink: portalLink("/portal/accept?token=" + token),
		Note: t.InviteNote, Invite: true,
	}
	go func() {
		m, err := renderMail(c)
		if err == nil {
			err = deliverMail(to, m)
		}
		if err != nil {
			log.Printf("[portal-mail] invitation to %s failed: %v", to, err)
		}
	}()
}

// POST /api/portal/invites/{id}/resend — admin+: issue a new link (the previous
// one stops working), restart the validity period and mail it again. Also how
// an expired invitation is renewed.
func (s *Server) portalResendInvite(w http.ResponseWriter, r *http.Request) {
	orgID, actorID, _, ok := s.portalRequireRole(w, r, "admin")
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
	token, err := newInviteToken()
	if err != nil {
		writeErr(w, 500, "no se pudo generar la invitación")
		return
	}
	inv, err := pg.ReissueInvite(orgID, id, hashToken(token))
	switch {
	case errors.Is(err, db.ErrInviteTooSoon):
		writeErr(w, 429, "espera un minuto antes de volver a enviar esta invitación")
		return
	case err != nil:
		writeErr(w, 503, "almacenamiento no disponible")
		return
	case inv == nil:
		writeErr(w, 404, "invitación no encontrada")
		return
	}
	s.sendInviteMail(orgID, actorID, inv.Email, inv.Role, token, inv.ExpiresAt)
	s.audit(r, &orgID, "user:"+strconv.FormatInt(actorID, 10), "portal_user", "invite.resend",
		"invite:"+strconv.FormatInt(id, 10), "email="+inv.Email)
	writeJSON(w, 200, map[string]any{
		"ok":         true,
		"id":         id,
		"token":      token,
		"email":      inv.Email,
		"role":       inv.Role,
		"expires_at": inv.ExpiresAt,
		"emailed":    true,
	})
}
