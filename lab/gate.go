// Package lab provides laboratory-only authorization and isolated scenarios.
// It never opens a network connection or changes an immutable safety module.
package lab

import (
	"context"
	"fmt"

	"github.com/nexus-ia/nexus/lab/network"
)

type Action struct {
	AgentID string `json:"agent_id"`
	TaskID  string `json:"task_id"`
	Name    string `json:"name"`
	Target  string `json:"target"`
}

type Check func(context.Context, Action) error

// Checks are adapters supplied by the host for ANS/NHI identity, OPSEC,
// ethical_boundary and ABM. There is no permissive production default.
type Checks struct{ Identity, OPSEC, Ethics, Monitor Check }

type Gate struct {
	scope  network.Scope
	checks Checks
}

func NewGate(scope network.Scope, checks Checks) (*Gate, error) {
	if len(scope.CIDRs()) == 0 {
		return nil, fmt.Errorf("lab scope is required")
	}
	if checks.Identity == nil || checks.OPSEC == nil || checks.Ethics == nil || checks.Monitor == nil {
		return nil, fmt.Errorf("identity, OPSEC, ethics and ABM checks are required")
	}
	return &Gate{scope: scope, checks: checks}, nil
}

func (g *Gate) Ready() bool {
	return g != nil && len(g.scope.CIDRs()) > 0 && g.checks.Identity != nil && g.checks.OPSEC != nil && g.checks.Ethics != nil && g.checks.Monitor != nil
}

func (g *Gate) Authorize(ctx context.Context, a Action) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if g == nil || g.checks.Identity == nil || g.checks.OPSEC == nil || g.checks.Ethics == nil || g.checks.Monitor == nil {
		return fmt.Errorf("lab authorization is not configured")
	}
	if a.AgentID == "" || a.TaskID == "" || a.Name == "" {
		return fmt.Errorf("agent, task and action are required")
	}
	if !g.scope.Allows(a.Target) {
		return fmt.Errorf("target %q is outside the laboratory", a.Target)
	}
	for _, step := range []struct {
		name  string
		check Check
	}{
		{"identity", g.checks.Identity}, {"ABM", g.checks.Monitor}, {"OPSEC", g.checks.OPSEC}, {"ethics", g.checks.Ethics},
	} {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := step.check(ctx, a); err != nil {
			return fmt.Errorf("%s denied action: %w", step.name, err)
		}
	}
	return ctx.Err()
}

type Authorizer interface {
	Authorize(context.Context, Action) error
}
