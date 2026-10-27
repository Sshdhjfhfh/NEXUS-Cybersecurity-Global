package server

import (
	"testing"
	"time"
)

func TestLoginBruteLocksAfterMaxFails(t *testing.T) {
	g := &bruteGuard{byIP: map[string]*loginAttempts{}, maxFails: 3, window: time.Minute, lockout: time.Minute}
	ip := "203.0.113.5"
	for i := 0; i < 2; i++ {
		g.fail(ip)
		if blocked, _ := g.blocked(ip); blocked {
			t.Fatalf("should not be locked after %d fails", i+1)
		}
	}
	g.fail(ip) // 3rd failure -> lock
	if blocked, _ := g.blocked(ip); !blocked {
		t.Fatal("should be locked after reaching maxFails")
	}
	// A successful login clears the lock for that IP.
	g.success(ip)
	if blocked, _ := g.blocked(ip); blocked {
		t.Fatal("success must clear the lockout")
	}
}

func TestCorsAllowlistEmptyByDefault(t *testing.T) {
	t.Setenv("NEXUS_CORS_ORIGINS", "")
	if len(corsAllowlist()) != 0 {
		t.Fatal("CORS allowlist must be empty (no wildcard) by default")
	}
	t.Setenv("NEXUS_CORS_ORIGINS", "https://app.example.com, https://admin.example.com")
	m := corsAllowlist()
	if !m["https://app.example.com"] || !m["https://admin.example.com"] {
		t.Fatalf("allowlist not parsed: %v", m)
	}
}
