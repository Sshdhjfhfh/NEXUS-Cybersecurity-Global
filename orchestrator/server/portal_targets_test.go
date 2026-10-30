package server

import (
	"context"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/db"
)

func TestNormalizeTarget(t *testing.T) {
	ok := map[string][2]string{
		"Example.COM":                     {"domain", "example.com"},
		"https://app.example.com/login?x": {"domain", "app.example.com"},
		"portal.example.gob.":             {"domain", "portal.example.gob"},
		"xn--bcher-kva.example":           {"domain", "xn--bcher-kva.example"},
		"203.0.113.7":                     {"ip", "203.0.113.7"},
		"10.0.0.5":                        {"ip", "10.0.0.5"}, // allowed, but only manual approval can verify it
		"[2001:db8::1]":                   {"ip", "2001:db8::1"},
	}
	for in, want := range ok {
		kind, value, err := normalizeTarget(in)
		if err != nil || kind != want[0] || value != want[1] {
			t.Errorf("normalizeTarget(%q) = %q %q %v, want %v", in, kind, value, err, want)
		}
	}
	for _, bad := range []string{"", "localhost", "*.example.com", "example.com:8443", "user@example.com",
		"exa mple.com", "-bad.example.com", "example.123", "127.0.0.1", "0.0.0.0", "::1", "224.0.0.1",
		strings.Repeat("a", 254) + ".com"} {
		if _, _, err := normalizeTarget(bad); err == nil {
			t.Errorf("normalizeTarget(%q) accepted", bad)
		}
	}
}

func TestPublicAddr(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254",
		"100.64.0.1", "0.1.2.3", "::1", "fc00::1", "fe80::1", "::ffff:127.0.0.1", "198.51.100.1"} {
		if publicAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must not be public", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "2606:4700::1111"} {
		if !publicAddr(netip.MustParseAddr(s)) {
			t.Errorf("%s must be public", s)
		}
	}
}

// The HTTP proof check must never reach internal addresses (SSRF).
func TestCheckTargetHTTPRefusesInternal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, host := range []string{"127.0.0.1", "169.254.169.254", "10.0.0.1", "::1"} {
		err := checkTargetHTTP(ctx, host, "nexus-verify-x")
		if err == nil || !strings.Contains(err.Error(), "no es pública") {
			t.Errorf("%s: want refusal, got %v", host, err)
		}
	}
}

func TestEngineScopeIsNarrowest(t *testing.T) {
	cases := []struct {
		t          db.PortalTarget
		kind, want string
	}{
		{db.PortalTarget{Kind: "domain", Value: "example.com", Method: "dns"}, "root_domain", "example.com"},
		{db.PortalTarget{Kind: "domain", Value: "example.com", Method: "http"}, "subdomain", "example.com"},
		{db.PortalTarget{Kind: "domain", Value: "app.example.com", Method: "dns"}, "subdomain", "app.example.com"},
		{db.PortalTarget{Kind: "domain", Value: "example.com", Method: "manual"}, "subdomain", "example.com"},
		{db.PortalTarget{Kind: "ip", Value: "203.0.113.7", Method: "http"}, "ip", "203.0.113.7"},
	}
	for _, c := range cases {
		k, v := engineScopeForTarget(&c.t)
		if k != c.kind || v != c.want {
			t.Errorf("%+v -> %s %s, want %s %s", c.t, k, v, c.kind, c.want)
		}
	}
}

func TestRoETermsAndHash(t *testing.T) {
	if len(roeTerms["es"]) != len(roeTerms["en"]) || len(roeTerms["es"]) == 0 {
		t.Fatal("ES/EN terms must have the same clauses")
	}
	if roeHash("es") == roeHash("en") || roeHash("es") != roeHash("es") || len(roeHash("es")) != 64 {
		t.Fatal("hash must be stable per language and differ across languages")
	}
}

func TestAuthorizationProblems(t *testing.T) {
	if p := authorizationProblems(nil); len(p) != 1 {
		t.Fatalf("nil auth: %v", p)
	}
	future := time.Now().UTC().AddDate(0, 0, 10).Format("2006-01-02")
	past := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	a := &db.RequestAuth{
		Targets: []*db.PortalTarget{{Value: "a.example.com", Status: "verified"}, {Value: "b.example.com", Status: "pending"}},
	}
	if p := authorizationProblems(a); len(p) != 2 {
		t.Fatalf("pending target + no RoE: %v", p)
	}
	a.Targets[1].Status = "verified"
	a.RoE = &db.RoE{WindowEnd: future}
	if p := authorizationProblems(a); len(p) != 0 {
		t.Fatalf("ready: %v", p)
	}
	a.RoE.WindowEnd = past
	if p := authorizationProblems(a); len(p) != 1 {
		t.Fatalf("ended window: %v", p)
	}
}
