package recall

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/nexus-ia/nexus/core/memory"
	"github.com/nexus-ia/nexus/lab/benchmark/recallfixture"
)

func TestInsufficientEvidencePreservesBaselineOrder(t *testing.T) {
	// Equal scores and timestamps preserve the caller's order in RankLegacy.
	// Building an adaptive index must not silently change that fallback.
	notes := []memory.Memory{{Name: "Zeta Beta"}, {Name: "Beta"}}
	want := memory.RankLegacy(notes, "Beta", 1)
	a := NewAdaptive(t.TempDir(), nil)
	got := a.Rank(context.Background(), notes, "Beta", 1)
	if a.Report().Decision != "insufficient" {
		t.Fatal("fixture must leave the baseline active")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("baseline order changed without a promotion: got %+v, want %+v", got, want)
	}
}

func TestPromotedStrategyFallsBackForUnsupportedTitle(t *testing.T) {
	name := "Referencia " + strings.Repeat("x", 257)
	notes := append(recallfixture.Memories(),
		memory.Memory{Name: name},
		memory.Memory{Name: "Referencias " + strings.Repeat("x", 257), Description: name, Content: name},
	)
	a := NewAdaptive(t.TempDir(), nil)
	got := a.Rank(context.Background(), notes, name, 1)
	if a.Report().Decision != "promoted" {
		t.Fatal("fixture must promote a strategy for supported titles")
	}
	want := memory.RankLegacy(notes, name, 1)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unsupported title used promoted ranking: got %+v, want %+v", got, want)
	}
}

func TestRolloutExcludesTitlesRejectedByEvaluator(t *testing.T) {
	for _, name := range []string{"de la", strings.Repeat("x", 257)} {
		t.Run(name[:min(len(name), 20)], func(t *testing.T) {
			idx, err := NewIndex([]memory.Memory{{Name: name}})
			if err != nil {
				t.Fatal(err)
			}
			if len(NewBenchmark(idx).Cases()) != 0 {
				t.Fatal("fixture must be outside the evaluator's supported titles")
			}
			if idx.Covers(name) {
				t.Fatal("rollout includes a title the evaluator cannot validate")
			}
		})
	}
}
