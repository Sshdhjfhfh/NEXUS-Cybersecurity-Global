package engagement

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nexus-ia/nexus/lab"
)

func TestAuthorizerEnforcesScope(t *testing.T) {
	a, err := NewAuthorizer(Scope{
		Engagement:        "acme-2026-q4",
		AuthorizedTargets: []string{"https://app.client.test", "10.0.0.5"},
	})
	if err != nil {
		t.Fatalf("new authorizer: %v", err)
	}
	inScope := []string{
		"https://app.client.test/login",
		"http://app.client.test:8443/admin",
		"10.0.0.5:22",
	}
	for _, tg := range inScope {
		if err := a.Authorize(context.Background(), lab.Action{AgentID: "nexus-scan", TaskID: "acme-2026-q4", Name: "red.httpx", Target: tg}); err != nil {
			t.Errorf("in-scope %q should be allowed: %v", tg, err)
		}
	}
	outOfScope := []string{
		"https://evil.test/",
		"8.8.8.8",
		"https://notapp.client.test/",
		"",
	}
	for _, tg := range outOfScope {
		if err := a.Authorize(context.Background(), lab.Action{AgentID: "nexus-scan", TaskID: "acme-2026-q4", Name: "red.nuclei", Target: tg}); err == nil {
			t.Errorf("out-of-scope %q must be denied", tg)
		}
	}
}

func TestNewAuthorizerRejectsEmpty(t *testing.T) {
	if _, err := NewAuthorizer(Scope{Engagement: "e"}); err == nil {
		t.Error("expected error when scope has no authorized targets")
	}
}

func TestLoadScope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.json")
	good := `{"engagement":"acme-2026-q4","client":"ACME","authorized_targets":["app.client.test","10.0.0.5"]}`
	if err := os.WriteFile(path, []byte(good), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadScope(path)
	if err != nil || s.Engagement != "acme-2026-q4" || len(s.AuthorizedTargets) != 2 {
		t.Fatalf("load good scope: %v %+v", err, s)
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"engagement":"","authorized_targets":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScope(bad); err == nil {
		t.Error("expected error for scope with no engagement/targets")
	}
}
