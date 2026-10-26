// Nexus IA v6.0 — Absorbido de norma/coordinator
// Upstream: github.com/Autumn-27/norma/coordinator
// Licencia: AGPL-3.0

package coordinator

import (
	"context"
	"errors"
	"iter"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/nexus-ia/nexus/core/llm"
	"github.com/nexus-ia/nexus/core/permission"
	"github.com/nexus-ia/nexus/core/tool"
)

type startCounter struct{ starts atomic.Int32 }

func (*startCounter) PreToolUse(context.Context, string, []byte) (bool, string, []byte) {
	return false, "", nil
}
func (*startCounter) PostToolUse(context.Context, string, []byte, []byte, bool) {}
func (*startCounter) Stop(context.Context, []llm.Message) (bool, []string, string) {
	return false, nil, ""
}
func (h *startCounter) FireSubagentStart(context.Context, string) { h.starts.Add(1) }

func TestCancelledCoordinatorDoesNotStartSubagents(t *testing.T) {
	hooks := &startCounter{}
	c := &Config{Provider: &echoProvider{}, Hooks: hooks, MaxParallel: 1}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	results := c.RunParallel(ctx, []string{"one", "two", "three"})
	if hooks.starts.Load() != 0 {
		t.Fatalf("started %d subagents after cancellation", hooks.starts.Load())
	}
	for i, r := range results {
		if r.Task != []string{"one", "two", "three"}[i] || !errors.Is(r.Err, context.Canceled) {
			t.Fatalf("cancelled result lost identity/cause: %+v", r)
		}
	}
	res, err := c.AgentTool().Call(ctx, []byte(`{"agent_type":"worker","prompt":"cancelled"}`), &tool.ToolContext{})
	if err != nil || !res.IsError || hooks.starts.Load() != 0 {
		t.Fatalf("cancelled direct subagent started: %v %+v", err, res)
	}
}

// echoProvider answers each worker with text derived from the task's last user
// message, so we can verify distinct workers ran.
type echoProvider struct{ count int32 }

func (p *echoProvider) Stream(_ context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	atomic.AddInt32(&p.count, 1)
	last := ""
	if n := len(req.Messages); n > 0 {
		last = req.Messages[n-1].Text()
	}
	return func(yield func(llm.StreamEvent, error) bool) {
		yield(llm.StreamEvent{Type: llm.SETextDelta, Text: "did: " + last}, nil)
		yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"}, nil)
		yield(llm.StreamEvent{Type: llm.SEMessageStop}, nil)
	}
}

func (p *echoProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return accumulateStream(ctx, p, req)
}

// accumulateStream drains a provider's own Stream to satisfy the non-streaming
// Complete method for test mocks.
func accumulateStream(ctx context.Context, p llm.Provider, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	acc := llm.NewAccumulator()
	for ev, err := range p.Stream(ctx, req) {
		if err != nil {
			return llm.Message{}, "", llm.Usage{}, err
		}
		acc.Add(ev)
	}
	return acc.Message(), acc.StopReason, acc.Usage, nil
}

func TestRunParallelFansOut(t *testing.T) {
	prov := &echoProvider{}
	c := &Config{
		Provider:       prov,
		WorkerTools:    tool.NewRegistry(tool.DefaultTools()...),
		PermissionMode: permission.ModeBypass,
	}
	tasks := []string{"task A", "task B", "task C"}
	results := c.RunParallel(context.Background(), tasks)
	if len(results) != 3 {
		t.Fatalf("results=%d", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("task %d err: %v", i, r.Err)
		}
		if !strings.Contains(r.Output, tasks[i]) {
			t.Fatalf("result %d mismatched: task=%q output=%q", i, tasks[i], r.Output)
		}
	}
	if atomic.LoadInt32(&prov.count) != 3 {
		t.Fatalf("expected 3 worker runs, got %d", prov.count)
	}
}

func TestAgentToolAndSystemSegment(t *testing.T) {
	c := &Config{
		Provider:    &echoProvider{},
		WorkerTools: tool.NewRegistry(tool.NewRead(), tool.NewBash()),
	}
	at := c.AgentTool()
	if at.Name() != "Agent" {
		t.Fatalf("agent tool name=%q", at.Name())
	}
	seg := c.SystemSegment()
	if !strings.Contains(seg, "Coordinator mode") || !strings.Contains(seg, "Read") || !strings.Contains(seg, "Bash") {
		t.Fatalf("system segment missing pieces:\n%s", seg)
	}
	// The Agent tool actually dispatches a worker.
	res, _ := at.Call(context.Background(), []byte(`{"agent_type":"worker","prompt":"find bugs"}`), &tool.ToolContext{})
	if !strings.Contains(res.Flatten(), "find bugs") {
		t.Fatalf("worker dispatch: %q", res.Flatten())
	}
}
