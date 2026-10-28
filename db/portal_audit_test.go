package db

import (
	"testing"
	"time"
)

func TestAuditHashDeterministicAndSensitive(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	org := int64(7)
	base := AuditEntry{
		OrgID: &org, Actor: "a@x.io", ActorType: "portal_user",
		Action: "auth.login", Target: "user:1", Detail: "", IP: "203.0.113.1",
		CreatedAt: ts,
	}
	h1 := auditHash("PREV", base)
	h2 := auditHash("PREV", base)
	if h1 != h2 {
		t.Fatal("hash must be deterministic for the same inputs")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(h1))
	}

	// Changing the previous hash must change the result (chain linkage).
	if auditHash("OTHER", base) == h1 {
		t.Fatal("hash must depend on the previous hash")
	}
	// Changing any bound field must change the result (tamper sensitivity).
	mut := base
	mut.Action = "auth.login_failed"
	if auditHash("PREV", mut) == h1 {
		t.Fatal("hash must depend on action")
	}
	mut = base
	mut.Detail = "tampered"
	if auditHash("PREV", mut) == h1 {
		t.Fatal("hash must depend on detail")
	}
	mut = base
	other := int64(8)
	mut.OrgID = &other
	if auditHash("PREV", mut) == h1 {
		t.Fatal("hash must depend on org id")
	}
	mut = base
	mut.CreatedAt = ts.Add(time.Second)
	if auditHash("PREV", mut) == h1 {
		t.Fatal("hash must depend on timestamp")
	}
}
