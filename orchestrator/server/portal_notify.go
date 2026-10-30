// Portal email notifications. Two kinds of triggers:
//   - events, mailed when they happen: results released (to the whole org),
//     finding assigned (to the new owner), verification requested (to NEXUS
//     staff at NEXUS_OPS_EMAIL);
//   - a periodic sweep (NEXUS_NOTIFY_INTERVAL, default 15m) over each org's
//     remediation state: NEXUS verification verdicts, findings past their
//     target date and risk acceptances about to expire.
//
// Every notification is claimed in the portal_notifications ledger before it is
// queued, so it is mailed at most once even across restarts or concurrent
// sweeps. Mail is queued in portal_mail_outbox and delivered by a background
// worker with retries, so SMTP latency or outages never reach a request.
// Recipients must be active, verified and not opted out (db.NotifyRecipients).
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/nexus-ia/nexus/db"
)

const (
	mailMaxAttempts   = 5
	mailBatch         = 20
	mailRetention     = 30 * 24 * time.Hour
	verdictFreshness  = 7 * 24 * time.Hour
	riskExpiryWarning = 7 // days
)

type portalNotifier struct {
	s       *Server
	wake    chan struct{}
	sweepMu sync.Mutex // one sweep at a time (timer and operator trigger)
}

func newPortalNotifier(s *Server) *portalNotifier {
	return &portalNotifier{s: s, wake: make(chan struct{}, 1)}
}

func notifyInterval() time.Duration {
	if d, err := time.ParseDuration(envStr("NEXUS_NOTIFY_INTERVAL", "")); err == nil && d >= time.Minute {
		return d
	}
	return 15 * time.Minute
}

// run delivers queued mail (every 30s, or right after something is queued) and
// sweeps for remediation events on the notify interval.
func (n *portalNotifier) run(ctx context.Context) {
	deliver := time.NewTicker(30 * time.Second)
	defer deliver.Stop()
	sweep := time.NewTimer(time.Minute) // first sweep shortly after start
	defer sweep.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deliver.C:
		case <-n.wake:
		case <-sweep.C:
			n.sweep(time.Now())
			sweep.Reset(notifyInterval())
		}
		n.deliver()
	}
}

func (n *portalNotifier) poke() {
	select {
	case n.wake <- struct{}{}:
	default:
	}
}

// deliver sends due outbox mail, rescheduling failures with exponential backoff
// (1, 2, 4, 8 minutes) and giving up after mailMaxAttempts.
func (n *portalNotifier) deliver() {
	pg := n.s.m.pg
	if pg == nil {
		return
	}
	due, err := pg.DueMail(mailBatch)
	if err != nil {
		log.Printf("[portal-notify] outbox: %v", err)
		return
	}
	for _, m := range due {
		err := deliverMail(m.To, mailMessage{Subject: m.Subject, Text: m.Text, HTML: m.HTML})
		if err == nil {
			_ = pg.MarkMailSent(m.ID)
			continue
		}
		attempt := m.Attempts + 1
		final := attempt >= mailMaxAttempts
		next := time.Now().Add(time.Minute << (attempt - 1))
		if final {
			log.Printf("[portal-notify] giving up on mail %d to %s after %d attempts: %v", m.ID, m.To, attempt, err)
		}
		_ = pg.MarkMailFailed(m.ID, truncate(err.Error(), 500), next, final)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// enqueue renders a notification and queues it for one recipient.
func (n *portalNotifier) enqueue(orgID *int64, kind, to string, c mailContent) bool {
	pg := n.s.m.pg
	if pg == nil {
		return false
	}
	m, err := renderMail(c)
	if err != nil {
		log.Printf("[portal-notify] render %s: %v", kind, err)
		return false
	}
	if err := pg.EnqueueMail(orgID, kind, to, m.Subject, m.Text, m.HTML); err != nil {
		log.Printf("[portal-notify] enqueue %s: %v", kind, err)
		return false
	}
	n.poke()
	return true
}

// claim records an event in the ledger; false means it was already notified.
func (n *portalNotifier) claim(orgID int64, kind, ref string) bool {
	ok, err := n.s.m.pg.ClaimNotification(orgID, kind, ref)
	if err != nil {
		log.Printf("[portal-notify] claim %s/%s: %v", kind, ref, err)
	}
	return ok
}

func remediationLink(fid int64) string {
	return portalLink(fmt.Sprintf("/portal?tab=remediation&finding=%d", fid))
}

// ---- events -----------------------------------------------------------------

// resultsReleased tells every recipient of the org that an evaluation's results
// are available. Mailed once per evaluation, even if it is retracted and
// released again.
func (n *portalNotifier) resultsReleased(pr *db.AdminPortalRequest) {
	if n == nil || n.s.m.pg == nil || !n.claim(pr.OrgID, "results", strconv.FormatInt(pr.ID, 10)) {
		return
	}
	recips, err := n.s.m.pg.NotifyRecipients(pr.OrgID)
	if err != nil {
		log.Printf("[portal-notify] recipients: %v", err)
		return
	}
	link := portalLink(fmt.Sprintf("/portal?tab=requests&request=%d", pr.ID))
	for _, r := range recips {
		t := notifyTextFor(r.Lang)
		n.enqueue(&pr.OrgID, "results_released", r.Email, mailContent{
			Lang: t.Lang, Subject: t.ReleasedSubject(pr.Title), Preheader: t.ReleasedIntro(pr.Title),
			Title: t.ReleasedTitle, Org: pr.OrgName, Intro: []string{t.ReleasedIntro(pr.Title)},
			CTA: t.ReleasedCTA, CTALink: link, Note: t.ReleasedNote,
		})
	}
}

// findingAssigned tells the new owner of a finding, unless they assigned it to
// themselves.
func (n *portalNotifier) findingAssigned(orgID, assigneeID, actorID int64, actor string, it *db.RemediationItem) {
	if n == nil || n.s.m.pg == nil || it == nil || assigneeID == actorID {
		return
	}
	r, err := n.s.m.pg.NotifyRecipientByID(orgID, assigneeID)
	if err != nil || r == nil {
		return
	}
	t := notifyTextFor(r.Lang)
	ref := t.FindingRef(it.FindingID)
	intro := []string{t.AssignedIntro(actor, ref, t.sevName(it.Severity), it.RequestTitle)}
	if it.DueDate != "" {
		intro = append(intro, t.DueLine(t.dateStr(it.DueDate)))
	}
	n.enqueue(&orgID, "finding_assigned", r.Email, mailContent{
		Lang: t.Lang, Subject: t.AssignedSubject(it.FindingID), Preheader: intro[0],
		Title: t.AssignedTitle, Org: n.orgName(orgID), Intro: intro,
		CTA: t.OpenCTA, CTALink: remediationLink(it.FindingID),
	})
}

// retestRequested tells NEXUS staff (NEXUS_OPS_EMAIL) that a client asks for a
// verification. Without that address the console queue is the only signal.
func (n *portalNotifier) retestRequested(orgID int64, it *db.RemediationItem) {
	to := envStr("NEXUS_OPS_EMAIL", "")
	if n == nil || to == "" || it == nil {
		return
	}
	t := notifyTextFor("es")
	org := n.orgName(orgID)
	intro := t.OpsIntro(org, t.FindingRef(it.FindingID), t.sevName(it.Severity), it.RequestTitle)
	n.enqueue(&orgID, "retest_requested", to, mailContent{
		Lang: t.Lang, Subject: t.OpsSubject(org, it.FindingID), Preheader: intro, Title: t.OpsTitle,
		Org: org, Intro: []string{intro}, CTA: t.OpsCTA, CTALink: consoleLink("/system/portal-retests"), Staff: true,
	})
}

func (n *portalNotifier) orgName(orgID int64) string {
	if o, err := n.s.m.pg.GetOrg(orgID); err == nil && o != nil {
		return o.Name
	}
	return ""
}

// ---- sweep ------------------------------------------------------------------

type digest struct {
	r        db.NotifyRecipient
	overdue  []*db.RemediationItem
	expiring []*db.RemediationItem
}

// sweep looks for remediation events in every org with released results and
// returns how many emails it queued.
func (n *portalNotifier) sweep(now time.Time) int {
	if n == nil || n.s.m.pg == nil {
		return 0
	}
	n.sweepMu.Lock()
	defer n.sweepMu.Unlock()
	pg := n.s.m.pg
	if purged, err := pg.PurgeOldMail(now.Add(-mailRetention)); err == nil && purged > 0 {
		log.Printf("[portal-notify] purged %d old outbox rows", purged)
	}
	if tk, ss, inv, err := pg.PortalHousekeeping(); err != nil {
		log.Printf("[portal-notify] housekeeping: %v", err)
	} else if tk+ss+inv > 0 {
		log.Printf("[portal-notify] housekeeping removed %d tokens, %d sessions, %d invitations", tk, ss, inv)
	}
	orgs, err := pg.NotifyOrgs()
	if err != nil {
		log.Printf("[portal-notify] orgs: %v", err)
		return 0
	}
	queued := 0
	for _, orgID := range orgs {
		queued += n.sweepOrg(orgID, now)
	}
	return queued
}

func (n *portalNotifier) sweepOrg(orgID int64, now time.Time) int {
	pg := n.s.m.pg
	items, err := pg.ListRemediation(orgID, now)
	if err != nil {
		log.Printf("[portal-notify] org %d: %v", orgID, err)
		return 0
	}
	recips, err := pg.NotifyRecipients(orgID)
	if err != nil {
		log.Printf("[portal-notify] org %d recipients: %v", orgID, err)
		return 0
	}
	requester, accepter, err := pg.RemediationActors(orgID)
	if err != nil {
		log.Printf("[portal-notify] org %d actors: %v", orgID, err)
		return 0
	}
	byID := map[int64]db.NotifyRecipient{}
	var admins []int64
	for _, r := range recips {
		byID[r.ID] = r
		if r.Role == "owner" || r.Role == "admin" {
			admins = append(admins, r.ID)
		}
	}
	// audience: eligible recipients among the given user ids, without repeats.
	audience := func(ids ...int64) []db.NotifyRecipient {
		seen := map[int64]bool{}
		out := []db.NotifyRecipient{}
		for _, id := range ids {
			if r, ok := byID[id]; ok && !seen[id] {
				seen[id] = true
				out = append(out, r)
			}
		}
		return out
	}
	org := n.orgName(orgID)
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	digests := map[int64]*digest{}
	addDigest := func(rs []db.NotifyRecipient, it *db.RemediationItem, overdue bool) {
		for _, r := range rs {
			d := digests[r.ID]
			if d == nil {
				d = &digest{r: r}
				digests[r.ID] = d
			}
			if overdue {
				d.overdue = append(d.overdue, it)
			} else {
				d.expiring = append(d.expiring, it)
			}
		}
	}
	queued := 0
	for _, it := range items {
		fid := it.FindingID
		assignee := int64(0)
		if it.AssigneeID != nil {
			assignee = *it.AssigneeID
		}

		// NEXUS verification verdict on a client request (fresh results only, so a
		// first deployment does not mail old history).
		if lr := it.LastRetest; lr != nil && lr.FinishedAt != nil && it.RetestRequestedAt != nil &&
			lr.FinishedAt.After(*it.RetestRequestedAt) && now.Sub(*lr.FinishedAt) <= verdictFreshness &&
			(lr.Verdict == "fixed" || lr.Verdict == "reproduced") &&
			n.claim(orgID, "verdict", fmt.Sprintf("%d:%d", fid, lr.FinishedAt.Unix())) {
			for _, r := range audience(append([]int64{assignee, requester[fid]}, admins...)...) {
				t := notifyTextFor(r.Lang)
				ref, sev := t.FindingRef(fid), t.sevName(it.Severity)
				c := mailContent{Lang: t.Lang, Org: org, CTA: t.OpenCTA, CTALink: remediationLink(fid)}
				if lr.Verdict == "fixed" {
					c.Subject, c.Title, c.Intro = t.FixedSubject(fid), t.FixedTitle, []string{t.FixedIntro(ref, sev, it.RequestTitle)}
				} else {
					c.Subject, c.Title, c.Intro = t.ReproSubject(fid), t.ReproTitle, []string{t.ReproIntro(ref, sev, it.RequestTitle)}
				}
				c.Preheader = c.Intro[0]
				if n.enqueue(&orgID, "retest_verdict", r.Email, c) {
					queued++
				}
			}
		}

		if it.Overdue && n.claim(orgID, "overdue", fmt.Sprintf("%d:%s", fid, it.DueDate)) {
			addDigest(audience(append([]int64{assignee}, admins...)...), it, true)
		}

		if it.State == db.RemRiskAccepted && it.RiskExpiresAt != "" {
			exp, err := time.Parse("2006-01-02", it.RiskExpiresAt)
			if err == nil && !exp.Before(today) && !exp.After(today.AddDate(0, 0, riskExpiryWarning)) &&
				n.claim(orgID, "risk_expiring", fmt.Sprintf("%d:%s", fid, it.RiskExpiresAt)) {
				addDigest(audience(append([]int64{accepter[fid]}, admins...)...), it, false)
			}
		}
	}

	for _, d := range digests {
		t := notifyTextFor(d.r.Lang)
		c := mailContent{
			Lang: t.Lang, Subject: t.DigestSubject(len(d.overdue), len(d.expiring)), Preheader: t.DigestIntro,
			Title: t.DigestTitle, Org: org, Intro: []string{t.DigestIntro},
			CTA: t.DigestCTA, CTALink: portalLink("/portal?tab=remediation"),
		}
		if len(d.overdue) > 0 {
			sec := mailSection{Title: t.OverdueSection}
			for _, it := range d.overdue {
				due, _ := time.Parse("2006-01-02", it.DueDate)
				days := int(today.Sub(due).Hours() / 24)
				sec.Items = append(sec.Items, mailItem{
					Sev: it.Severity, SevLabel: t.sevLabel(it.Severity),
					Text: t.ItemText(t.FindingRef(it.FindingID), it.RequestTitle), Meta: t.OverdueMeta(t.dateStr(it.DueDate), days),
				})
			}
			c.Sections = append(c.Sections, sec)
		}
		if len(d.expiring) > 0 {
			sec := mailSection{Title: t.ExpiringSection}
			for _, it := range d.expiring {
				sec.Items = append(sec.Items, mailItem{
					Sev: it.Severity, SevLabel: t.sevLabel(it.Severity),
					Text: t.ItemText(t.FindingRef(it.FindingID), it.RequestTitle), Meta: t.ExpiresMeta(t.dateStr(it.RiskExpiresAt)),
				})
			}
			c.Sections = append(c.Sections, sec)
		}
		if n.enqueue(&orgID, "remediation_digest", d.r.Email, c) {
			queued++
		}
	}
	return queued
}

// ---- operator endpoints -------------------------------------------------------

// POST /api/admin/portal/notifications/run — run a sweep now and deliver.
func (s *Server) adminRunNotifications(w http.ResponseWriter, r *http.Request) {
	if s.portalDB(w) == nil || s.portalNotify == nil {
		return
	}
	queued := s.portalNotify.sweep(time.Now())
	s.portalNotify.deliver()
	writeJSON(w, 200, map[string]any{"ok": true, "queued": queued})
}

// GET /api/admin/portal/mail-outbox — recent notification mail (no bodies).
func (s *Server) adminListMailOutbox(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	mail, err := pg.AdminListOutbox(200)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la bandeja de salida")
		return
	}
	writeJSON(w, 200, map[string]any{
		"mail":          mail,
		"smtp":          envStr("NEXUS_SMTP_HOST", "") != "",
		"ops_email_set": envStr("NEXUS_OPS_EMAIL", "") != "",
		"interval":      notifyInterval().String(),
	})
}

// GET /api/admin/portal/mail-outbox/{id}/preview — the HTML of one message, with
// the logo inlined, under a CSP that allows nothing but inline styles and images.
func (s *Server) adminPreviewMail(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	m, err := pg.GetOutboxMail(id)
	if err != nil {
		writeErr(w, 500, "no se pudo leer el correo")
		return
	}
	if m == nil {
		writeErr(w, 404, "correo no encontrado")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src data:; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(previewHTML(m.HTML)))
}
