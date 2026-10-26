// Original NEXUS generation laboratory for registered, compiled strategies.
// This runner executes no generated code, changes no permissions and never
// connects selected strategies to the production agent runtime.
package evolve

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ExperimentConfig struct {
	AgentID          string
	Generations      int
	Patience         int
	EvaluationBudget int
	Timeout          time.Duration
	CostUnit         string
	Engine           Config
}
type Generation struct {
	Number             int      `json:"number"`
	AgentID            string   `json:"agent_id"`
	Strategy           string   `json:"strategy"`
	Cases              int      `json:"cases"`
	InitialPassed      int      `json:"initial_passed"`
	Passed             int      `json:"passed"`
	Regressions        int      `json:"regressions"`
	SuccessRate        float64  `json:"success_rate"`
	GainFromInitial    float64  `json:"gain_from_initial"`
	RegressionRate     float64  `json:"regression_rate"`
	HoldoutSuccessRate float64  `json:"holdout_success_rate"`
	CriticalFailures   int      `json:"critical_failures"`
	Evaluations        int      `json:"evaluations"`
	Cost               int64    `json:"cost"`
	ElapsedNS          int64    `json:"elapsed_ns"`
	Tokens             *int64   `json:"tokens"`          // nil: this benchmark does not measure tokens
	SkillsAcquired     []string `json:"skills_acquired"` // no skills are invented by selection
	Evidence           Report   `json:"evidence"`
}
type Experiment struct {
	AgentID                string       `json:"agent_id"`
	InitialStrategy        string       `json:"initial_strategy"`
	CostUnit               string       `json:"cost_unit"`
	Generations            []Generation `json:"generations"`
	Evaluations            int          `json:"evaluations"`
	LastImprovedGeneration int          `json:"last_improved_generation"`
	StoppedAtGeneration    int          `json:"stopped_at_generation"`
	StopReason             string       `json:"stop_reason"`
}

// BenchmarkFactory is supplied by the evaluator owner, not the candidate.
// It must reserve fresh input/groups for every generation. Hash uniqueness
// rejects exact replay; it does not establish statistical independence.
type BenchmarkFactory func(context.Context, int) (Benchmark, error)

var ErrEvaluationBudget = errors.New("experiment evaluation budget exhausted")

type budgetBenchmark struct {
	Benchmark
	used  *int
	limit int
	cases []Case
}

func (b budgetBenchmark) Cases() []Case { return append([]Case(nil), b.cases...) }

func (b budgetBenchmark) Evaluate(ctx context.Context, strategy, id string) (Outcome, error) {
	if err := ctx.Err(); err != nil {
		return Outcome{}, err
	}
	if *b.used >= b.limit {
		return Outcome{}, ErrEvaluationBudget
	}
	*b.used = *b.used + 1
	return b.Benchmark.Evaluate(ctx, strategy, id)
}

// RunExperiment repeatedly compares the host-registered variants with the
// existing independent scorer. Budgets, evaluator, catalog and policy are fixed
// before the first generation. Benchmarks are trusted compiled host code; this
// API is not a sandbox for arbitrary plugins or a model-training pipeline.
func RunExperiment(ctx context.Context, cfg ExperimentConfig, factory BenchmarkFactory) (Experiment, error) {
	out := Experiment{AgentID: cfg.AgentID, InitialStrategy: cfg.Engine.Baseline, CostUnit: cfg.CostUnit, Generations: []Generation{}}
	if cfg.AgentID == "" || len(cfg.AgentID) > 128 || cfg.Generations < 1 || cfg.Generations > 32 || cfg.Patience < 1 || cfg.Patience > cfg.Generations || cfg.EvaluationBudget < 1 || cfg.EvaluationBudget > 8192 || cfg.Timeout <= 0 || cfg.Timeout > time.Hour || cfg.CostUnit == "" || cfg.Engine.Dir != "" || factory == nil {
		return out, fmt.Errorf("invalid isolated experiment configuration")
	}
	cfg.Engine.Candidates = append([]string(nil), cfg.Engine.Candidates...)
	if _, err := New(cfg.Engine); err != nil {
		return out, err
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	active := cfg.Engine.Baseline
	idle := 0
	seenInputs, seenGroups := map[string]bool{}, map[string]bool{}
	for number := 1; number <= cfg.Generations; number++ {
		if err := ctx.Err(); err != nil {
			out.StopReason = "cancelled"
			return out, err
		}
		if out.Evaluations >= cfg.EvaluationBudget {
			out.StopReason = "evaluation_budget"
			return out, ErrEvaluationBudget
		}
		started := time.Now()
		b, err := factory(ctx, number)
		if err != nil {
			out.StopReason = "evaluator_failed"
			return out, err
		}
		if b == nil {
			out.StopReason = "invalid_evaluator"
			return out, fmt.Errorf("missing generation evaluator")
		}
		cases := b.Cases()
		if err := validateCases(cases); err != nil {
			out.StopReason = "invalid_cases"
			return out, err
		}
		for _, c := range cases {
			if !validDigest(c.InputDigest) || seenInputs[c.InputDigest] || seenGroups[c.Group] {
				out.StopReason = "replayed_or_unidentified_input"
				return out, fmt.Errorf("generation input/group is missing or repeated")
			}
			seenInputs[c.InputDigest] = true
			seenGroups[c.Group] = true
		}
		e, err := New(cfg.Engine)
		if err != nil {
			return out, err
		}
		e.state.Active = active // memory-only experiment; no production state directory
		before := out.Evaluations
		report, err := e.Run(ctx, budgetBenchmark{Benchmark: b, used: &out.Evaluations, limit: cfg.EvaluationBudget, cases: append([]Case(nil), cases...)})
		out.StoppedAtGeneration = number
		if err != nil {
			if errors.Is(err, ErrEvaluationBudget) {
				out.StopReason = "evaluation_budget"
			} else {
				out.StopReason = "evaluation_failed"
			}
			return out, err
		}
		if report.Decision == "insufficient" {
			out.StopReason = "insufficient_evidence"
			return out, nil
		}
		g := Generation{Number: number, AgentID: cfg.AgentID, Strategy: report.Active, Cases: len(cases), Evaluations: out.Evaluations - before, ElapsedNS: time.Since(started).Nanoseconds(), SkillsAcquired: []string{}, Evidence: report}
		results := map[string]map[string]Outcome{}
		for _, trial := range report.Trials {
			if results[trial.Strategy] == nil {
				results[trial.Strategy] = map[string]Outcome{}
			}
			results[trial.Strategy][trial.Case.ID] = trial.Outcome
			g.Cost += trial.Outcome.Cost
		}
		holdout, holdoutPassed := 0, 0
		for _, c := range cases {
			initial, okInitial := results[cfg.Engine.Baseline][c.ID]
			current, okCurrent := results[report.Active][c.ID]
			if !okInitial || !okCurrent {
				out.StopReason = "incomplete_evidence"
				return out, fmt.Errorf("missing scored case")
			}
			if initial.Passed {
				g.InitialPassed++
			}
			if current.Passed {
				g.Passed++
			}
			if initial.Passed && !current.Passed {
				g.Regressions++
			}
			if c.Critical && !current.Passed {
				g.CriticalFailures++
			}
			if c.Split == Holdout {
				holdout++
				if current.Passed {
					holdoutPassed++
				}
			}
		}
		if g.Cases > 0 {
			g.SuccessRate = float64(g.Passed) / float64(g.Cases)
			g.GainFromInitial = float64(g.Passed-g.InitialPassed) / float64(g.Cases)
			g.RegressionRate = float64(g.Regressions) / float64(g.Cases)
		}
		if holdout > 0 {
			g.HoldoutSuccessRate = float64(holdoutPassed) / float64(holdout)
		}
		out.Generations = append(out.Generations, g)
		if report.Decision == "promoted" {
			out.LastImprovedGeneration = number
			idle = 0
		} else {
			idle++
		}
		active = report.Active
		if idle >= cfg.Patience {
			out.StopReason = "plateau"
			return out, nil
		}
	}
	out.StopReason = "generation_limit"
	return out, nil
}
