package detectbench

import (
	"context"
	"testing"

	"github.com/nexus-ia/nexus/evolve"
)

// TestVerifiedImprovementPromoted shows the honest evolution loop: a ruleset
// (v2) that genuinely detects more attacks than the incumbent (v1), without
// regressing on benign traffic, is promoted — and only after passing an
// independent holdout split with statistical significance.
func TestVerifiedImprovementPromoted(t *testing.T) {
	b, err := New()
	if err != nil {
		t.Fatalf("build benchmark: %v", err)
	}
	e, err := evolve.New(evolve.Config{
		Baseline:   "v1",
		Candidates: []string{"v2"},
		Policy:     evolve.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	rep, err := e.Run(context.Background(), b)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if rep.Decision != "promoted" || rep.Active != "v2" {
		t.Fatalf("expected v2 promoted, got decision=%q active=%q reason=%q (holdout gain=%.2f p=%.4f)",
			rep.Decision, rep.Active, rep.Reason, rep.Holdout.Gain, rep.Holdout.PValue)
	}
}

// TestNoisyCandidateRejected shows the safety side: a ruleset that "detects
// more" but causes false positives on critical benign cases is NOT promoted.
// This is the regression gate that keeps an agent from getting worse while
// claiming to improve.
func TestNoisyCandidateRejected(t *testing.T) {
	b, err := New()
	if err != nil {
		t.Fatalf("build benchmark: %v", err)
	}
	e, err := evolve.New(evolve.Config{
		Baseline:   "v1",
		Candidates: []string{"noisy"},
		Policy:     evolve.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	rep, err := e.Run(context.Background(), b)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if rep.Decision == "promoted" || rep.Active != "v1" {
		t.Fatalf("noisy ruleset must not be promoted, got decision=%q active=%q", rep.Decision, rep.Active)
	}
}
