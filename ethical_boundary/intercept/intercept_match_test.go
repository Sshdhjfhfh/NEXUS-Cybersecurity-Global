package intercept

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/db"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	dsn, _, err := db.DSN()
	if err != nil {
		t.Skipf("no database config (%v) — skipping", err)
	}
	d, err := db.Open(dsn)
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	t.Cleanup(func() { d.Close() }) // first registered → runs after the row cleanups
	return d
}

// Rules are evaluated by priority (highest first), match by tool name or input,
// as substring or regex; disabled rules and rules with a broken regex never match.
func TestMatchPriorityAndTargets(t *testing.T) {
	d := openTestDB(t)
	tag := fmt.Sprintf("zz-intercept-test-%d", time.Now().UnixNano())
	mk := func(name, target, typ, pattern, action string, prio int, enabled bool) db.InterceptRule {
		r, err := d.CreateInterceptRule(tag+name, target, typ, pattern, action, "", prio, enabled, false, 0, "deny")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = d.DeleteInterceptRule(r.ID) })
		return r
	}
	// Patterns carry the tag so they cannot fire on any other rule's traffic.
	allow := mk("-allow", "tool_input", "string", tag+"-safe", "allow", 100000, true)
	deny := mk("-deny", "tool_input", "regex", `(?i)`+tag+`-(safe|drop)`, "deny", 99999, true)
	mk("-off", "tool_input", "string", tag+"-drop", "allow", 100001, false)
	mk("-badre", "tool_input", "regex", tag+"-(", "allow", 100002, true)
	ask := mk("-ask", "tool_name", "string", tag+"Tool", "ask", 99998, true)

	ic := New(d)
	if dec, ok := ic.Match("Bash", []byte(`{"command":"`+tag+`-safe"}`)); !ok || dec.RuleID != allow.ID || dec.Action != "allow" {
		t.Fatalf("highest-priority allow must win: ok=%v dec=%+v", ok, dec)
	}
	dec, ok := ic.Match("Bash", []byte(`{"command":"`+tag+`-DROP"}`))
	if !ok || dec.RuleID != deny.ID || dec.Action != "deny" {
		t.Fatalf("regex deny not matched (disabled/broken rules must be skipped): ok=%v dec=%+v", ok, dec)
	}
	if dec.Message == "" || dec.ConfigDigest == "" {
		t.Errorf("deny decision lacks default message or config digest: %+v", dec)
	}
	if dec, ok := ic.Match(tag+"Tool", []byte(`{}`)); !ok || dec.RuleID != ask.ID || dec.Action != "ask" {
		t.Fatalf("tool_name rule not matched: ok=%v dec=%+v", ok, dec)
	}
	// The cache is only refreshed on Invalidate.
	extra := mk("-late", "tool_input", "string", tag+"-late", "deny", 99997, true)
	if _, ok := ic.Match("Bash", []byte(tag+"-late")); ok {
		t.Fatal("rule visible before Invalidate — cache not in effect")
	}
	ic.Invalidate()
	if dec, ok := ic.Match("Bash", []byte(tag+"-late")); !ok || dec.RuleID != extra.ID {
		t.Fatalf("rule not visible after Invalidate: ok=%v dec=%+v", ok, dec)
	}
}

func TestRuleMatchesUnit(t *testing.T) {
	r := compiledRule{InterceptRule: db.InterceptRule{MatchTarget: "nope", Pattern: "x"}}
	if ruleMatches(r, "x", []byte("x")) {
		t.Error("unknown match target must never match")
	}
	r = compiledRule{InterceptRule: db.InterceptRule{MatchTarget: "tool_input", MatchType: "regex", Pattern: "x"}}
	if ruleMatches(r, "", []byte("x")) {
		t.Error("regex rule without a compiled regex must not match")
	}
	if defaultMessage("allow", "r") != "" || defaultMessage("deny", "r") == "" || defaultMessage("ask", "r") == "" {
		t.Error("default messages: allow has none, deny/ask do")
	}
	for _, a := range []string{"allow", "deny", "ask"} {
		if judgeActionLabel(a) == a {
			t.Errorf("no label for %q", a)
		}
	}
	if judgeActionLabel("other") != "other" {
		t.Error("unknown action label must pass through")
	}
}

// Without a wired reviewer the fallback judge never decides (callers then allow),
// regardless of the enabled setting.
func TestJudgeUnwiredDoesNotDecide(t *testing.T) {
	d := openTestDB(t)
	if _, ok := New(d).Judge(context.Background(), "Bash", []byte(`{"command":"id"}`)); ok {
		t.Fatal("judge decided without a reviewer")
	}
}

func TestDecideUnknownPendingAndContextHelpers(t *testing.T) {
	d := openTestDB(t)
	ic := New(d)
	if err := ic.Decide(-1, true); !errors.Is(err, ErrAlreadyDecided) {
		t.Fatalf("deciding a missing request: %v", err)
	}
	ctx := WithConvID(context.Background(), 42)
	if ConvIDFromContext(ctx) != 42 || ConvIDFromContext(context.Background()) != 0 {
		t.Error("conversation id round-trip")
	}
	ctx = WithTaskContext(ctx, "7", "worker · #1", nil)
	if id, name := taskInfoFromCtx(ctx); id != "7" || name != "worker · #1" {
		t.Errorf("task info = %q %q", id, name)
	}
	if taskEmitFromCtx(ctx) != nil {
		t.Error("nil emit must not be stored")
	}
}

// The pending manager hands a decision to exactly the waiter registered for it.
func TestPendingManager(t *testing.T) {
	p := newPendingManager()
	ch := p.add(5)
	p.resolve(6, true) // unknown id: no-op
	p.resolve(5, true)
	select {
	case v := <-ch:
		if !v {
			t.Fatal("wrong decision delivered")
		}
	default:
		t.Fatal("decision not delivered")
	}
	p.add(9)
	p.remove(9)
	p.resolve(9, true) // removed: must not block or panic
}
