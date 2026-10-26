package detectbench

import (
	"context"
	"testing"

	"github.com/nexus-ia/nexus/evolve"
)

func TestFixturesHaveNoExactReplayAndDigestsCoverContent(t *testing.T) {
	seen := map[string]bool{}
	for generation := 0; generation < 3; generation++ {
		b, err := NewGeneration(generation)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range b.Cases() {
			if len(c.InputDigest) != 64 || seen[c.InputDigest] {
				t.Fatal("identical input reused across split/generation")
			}
			seen[c.InputDigest] = true
		}
		prior := b.Digest()
		id := b.Cases()[0].ID
		b.events[id]["independent-evidence"] = "changed"
		if b.fingerprint() == prior {
			t.Fatal("digest omits event input")
		}
		delete(b.events[id], "independent-evidence")
		b.rulesets["v1"][0].Description = "changed"
		if b.fingerprint() == prior {
			t.Fatal("digest omits rule contents")
		}
	}
}

// TestVerifiedImprovementPromoted shows the honest evolution loop: a ruleset
// (v2) that genuinely detects more attacks than the incumbent (v1), without
// regressing on benign traffic, is promoted — and only after passing an
// held-out synthetic fixture split with the configured statistical gate. The
// synthetic families themselves are not independent population measurements.
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
