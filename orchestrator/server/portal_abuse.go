package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Automatic abuse control: an authenticated member who trips enough high-risk
// signals in a sliding window is automatically suspended (account disabled +
// all sessions revoked), with an audit entry and an operator alert. This adds
// the automatic *reaction* the audit trail alone did not provide; manual
// disable, brute-force login lockout and scope enforcement remain.
//
// Rollout safety (shadow mode): suspension is OFF by default. Until
// NEXUS_ABUSE_AUTOSUSPEND is enabled, a trip only audits ("security.abuse_detected")
// and alerts, so weights/thresholds can be tuned against real traffic before an
// account is actually disabled. Owners are never auto-suspended (an org must not
// lock itself out), and suspension is reversible (disable, not delete).
//
// Signals come only from *authenticated* members, never from failed logins —
// auto-suspending on failed logins would let an attacker lock out a victim; that
// is the brute-force lockout's job, which is temporary and IP-scoped.

const (
	abuseWindow    = 15 * time.Minute
	abuseThreshold = 6
	abuseMaxUsers  = 10000
)

// abuseWeights scores each signal; acting outside scope weighs more than hitting
// a network location the org policy disallows.
var abuseWeights = map[string]int{
	"auth.ip_blocked":  1, // kept hitting a disallowed network under the org policy
	"authz.forbidden":  2, // kept trying actions above their role / not theirs
	"scope.blocked":    3, // tried to act outside the authorized engagement scope
	"export.excessive": 3, // abnormal bulk export
}

type abuseEvent struct {
	t time.Time
	w int
}

var abuse = struct {
	sync.Mutex
	events    map[int64][]abuseEvent
	suspended map[int64]time.Time
}{events: map[int64][]abuseEvent{}, suspended: map[int64]time.Time{}}

// abuseRecord adds one signal for userID and returns the weighted score within
// the window and whether this crossed the threshold for the first time in the
// window. Pure in-memory and deterministic in `now`, so it is unit-testable
// without a database.
func abuseRecord(userID int64, kind string, now time.Time) (score int, trip bool) {
	w := abuseWeights[kind]
	if w == 0 {
		w = 1
	}
	abuse.Lock()
	defer abuse.Unlock()
	kept := abuse.events[userID][:0]
	for _, e := range abuse.events[userID] {
		if now.Sub(e.t) < abuseWindow {
			kept = append(kept, e)
			score += e.w
		}
	}
	kept = append(kept, abuseEvent{now, w})
	score += w
	abuse.events[userID] = kept
	if len(abuse.events) > abuseMaxUsers { // bound memory; a reset only loses in-window history
		abuse.events = map[int64][]abuseEvent{}
	}
	if t, ok := abuse.suspended[userID]; ok && now.Sub(t) < abuseWindow {
		return score, false // already acted on this user recently
	}
	if score >= abuseThreshold {
		abuse.suspended[userID] = now
		delete(abuse.events, userID)
		return score, true
	}
	return score, false
}

func abuseAutoSuspendEnabled() bool {
	v := strings.TrimSpace(envStr("NEXUS_ABUSE_AUTOSUSPEND", ""))
	return v == "1" || strings.EqualFold(v, "true")
}

// flagAbuse records a signal for an authenticated member and reacts when the
// windowed score crosses the threshold. Best-effort; never blocks the request.
// userID <= 0 (not authenticated) is ignored — there is no account to act on.
func (s *Server) flagAbuse(r *http.Request, orgID, userID int64, kind string) {
	if userID <= 0 {
		return
	}
	if score, trip := abuseRecord(userID, kind, time.Now()); trip {
		s.reactAbuse(r, orgID, userID, kind, score)
	}
}

// reactAbuse runs off the request path. In shadow mode it only audits and
// alerts; when auto-suspend is enabled it also disables the member and revokes
// their sessions. Owners are never auto-suspended.
func (s *Server) reactAbuse(r *http.Request, orgID, userID int64, kind string, score int) {
	go func() {
		pg := s.m.pg
		if pg == nil {
			return
		}
		target := "user:" + strconv.FormatInt(userID, 10)
		email := target
		isOwner := false
		if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
			email = u.Email
			isOwner = u.Role == "owner"
		}
		detail := "score=" + strconv.Itoa(score) + " last=" + kind + " window=" + abuseWindow.String()
		if !abuseAutoSuspendEnabled() || isOwner {
			reason := "shadow mode"
			if isOwner {
				reason = "owner exempt"
			}
			s.audit(r, &orgID, email, "system", "security.abuse_detected", target, detail+" action=none("+reason+")")
			log.Printf("[abuse] detected user %d org %d (%s) — not suspended (%s)", userID, orgID, detail, reason)
			s.alertAbuse(orgID, email, score, kind, false)
			return
		}
		if err := pg.SetPortalUserDisabled(orgID, userID, true); err != nil {
			log.Printf("[abuse] auto-suspend user %d failed: %v", userID, err)
			return
		}
		_ = pg.RevokePortalUserSessionsByID(userID)
		s.audit(r, &orgID, email, "system", "security.auto_suspended", target, detail)
		log.Printf("[abuse] auto-suspended user %d org %d (%s)", userID, orgID, detail)
		s.alertAbuse(orgID, email, score, kind, true)
	}()
}

// alertAbuse notifies the operator (NEXUS_OPS_EMAIL). Best-effort; the event is
// always in the audit trail regardless of whether mail is configured.
func (s *Server) alertAbuse(orgID int64, email string, score int, kind string, suspended bool) {
	to := envStr("NEXUS_OPS_EMAIL", "")
	if to == "" {
		return
	}
	title := "Abuso detectado (modo sombra)"
	action := "No se suspendió la cuenta (modo sombra). Activa NEXUS_ABUSE_AUTOSUSPEND para suspender automáticamente tras afinar los umbrales."
	if suspended {
		title = "Suspensión automática por abuso"
		action = "La cuenta fue desactivada y sus sesiones revocadas. Un administrador puede reactivarla si procede."
	}
	s.sendSecurityMailAsync(to, mailContent{
		Lang:      "es",
		Subject:   "NEXUS · " + title,
		Preheader: "Señales de abuso de un miembro",
		Title:     title,
		Org:       "org:" + strconv.FormatInt(orgID, 10),
		Intro: []string{
			"NEXUS superó el umbral de señales de abuso para un miembro autenticado.",
			"Miembro: " + email,
			"Puntaje: " + strconv.Itoa(score) + " · Última señal: " + kind,
		},
		Note:     action + " Revisa la auditoría de la organización.",
		Security: true,
	})
}
