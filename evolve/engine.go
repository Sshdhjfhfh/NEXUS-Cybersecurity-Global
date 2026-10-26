// Package evolve evaluates bounded strategies against independently scored cases.
// It never executes generated code, changes permissions, or accepts model scores.
package evolve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

const (
	Train      = "train"
	Holdout    = "holdout"
	MaxCases   = 256
	MaxHistory = 32
)

type Case struct {
	ID string `json:"id"`
	// Group identifies the independent unit, not a paraphrase or repeated trial.
	Group    string `json:"group"`
	Split    string `json:"split"`
	Critical bool   `json:"critical"`
}

type Outcome struct {
	Passed         bool   `json:"passed"`
	Cost           int64  `json:"cost"`
	EvidenceDigest string `json:"evidence_digest"`
}

// Benchmark owns inputs, expected answers and scoring. A strategy receives only
// its input through the benchmark implementation, never the expected answer.
type Benchmark interface {
	Version() string
	Digest() string
	Cases() []Case
	Evaluate(context.Context, string, string) (Outcome, error)
}

type Policy struct {
	MinCases     int           `json:"min_cases"`
	MinGain      float64       `json:"min_gain"`
	MaxPValue    float64       `json:"max_p_value"`
	MaxCostRatio float64       `json:"max_cost_ratio"`
	Timeout      time.Duration `json:"timeout"`
}

func DefaultPolicy() Policy {
	return Policy{MinCases: 8, MinGain: .10, MaxPValue: .05, MaxCostRatio: 1.25, Timeout: 2 * time.Second}
}

type Config struct {
	Dir        string
	Baseline   string
	Candidates []string
	Policy     Policy
}

type Trial struct {
	Case     Case    `json:"case"`
	Strategy string  `json:"strategy"`
	Outcome  Outcome `json:"outcome"`
}

type Comparison struct {
	Cases           int     `json:"cases"`
	BaselinePassed  int     `json:"baseline_passed"`
	CandidatePassed int     `json:"candidate_passed"`
	Wins            int     `json:"wins"`
	Losses          int     `json:"losses"`
	Gain            float64 `json:"gain"`
	PValue          float64 `json:"p_value"`
	CostRatio       float64 `json:"cost_ratio"`
	CriticalFailure bool    `json:"critical_failure"`
}

type Report struct {
	Dataset   string     `json:"dataset"`
	Evaluator string     `json:"evaluator"`
	Contract  string     `json:"contract"`
	At        time.Time  `json:"at"`
	Before    string     `json:"before"`
	Active    string     `json:"active"`
	Candidate string     `json:"candidate,omitempty"`
	Decision  string     `json:"decision"`
	Reason    string     `json:"reason"`
	Train     Comparison `json:"train"`
	Holdout   Comparison `json:"holdout"`
	Trials    []Trial    `json:"trials"`
}

type Snapshot struct {
	Version int      `json:"version"`
	Active  string   `json:"active"`
	Reports []Report `json:"reports"`
}

type Engine struct {
	mu       sync.Mutex
	cfg      Config
	contract string
	state    Snapshot
	verified string
}

func New(cfg Config) (*Engine, error) {
	p := cfg.Policy
	if cfg.Baseline == "" || len(cfg.Baseline) > 64 || len(cfg.Candidates) > 8 ||
		p.MinCases < 5 || p.MinCases > MaxCases/2 || !finite(p.MinGain) || p.MinGain <= 0 || p.MinGain > 1 ||
		!finite(p.MaxPValue) || p.MaxPValue <= 0 || p.MaxPValue > .05 || !finite(p.MaxCostRatio) || p.MaxCostRatio < 1 || p.MaxCostRatio > 4 ||
		p.Timeout <= 0 || p.Timeout > time.Minute {
		return nil, errors.New("evolve: invalid configuration")
	}
	seen := map[string]bool{cfg.Baseline: true}
	for _, id := range cfg.Candidates {
		if id == "" || len(id) > 64 || seen[id] {
			return nil, errors.New("evolve: invalid strategy catalog")
		}
		seen[id] = true
	}
	cfg.Candidates = append([]string(nil), cfg.Candidates...)
	sort.Strings(cfg.Candidates)
	raw, _ := json.Marshal(struct {
		Baseline   string
		Candidates []string
		Policy     Policy
	}{cfg.Baseline, cfg.Candidates, p})
	e := &Engine{cfg: cfg, contract: digest(raw), state: Snapshot{Version: 1, Active: cfg.Baseline}}
	if cfg.Dir != "" {
		state, err := readState(cfg.Dir)
		if err != nil {
			return nil, err
		}
		if state != nil {
			if !seen[state.Active] {
				return nil, errors.New("evolve: stored strategy is outside the catalog")
			}
			e.state = *state
		}
	}
	// Persisted state is advisory. Every new engine revalidates before use.
	return e, nil
}

func finite(n float64) bool  { return !math.IsNaN(n) && !math.IsInf(n, 0) }
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func validDigest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == sha256.Size
}

func (e *Engine) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	raw, _ := json.Marshal(e.state)
	var out Snapshot
	_ = json.Unmarshal(raw, &out)
	return out
}

// Run performs a paired comparison. Only the training winner reaches holdout.
// Cancellation, evaluator errors and persistence failures never activate a change.
func (e *Engine) Run(ctx context.Context, b Benchmark) (Report, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, e.cfg.Policy.Timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	if b == nil || b.Version() == "" || len(b.Version()) > 128 || !validDigest(b.Digest()) {
		return Report{}, errors.New("evolve: invalid benchmark identity")
	}
	cases := append([]Case(nil), b.Cases()...)
	if err := validateCases(cases); err != nil {
		return Report{}, err
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	manifest, _ := json.Marshal(cases)
	key := digest([]byte(e.contract + b.Version() + b.Digest() + digest(manifest)))
	if e.verified == key && len(e.state.Reports) > 0 {
		return cloneReport(e.state.Reports[len(e.state.Reports)-1]), nil
	}
	r := Report{Dataset: b.Digest(), Evaluator: b.Version(), Contract: e.contract, At: time.Now().UTC(), Before: e.state.Active, Active: e.state.Active, Decision: "kept", Reason: "no_validated_improvement"}
	counts := map[string]int{}
	for _, c := range cases {
		counts[c.Split]++
	}
	if counts[Train] < e.cfg.Policy.MinCases || counts[Holdout] < e.cfg.Policy.MinCases {
		r.Active = e.cfg.Baseline
		r.Reason = "insufficient_independent_cases"
		if r.Before != r.Active {
			r.Decision = "rolled_back"
		} else {
			r.Decision = "insufficient"
		}
		return e.commit(ctx, r, key)
	}
	results := map[string]map[string]Outcome{}
	evaluate := func(id, split string) error {
		if results[id] == nil {
			results[id] = map[string]Outcome{}
		}
		for _, c := range cases {
			if c.Split != split {
				continue
			}
			if _, ok := results[id][c.ID]; ok {
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			o, err := b.Evaluate(ctx, id, c.ID)
			if err != nil {
				return fmt.Errorf("evolve: evaluation failed: %w", err)
			}
			if o.Cost < 0 || o.Cost > 1e9 || !validDigest(o.EvidenceDigest) {
				return errors.New("evolve: invalid evaluation evidence")
			}
			results[id][c.ID] = o
			r.Trials = append(r.Trials, Trial{Case: c, Strategy: id, Outcome: o})
		}
		return ctx.Err()
	}
	compare := func(base, candidate, split string) Comparison {
		return compareCases(cases, results[base], results[candidate], split)
	}
	for _, split := range []string{Train, Holdout} {
		if err := evaluate(e.cfg.Baseline, split); err != nil {
			return Report{}, err
		}
	}
	// Revalidate the incumbent on changed data before trying to improve it.
	if r.Active != e.cfg.Baseline {
		for _, split := range []string{Train, Holdout} {
			if err := evaluate(r.Active, split); err != nil {
				return Report{}, err
			}
		}
		r.Train = compare(e.cfg.Baseline, r.Active, Train)
		r.Holdout = compare(e.cfg.Baseline, r.Active, Holdout)
		if degraded(r.Train, e.cfg.Policy) || degraded(r.Holdout, e.cfg.Policy) {
			r.Active = e.cfg.Baseline
			r.Decision = "rolled_back"
			r.Reason = "incumbent_regression"
			return e.commit(ctx, r, key)
		}
	}
	best := ""
	var bestTrain Comparison
	for _, candidate := range e.cfg.Candidates {
		if candidate == r.Active {
			continue
		}
		if err := evaluate(candidate, Train); err != nil {
			return Report{}, err
		}
		c := compare(r.Active, candidate, Train)
		if c.Gain < e.cfg.Policy.MinGain || degraded(c, e.cfg.Policy) {
			continue
		}
		if best == "" || c.Gain > bestTrain.Gain || c.Gain == bestTrain.Gain && c.CostRatio < bestTrain.CostRatio {
			best = candidate
			bestTrain = c
		}
	}
	if best == "" {
		return e.commit(ctx, r, key)
	}
	r.Candidate = best
	r.Train = bestTrain
	if err := evaluate(best, Holdout); err != nil {
		return Report{}, err
	}
	r.Holdout = compare(r.Active, best, Holdout)
	if r.Holdout.Gain >= e.cfg.Policy.MinGain && r.Holdout.PValue <= e.cfg.Policy.MaxPValue && !degraded(r.Holdout, e.cfg.Policy) {
		r.Active = best
		r.Decision = "promoted"
		r.Reason = "paired_holdout_improvement"
	} else {
		r.Reason = "holdout_rejected"
	}
	return e.commit(ctx, r, key)
}

func validateCases(cases []Case) error {
	if len(cases) > MaxCases {
		return errors.New("evolve: case budget exceeded")
	}
	ids, groups := map[string]bool{}, map[string]bool{}
	for _, c := range cases {
		if c.ID == "" || len(c.ID) > 128 || c.Group == "" || len(c.Group) > 128 || ids[c.ID] || groups[c.Group] || (c.Split != Train && c.Split != Holdout) {
			return errors.New("evolve: duplicate, correlated or invalid cases")
		}
		ids[c.ID] = true
		groups[c.Group] = true
	}
	return nil
}

func compareCases(cases []Case, a, b map[string]Outcome, split string) Comparison {
	c := Comparison{PValue: 1, CostRatio: 1}
	var ac, bc int64
	for _, item := range cases {
		if item.Split != split {
			continue
		}
		x, y := a[item.ID], b[item.ID]
		c.Cases++
		ac += x.Cost
		bc += y.Cost
		if x.Passed {
			c.BaselinePassed++
		}
		if y.Passed {
			c.CandidatePassed++
		}
		if y.Passed && !x.Passed {
			c.Wins++
		}
		if x.Passed && !y.Passed {
			c.Losses++
		}
		if item.Critical && !y.Passed {
			c.CriticalFailure = true
		}
	}
	if c.Cases > 0 {
		c.Gain = float64(c.CandidatePassed-c.BaselinePassed) / float64(c.Cases)
	}
	c.CostRatio = float64(max(bc, 1)) / float64(max(ac, 1))
	// Exact one-sided paired sign test over discordant independent cases.
	n := c.Wins + c.Losses
	if n > 0 {
		term := math.Ldexp(1, -n)
		c.PValue = 0
		for k := 0; k <= n; k++ {
			if k >= c.Wins {
				c.PValue += term
			}
			term *= float64(n-k) / float64(k+1)
		}
		c.PValue = math.Min(1, c.PValue)
	}
	return c
}

func degraded(c Comparison, p Policy) bool {
	return c.Gain < 0 || c.CriticalFailure || c.CostRatio > p.MaxCostRatio
}

func cloneReport(r Report) Report { r.Trials = append([]Trial(nil), r.Trials...); return r }

func (e *Engine) commit(ctx context.Context, r Report, key string) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	next := Snapshot{Version: 1, Active: r.Active, Reports: append(append([]Report(nil), e.state.Reports...), r)}
	if len(next.Reports) > MaxHistory {
		next.Reports = next.Reports[len(next.Reports)-MaxHistory:]
	}
	// Retain a bounded rolling journal even for a larger future evaluator.
	for len(next.Reports) > 1 {
		raw, err := json.Marshal(next)
		if err != nil {
			return Report{}, err
		}
		if len(raw) <= maxStateBytes {
			break
		}
		next.Reports = next.Reports[1:]
	}
	if e.cfg.Dir != "" {
		if err := writeState(e.cfg.Dir, next); err != nil {
			return Report{}, err
		}
	}
	e.state = next
	e.verified = key
	return cloneReport(r), nil
}
