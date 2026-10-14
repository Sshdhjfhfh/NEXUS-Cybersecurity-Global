package lab

import (
	"context"
	"errors"
	"github.com/nexus-ia/nexus/lab/network"
	"testing"
)

func TestAuditedGateRecordsAllowDenyAndCanceledDecisions(t *testing.T) {
	scope, err := network.NewScope("10.20.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	allow := func(context.Context, Action) error { return nil }
	deny := errors.New("OPSEC denied")
	calls := 0
	var recorded error
	recorder := func(ctx context.Context, a Action, decision error) error {
		if ctx.Err() != nil {
			t.Fatal("audit inherited cancellation")
		}
		calls++
		recorded = decision
		return nil
	}
	gate, err := NewAuditedGate(scope, Checks{Identity: allow, Monitor: allow, OPSEC: allow, Ethics: allow}, recorder)
	if err != nil {
		t.Fatal(err)
	}
	action := Action{AgentID: "test", TaskID: "7", Name: "blue.analyze", Target: "10.20.0.3"}
	if err := gate.Authorize(t.Context(), action); err != nil || calls != 1 || recorded != nil {
		t.Fatal("allowed decision missing", err)
	}
	gate.checks.OPSEC = func(context.Context, Action) error { return deny }
	if err := gate.Authorize(t.Context(), action); !errors.Is(err, deny) || !errors.Is(recorded, deny) {
		t.Fatal("denial missing", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := gate.Authorize(ctx, action); !errors.Is(err, context.Canceled) || !errors.Is(recorded, context.Canceled) {
		t.Fatal("canceled decision missing", err)
	}
	gate.checks.OPSEC = allow
	gate.audit = func(context.Context, Action, error) error { return errors.New("storage unavailable") }
	if err := gate.Authorize(t.Context(), action); err == nil {
		t.Fatal("audit failure allowed an action")
	}
}
