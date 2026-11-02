// Operations watchdog. It checks what silently breaks a running install — the
// database, backups, free disk, server errors and outgoing mail — and tells
// NEXUS staff at NEXUS_OPS_EMAIL: one mail when a check starts failing, a
// reminder every opsRemindEvery while it keeps failing, and one when it
// recovers. Every change is also logged as [ops-alert]. The console shows the
// current state (/system/ops). It cannot report the backend itself being down:
// an external uptime monitor on /api/ready covers that
// (docs/engineering/MONITORING.md).
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const opsRemindEvery = 24 * time.Hour

// Thresholds for the server-error check over one interval.
const (
	opsMin5xx      = 10 // fewer than this is noise
	opsMax5xxShare = 20 // alert when 1 in 20 responses (5%) or more is a 5xx
)

type opsCheck struct {
	Name   string    `json:"name"`
	Title  string    `json:"title"`
	Status string    `json:"status"` // ok | fail | off (not configured here)
	Detail string    `json:"detail"`
	Since  time.Time `json:"since"` // in this status since
}

type opsResult struct {
	status string
	detail string
}

func opsOK(f string, a ...any) opsResult   { return opsResult{"ok", fmt.Sprintf(f, a...)} }
func opsFail(f string, a ...any) opsResult { return opsResult{"fail", fmt.Sprintf(f, a...)} }
func opsOff(f string, a ...any) opsResult  { return opsResult{"off", fmt.Sprintf(f, a...)} }

type opsWatch struct {
	s       *Server
	dataDir string
	started time.Time
	notify  func(subject string, c mailContent) // sends one alert; replaced in tests

	runMu   sync.Mutex // one round at a time (timer and operator trigger)
	mu      sync.Mutex
	checks  map[string]*opsCheck
	alerted map[string]time.Time // last alert for the current failure
	lastRun time.Time

	// server-error counters at the previous round (or at creation)
	base5xx, baseAll, basePanic int64
}

var opsTitles = []struct{ name, title string }{
	{"database", "Base de datos"},
	{"backups", "Copias de seguridad"},
	{"disk", "Espacio en disco"},
	{"errors", "Errores del servidor"},
	{"mail", "Correo saliente"},
	{"alerts", "Destino de las alertas"},
}

func newOpsWatch(s *Server, dataDir string) *opsWatch {
	o := &opsWatch{s: s, dataDir: dataDir, started: time.Now(), checks: map[string]*opsCheck{}, alerted: map[string]time.Time{},
		base5xx: metrics.c5xx.Load(), baseAll: metrics.total.Load(), basePanic: metrics.panics.Load()}
	o.notify = o.sendAlert
	return o
}

func opsInterval() time.Duration {
	if d, err := time.ParseDuration(envStr("NEXUS_OPS_INTERVAL", "")); err == nil && d >= time.Minute {
		return d
	}
	return 5 * time.Minute
}

func (o *opsWatch) run(ctx context.Context) {
	t := time.NewTimer(time.Minute) // first round shortly after start
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			o.round(time.Now())
			t.Reset(opsInterval())
		}
	}
}

// round runs every check once, records the results and sends the alerts that
// the changes call for.
func (o *opsWatch) round(now time.Time) []opsCheck {
	o.runMu.Lock()
	defer o.runMu.Unlock()
	results := map[string]opsResult{
		"database": o.checkDatabase(),
		"backups":  o.checkBackups(now),
		"disk":     o.checkDisk(),
		"errors":   o.checkErrors(now),
		"mail":     o.checkMail(now),
		"alerts":   o.checkAlertTarget(),
	}
	o.mu.Lock()
	var events []opsEvent
	for _, c := range opsTitles {
		r := results[c.name]
		prev, seen := o.checks[c.name]
		if !seen || prev.Status != r.status {
			wasFailing := seen && prev.Status == "fail"
			o.checks[c.name] = &opsCheck{Name: c.name, Title: c.title, Status: r.status, Detail: r.detail, Since: now}
			switch {
			case r.status == "fail":
				events = append(events, opsEvent{check: *o.checks[c.name], kind: "fail"})
				o.alerted[c.name] = now
			case wasFailing:
				events = append(events, opsEvent{check: *o.checks[c.name], kind: "resolved", was: prev.Detail})
				delete(o.alerted, c.name)
			}
			continue
		}
		prev.Detail = r.detail
		if r.status == "fail" && now.Sub(o.alerted[c.name]) >= opsRemindEvery {
			events = append(events, opsEvent{check: *prev, kind: "reminder"})
			o.alerted[c.name] = now
		}
	}
	o.lastRun = now
	snapshot := o.snapshotLocked()
	o.mu.Unlock()
	for _, e := range events {
		o.alert(e, snapshot)
	}
	return snapshot
}

func (o *opsWatch) snapshotLocked() []opsCheck {
	out := make([]opsCheck, 0, len(opsTitles))
	for _, c := range opsTitles {
		if st, ok := o.checks[c.name]; ok {
			out = append(out, *st)
		}
	}
	return out
}

// ---- checks -------------------------------------------------------------------

func (o *opsWatch) checkDatabase() opsResult {
	pg := o.s.m.pg
	if pg == nil {
		return opsOff("sin PostgreSQL configurado")
	}
	ctx, cancel := context.WithTimeout(o.s.ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	if err := pg.PingContext(ctx); err != nil {
		return opsFail("la base de datos no responde: %v", err)
	}
	return opsOK("responde (%s)", time.Since(start).Round(time.Millisecond))
}

// checkBackups looks at the newest complete archive in NEXUS_BACKUP_DIR (the
// "backup" service's volume, mounted read-only).
func (o *opsWatch) checkBackups(now time.Time) opsResult {
	dir := envStr("NEXUS_BACKUP_DIR", "")
	if dir == "" {
		return opsOff("NEXUS_BACKUP_DIR no está configurado; no se vigilan las copias")
	}
	maxAge := time.Duration(envInt("NEXUS_BACKUP_MAX_AGE_HOURS", 26)) * time.Hour
	newest, count, err := newestBackup(dir)
	if err != nil {
		return opsFail("no se puede leer %s: %v", dir, err)
	}
	if count == 0 {
		if now.Sub(o.started) < time.Hour {
			return opsOK("esperando la primera copia")
		}
		return opsFail("no hay ninguna copia de seguridad en %s", dir)
	}
	age := now.Sub(newest)
	if age > maxAge {
		return opsFail("la última copia es de hace %s (máximo %s): el servicio de copias no está funcionando", opsAge(age), opsAge(maxAge))
	}
	return opsOK("última copia hace %s · %d guardadas", opsAge(age), count)
}

func newestBackup(dir string) (time.Time, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, 0, err
	}
	var newest time.Time
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".dump") {
			continue // *.dump.partial is a copy still being written
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		n++
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	return newest, n, nil
}

func (o *opsWatch) checkDisk() opsResult {
	minPct := envInt("NEXUS_DISK_MIN_FREE_PCT", 10)
	type place struct{ label, path string }
	places := []place{{"datos", o.dataDir}, {"copias", envStr("NEXUS_BACKUP_DIR", "")}}
	var parts []string
	failing := false
	for _, p := range places {
		if p.path == "" {
			continue
		}
		free, total, err := diskSpace(p.path)
		if err != nil {
			parts = append(parts, fmt.Sprintf("%s: no se pudo medir (%v)", p.label, err))
			continue
		}
		if total == 0 {
			continue
		}
		pct := int(free * 100 / total)
		parts = append(parts, fmt.Sprintf("%s: %d%% libre (%s)", p.label, pct, opsBytes(free)))
		if pct < minPct {
			failing = true
		}
	}
	if len(parts) == 0 {
		return opsOff("no se puede medir el disco en este sistema")
	}
	if failing {
		return opsFail("queda poco espacio (mínimo %d%%) · %s", minPct, strings.Join(parts, " · "))
	}
	return opsOK("%s", strings.Join(parts, " · "))
}

// checkErrors compares the request counters with the previous round.
func (o *opsWatch) checkErrors(now time.Time) opsResult {
	all, c5, panics := metrics.total.Load(), metrics.c5xx.Load(), metrics.panics.Load()
	o.mu.Lock()
	d5, dAll, dPanic := c5-o.base5xx, all-o.baseAll, panics-o.basePanic
	o.base5xx, o.baseAll, o.basePanic = c5, all, panics
	from := o.lastRun
	if from.IsZero() {
		from = o.started
	}
	o.mu.Unlock()
	window := opsAge(now.Sub(from))
	if dPanic > 0 {
		return opsFail("%d fallos internos (panic) en los últimos %s; ver el log del backend ([panic])", dPanic, window)
	}
	if d5 >= opsMin5xx && d5*opsMax5xxShare >= dAll {
		return opsFail("%d de %d respuestas fueron errores 5xx en los últimos %s; ver el log del backend ([http])", d5, dAll, window)
	}
	return opsOK("%d errores 5xx de %d respuestas en los últimos %s", d5, dAll, window)
}

// opsPublic reports a public deployment (HTTPS portal address): there, missing
// mail settings are a fault, not a development default.
func opsPublic() bool {
	return strings.HasPrefix(envStr("NEXUS_PORTAL_BASE_URL", ""), "https://")
}

func (o *opsWatch) checkMail(now time.Time) opsResult {
	if envStr("NEXUS_SMTP_HOST", "") == "" {
		if opsPublic() {
			return opsFail("NEXUS_SMTP_HOST no está configurado: invitaciones, verificaciones y recuperación de contraseña no llegan a nadie")
		}
		return opsOff("sin servidor de correo (desarrollo): los emails se escriben en el log")
	}
	pg := o.s.m.pg
	if pg == nil {
		return opsOff("sin PostgreSQL: no hay bandeja de salida")
	}
	failed, stuck, lastErr, err := pg.MailHealth(now.Add(-24*time.Hour), now.Add(-time.Hour))
	if err != nil {
		return opsFail("no se pudo leer la bandeja de salida: %v", err)
	}
	if failed > 0 || stuck > 0 {
		msg := fmt.Sprintf("%d emails no se pudieron entregar en 24 h; %d esperan desde hace más de 1 h", failed, stuck)
		if lastErr != "" {
			msg += " · último error: " + opsClip(lastErr, 160)
		}
		return opsFail("%s", msg)
	}
	return opsOK("entregando con normalidad")
}

func (o *opsWatch) checkAlertTarget() opsResult {
	if envStr("NEXUS_OPS_EMAIL", "") == "" {
		if opsPublic() {
			return opsFail("NEXUS_OPS_EMAIL no está configurado: las alertas solo quedan en el log")
		}
		return opsOff("NEXUS_OPS_EMAIL no está configurado (desarrollo): las alertas solo quedan en el log")
	}
	return opsOK("las alertas se envían a %s", envStr("NEXUS_OPS_EMAIL", ""))
}

// ---- alerts -------------------------------------------------------------------

type opsEvent struct {
	check opsCheck
	kind  string // fail | reminder | resolved
	was   string // detail while failing (resolved only)
}

func (o *opsWatch) alert(e opsEvent, all []opsCheck) {
	var subject, title, intro string
	switch e.kind {
	case "fail":
		subject, title = "[NEXUS] Alerta: "+e.check.Title, "Algo ha dejado de funcionar"
		intro = e.check.Title + ": " + e.check.Detail
	case "reminder":
		subject, title = "[NEXUS] Sigue fallando: "+e.check.Title, "Sigue sin resolverse"
		intro = e.check.Title + " falla desde " + e.check.Since.UTC().Format("2006-01-02 15:04") + " UTC: " + e.check.Detail
	default:
		subject, title = "[NEXUS] Resuelto: "+e.check.Title, "Vuelve a funcionar"
		intro = e.check.Title + " vuelve a estar bien: " + e.check.Detail
		if e.was != "" {
			intro += " (antes: " + e.was + ")"
		}
	}
	log.Printf("[ops-alert] %s %s: %s", e.kind, e.check.Name, e.check.Detail)
	items := make([]mailItem, 0, len(all))
	for _, c := range all {
		sev, label := "low", "OK"
		switch c.Status {
		case "fail":
			sev, label = "critical", "FALLA"
		case "off":
			sev, label = "unrated", "—"
		}
		items = append(items, mailItem{Sev: sev, SevLabel: label, Text: c.Title, Meta: opsClip(c.Detail, 200)})
	}
	o.notify(subject, mailContent{
		Lang: "es", Subject: subject, Preheader: intro, Title: title, Intro: []string{intro},
		Sections: []mailSection{{Title: "Estado actual", Items: items}},
		CTA:      "Ver el estado del servicio", CTALink: consoleLink("/system/ops"), Staff: true,
	})
}

// sendAlert mails NEXUS staff directly: never through the outbox, which may be
// the very thing that is failing.
func (o *opsWatch) sendAlert(subject string, c mailContent) {
	to := envStr("NEXUS_OPS_EMAIL", "")
	if to == "" {
		return
	}
	go func() {
		m, err := renderMail(c)
		if err == nil {
			err = deliverMail(to, m)
		}
		if err != nil {
			log.Printf("[ops-alert] could not mail %q to %s: %v", subject, to, err)
		}
	}()
}

// ---- console API --------------------------------------------------------------

func (o *opsWatch) status() map[string]any {
	o.mu.Lock()
	defer o.mu.Unlock()
	var checked any
	if !o.lastRun.IsZero() {
		checked = o.lastRun
	}
	return map[string]any{
		"checks":        o.snapshotLocked(),
		"checked_at":    checked,
		"interval":      opsInterval().String(),
		"ops_email_set": envStr("NEXUS_OPS_EMAIL", "") != "",
		"smtp":          envStr("NEXUS_SMTP_HOST", "") != "",
	}
}

// GET /api/admin/ops/status — the last round's results.
func (s *Server) adminOpsStatus(w http.ResponseWriter, r *http.Request) {
	if s.ops.lastRunZero() {
		s.ops.round(time.Now())
	}
	writeJSON(w, 200, s.ops.status())
}

// POST /api/admin/ops/check — run every check now.
func (s *Server) adminOpsCheck(w http.ResponseWriter, r *http.Request) {
	s.ops.round(time.Now())
	writeJSON(w, 200, s.ops.status())
}

func (o *opsWatch) lastRunZero() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lastRun.IsZero()
}

// ---- formatting ---------------------------------------------------------------

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(envStr(key, "")); err == nil && n > 0 {
		return n
	}
	return def
}

func opsAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d s", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%.1f h", d.Hours())
	default:
		return fmt.Sprintf("%d días", int(d.Hours()/24))
	}
}

func opsBytes(b uint64) string {
	const gb = 1 << 30
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/gb)
	}
	return fmt.Sprintf("%d MB", b>>20)
}

func opsClip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
