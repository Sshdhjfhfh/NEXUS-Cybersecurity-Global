package evolve_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/agents/blue/detectbench"
	"github.com/nexus-ia/nexus/evolve"
)

func experimentConfig() evolve.ExperimentConfig {
	return evolve.ExperimentConfig{AgentID: "blue-lab", Generations: 5, Patience: 2, EvaluationBudget: 512, Timeout: 10 * time.Second, CostUnit: "synthetic-case-evaluations", Engine: evolve.Config{Baseline: "v1", Candidates: []string{"v2", "noisy"}, Policy: evolve.DefaultPolicy()}}
}
func freshBenchmark(_ context.Context, g int) (evolve.Benchmark, error) {
	return detectbench.NewGeneration(g)
}

func TestGenerationsPromoteRejectRegressionAndStopAtPlateau(t *testing.T) {
	out, err := evolve.RunExperiment(t.Context(), experimentConfig(), freshBenchmark)
	if err != nil {
		t.Fatal(err)
	}
	if out.LastImprovedGeneration != 1 || out.StoppedAtGeneration != 3 || out.StopReason != "plateau" || len(out.Generations) != 3 {
		t.Fatalf("incorrect history: %+v", out)
	}
	for _, g := range out.Generations {
		if g.Strategy != "v2" || g.Cases != 20 || g.Passed != 20 || g.InitialPassed != 8 || g.SuccessRate != 1 || g.HoldoutSuccessRate != 1 || g.Regressions != 0 || g.CriticalFailures != 0 {
			t.Fatalf("invalid measured metrics: %+v", g)
		}
		if g.Tokens != nil || len(g.SkillsAcquired) != 0 {
			t.Fatal("invented token measurements or acquired skills")
		}
		if g.Evidence.Active == "noisy" {
			t.Fatal("false positive strategy promoted")
		}
	}
	if out.Evaluations != 150 {
		t.Fatalf("evaluation accounting: %d != 150", out.Evaluations)
	}
}

func TestEvaluationBudgetStopsBeforeTheNextScoringCall(t *testing.T) {
	cfg := experimentConfig()
	cfg.EvaluationBudget = 20
	out, err := evolve.RunExperiment(t.Context(), cfg, freshBenchmark)
	if !errors.Is(err, evolve.ErrEvaluationBudget) || out.Evaluations != 20 || out.StopReason != "evaluation_budget" || len(out.Generations) != 0 {
		t.Fatalf("budget overrun or partial promotion: %+v %v", out, err)
	}
}

type renamedGroups struct {
	evolve.Benchmark
	generation int
}

func (b renamedGroups) Cases() []evolve.Case {
	cases := b.Benchmark.Cases()
	for i := range cases {
		cases[i].Group = fmt.Sprintf("new-name-%d-%d", b.generation, i)
	}
	return cases
}

func TestRenamingCasesDoesNotHideInputReplay(t *testing.T) {
	b, err := detectbench.NewGeneration(1)
	if err != nil {
		t.Fatal(err)
	}
	out, err := evolve.RunExperiment(t.Context(), experimentConfig(), func(_ context.Context, g int) (evolve.Benchmark, error) {
		return renamedGroups{Benchmark: b, generation: g}, nil
	})
	if err == nil || out.StopReason != "replayed_or_unidentified_input" || len(out.Generations) != 1 {
		t.Fatalf("replayed evaluator input accepted: %+v %v", out, err)
	}
}

func TestExperimentCannotWriteProductionStateOrRunAfterCancellation(t *testing.T) {
	cfg := experimentConfig()
	cfg.Engine.Dir = t.TempDir()
	if _, err := evolve.RunExperiment(t.Context(), cfg, freshBenchmark); err == nil {
		t.Fatal("persistent agent state accepted in laboratory")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	calls := 0
	_, err := evolve.RunExperiment(ctx, experimentConfig(), func(context.Context, int) (evolve.Benchmark, error) { calls++; return nil, nil })
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("cancelled experiment created evaluator")
	}
}
