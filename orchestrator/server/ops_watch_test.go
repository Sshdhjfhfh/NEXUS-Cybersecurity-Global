package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type sentAlert struct{ subject, body string }

func testOpsWatch(t *testing.T) (*opsWatch, *[]sentAlert) {
	t.Helper()
	for _, k := range []string{"NEXUS_SMTP_HOST", "NEXUS_OPS_EMAIL", "NEXUS_PORTAL_BASE_URL", "NEXUS_BACKUP_DIR"} {
		t.Setenv(k, "")
	}
	o := newOpsWatch(&Server{m: &Manager{}, ctx: context.Background()}, t.TempDir())
	var sent []sentAlert
	o.notify = func(subject string, c mailContent) {
		var b strings.Builder
		b.WriteString(strings.Join(c.Intro, " "))
		for _, s := range c.Sections {
			for _, it := range s.Items {
				b.WriteString(" | " + it.SevLabel + " " + it.Text)
			}
		}
		sent = append(sent, sentAlert{subject, b.String()})
	}
	return o, &sent
}

func opsStatusOf(cs []opsCheck, name string) opsCheck {
	for _, c := range cs {
		if c.Name == name {
			return c
		}
	}
	return opsCheck{}
}

func writeDump(t *testing.T, dir, name string, age time.Duration) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(p, when, when); err != nil {
		t.Fatal(err)
	}
}

func TestOpsBackupAlertLifecycle(t *testing.T) {
	o, sent := testOpsWatch(t)
	dir := t.TempDir()
	t.Setenv("NEXUS_BACKUP_DIR", dir)
	now := time.Now()
	o.started = now.Add(-2 * time.Hour)

	// No archive at all (a half-written one does not count).
	writeDump(t, dir, "nexus-20261101-000000.dump.partial", time.Minute)
	cs := o.round(now)
	if c := opsStatusOf(cs, "backups"); c.Status != "fail" || !strings.Contains(c.Detail, "no hay ninguna copia") {
		t.Fatalf("backups = %+v, want fail with no archive", c)
	}
	if len(*sent) != 1 || !strings.Contains((*sent)[0].subject, "Alerta: Copias de seguridad") {
		t.Fatalf("alerts = %+v, want one failure alert", *sent)
	}
	if !strings.Contains((*sent)[0].body, "FALLA Copias de seguridad") {
		t.Fatalf("alert body lacks the status list: %q", (*sent)[0].body)
	}

	// Only a stale archive: still failing, no repeat within the reminder period.
	writeDump(t, dir, "nexus-20261031-000000.dump", 30*time.Hour)
	cs = o.round(now.Add(5 * time.Minute))
	if c := opsStatusOf(cs, "backups"); c.Status != "fail" || !strings.Contains(c.Detail, "última copia es de hace") {
		t.Fatalf("backups = %+v, want stale-copy failure", c)
	}
	if len(*sent) != 1 {
		t.Fatalf("alerts = %d, want no repeat before the reminder period", len(*sent))
	}

	// A day later it is still failing: one reminder.
	o.round(now.Add(opsRemindEvery + 10*time.Minute))
	if len(*sent) != 2 || !strings.Contains((*sent)[1].subject, "Sigue fallando") {
		t.Fatalf("alerts = %+v, want a reminder", *sent)
	}

	// A fresh archive: recovered, one "resolved" mail.
	writeDump(t, dir, "nexus-20261102-000000.dump", time.Minute)
	cs = o.round(time.Now())
	if c := opsStatusOf(cs, "backups"); c.Status != "ok" || !strings.Contains(c.Detail, "2 guardadas") {
		t.Fatalf("backups = %+v, want ok with 2 archives", c)
	}
	if len(*sent) != 3 || !strings.Contains((*sent)[2].subject, "Resuelto: Copias de seguridad") {
		t.Fatalf("alerts = %+v, want a resolved mail", *sent)
	}
	o.round(time.Now())
	if len(*sent) != 3 {
		t.Fatalf("alerts = %d, want silence while healthy", len(*sent))
	}
}

func TestOpsBackupGraceAfterStart(t *testing.T) {
	o, sent := testOpsWatch(t)
	t.Setenv("NEXUS_BACKUP_DIR", t.TempDir())
	cs := o.round(time.Now())
	if c := opsStatusOf(cs, "backups"); c.Status != "ok" || !strings.Contains(c.Detail, "esperando") {
		t.Fatalf("backups = %+v, want waiting for the first copy", c)
	}
	if len(*sent) != 0 {
		t.Fatalf("alerts = %+v, want none right after start", *sent)
	}
}

func TestOpsServerErrors(t *testing.T) {
	o, sent := testOpsWatch(t)
	now := time.Now()
	o.round(now) // baseline
	metrics.total.Add(100)
	metrics.c5xx.Add(20)
	cs := o.round(now.Add(5 * time.Minute))
	if c := opsStatusOf(cs, "errors"); c.Status != "fail" || !strings.Contains(c.Detail, "20 de 100") {
		t.Fatalf("errors = %+v, want 20 of 100 failing", c)
	}
	if len(*sent) != 1 || !strings.Contains((*sent)[0].subject, "Errores del servidor") {
		t.Fatalf("alerts = %+v, want a server-error alert", *sent)
	}
	// A few 5xx among many requests is not an incident.
	metrics.total.Add(1000)
	metrics.c5xx.Add(3)
	cs = o.round(now.Add(10 * time.Minute))
	if c := opsStatusOf(cs, "errors"); c.Status != "ok" {
		t.Fatalf("errors = %+v, want ok", c)
	}
	if len(*sent) != 2 || !strings.Contains((*sent)[1].subject, "Resuelto") {
		t.Fatalf("alerts = %+v, want resolved", *sent)
	}
	metrics.panics.Add(1)
	if c := opsStatusOf(o.round(now.Add(15*time.Minute)), "errors"); c.Status != "fail" || !strings.Contains(c.Detail, "panic") {
		t.Fatalf("errors = %+v, want a panic failure", c)
	}
}

func TestOpsMailSettings(t *testing.T) {
	o, _ := testOpsWatch(t)
	cs := o.round(time.Now())
	for _, n := range []string{"mail", "alerts", "backups", "database"} {
		if c := opsStatusOf(cs, n); c.Status != "off" {
			t.Fatalf("%s = %+v, want off in development", n, c)
		}
	}
	// A public deployment without mail settings is a fault.
	t.Setenv("NEXUS_PORTAL_BASE_URL", "https://nexus.example.test/portal")
	cs = o.round(time.Now())
	if c := opsStatusOf(cs, "mail"); c.Status != "fail" || !strings.Contains(c.Detail, "NEXUS_SMTP_HOST") {
		t.Fatalf("mail = %+v, want fail without SMTP in production", c)
	}
	if c := opsStatusOf(cs, "alerts"); c.Status != "fail" {
		t.Fatalf("alerts = %+v, want fail without NEXUS_OPS_EMAIL in production", c)
	}
	if c := opsStatusOf(cs, "disk"); c.Status != "ok" || !strings.Contains(c.Detail, "datos:") {
		t.Fatalf("disk = %+v, want the data dir measured", c)
	}
}
