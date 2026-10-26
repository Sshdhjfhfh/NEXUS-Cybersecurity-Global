package evolve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type fixture struct {
	cases               []Case
	version, generation string
	pass                func(string, Case) bool
	cost                func(string, Case) int64
	calls               map[string]int
	cancel              context.CancelFunc
}

func benchmark() *fixture {
	b := &fixture{version: "test-v1", generation: "first", calls: map[string]int{}}
	for _, split := range []string{Train, Holdout} {
		for n := 0; n < 32; n++ {
			id := fmt.Sprintf("%s-%02d", split, n)
			b.cases = append(b.cases, Case{ID: id, Group: id, Split: split})
		}
	}
	b.pass = func(strategy string, c Case) bool { return strategy != "base" || strings.HasSuffix(c.ID, "0") }
	return b
}
func (b *fixture) Version() string { return b.version }
func (b *fixture) Digest() string  { return digest([]byte(b.generation)) }
func (b *fixture) Cases() []Case   { return append([]Case(nil), b.cases...) }
func (b *fixture) Evaluate(ctx context.Context, s, id string) (Outcome, error) {
	if b.cancel != nil {
		b.cancel()
		return Outcome{}, ctx.Err()
	}
	for _, c := range b.cases {
		if c.ID == id {
			b.calls[s+"/"+c.Split]++
			cost := int64(100)
			if b.cost != nil {
				cost = b.cost(s, c)
			}
			return Outcome{Passed: b.pass(s, c), Cost: cost, EvidenceDigest: digest([]byte(s + id + b.generation))}, nil
		}
	}
	return Outcome{}, fmt.Errorf("unknown case")
}
func config(dir string) Config {
	return Config{Dir: dir, Baseline: "base", Candidates: []string{"candidate"}, Policy: DefaultPolicy()}
}
func engine(t *testing.T, dir string) *Engine {
	t.Helper()
	e, err := New(config(dir))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestPromoteReopenRevalidateAndRollback(t *testing.T) {
	dir := t.TempDir()
	e := engine(t, dir)
	b := benchmark()
	r, err := e.Run(context.Background(), b)
	if err != nil || r.Decision != "promoted" || r.Active != "candidate" || r.Holdout.PValue > .05 {
		t.Fatalf("promotion: %+v %v", r, err)
	}
	info, err := os.Stat(filepath.Join(dir, stateFile))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private state: %v %v", info, err)
	}
	// Repeating an unchanged generation does not generate new evidence.
	before := b.calls["candidate/"+Holdout]
	r.Trials[0].Strategy = "tampered"
	r, err = e.Run(context.Background(), b)
	if err != nil || b.calls["candidate/"+Holdout] != before || len(e.Snapshot().Reports) != 1 || r.Trials[0].Strategy == "tampered" {
		t.Fatal("cached report was mutated or counted twice")
	}
	// Even an unchanged dataset is revalidated after reopening; saved scores
	// are not accepted as new evidence or as authority to activate a strategy.
	e = engine(t, dir)
	if _, err = e.Run(context.Background(), b); err != nil || b.calls["candidate/"+Holdout] == before {
		t.Fatalf("revalidation: %v", err)
	}
	b.generation = "changed"
	b.pass = func(s string, _ Case) bool { return s == "base" }
	r, err = e.Run(context.Background(), b)
	if err != nil || r.Decision != "rolled_back" || r.Active != "base" {
		t.Fatalf("rollback: %+v %v", r, err)
	}
	if engine(t, dir).Snapshot().Active != "base" {
		t.Fatal("rollback did not survive reopening")
	}
}

func TestTrainingWinnerCannotUseHoldoutForSelection(t *testing.T) {
	cfg := config("")
	cfg.Candidates = []string{"a", "b"}
	e, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b := benchmark()
	b.pass = func(s string, c Case) bool {
		if s == "base" {
			return strings.HasSuffix(c.ID, "0")
		}
		return c.Split == Train || s == "b"
	}
	r, err := e.Run(context.Background(), b)
	if err != nil || r.Active != "base" || r.Candidate != "a" || r.Reason != "holdout_rejected" {
		t.Fatalf("overfit candidate: %+v %v", r, err)
	}
	if b.calls["b/"+Holdout] != 0 {
		t.Fatal("runner shopped for another winner on the holdout")
	}
}

func TestCriticalFailureAndCostRegressionRejectPromotion(t *testing.T) {
	for _, mode := range []string{"critical", "cost"} {
		t.Run(mode, func(t *testing.T) {
			b := benchmark()
			original := b.pass
			if mode == "critical" {
				b.cases[33].Critical = true
				b.pass = func(s string, c Case) bool {
					if s == "candidate" && c.Critical {
						return false
					}
					return original(s, c)
				}
			} else {
				b.cost = func(s string, c Case) int64 {
					if s == "candidate" && c.Split == Holdout {
						return 1000
					}
					return 100
				}
			}
			r, err := engine(t, "").Run(context.Background(), b)
			if err != nil || r.Active != "base" {
				t.Fatalf("accepted %s regression: %+v %v", mode, r, err)
			}
		})
	}
}

func TestDuplicateGroupsAndCancellationCannotChangeState(t *testing.T) {
	e := engine(t, "")
	b := benchmark()
	b.cases[32].Group = b.cases[0].Group
	if _, err := e.Run(context.Background(), b); err == nil {
		t.Fatal("allowed group leakage into holdout")
	}
	b = benchmark()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.cancel = cancel
	if _, err := e.Run(ctx, b); err == nil {
		t.Fatal("accepted canceled evaluation")
	}
	if e.Snapshot().Active != "base" || len(e.Snapshot().Reports) != 0 {
		t.Fatal("failure mutated state")
	}
}

func TestInsufficientEvidenceResetsIncumbent(t *testing.T) {
	e := engine(t, "")
	b := benchmark()
	if _, err := e.Run(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	b.generation = "tiny"
	b.cases = b.cases[:1]
	r, err := e.Run(context.Background(), b)
	if err != nil || r.Decision != "rolled_back" || r.Active != "base" || r.Reason != "insufficient_independent_cases" {
		t.Fatalf("small generation: %+v %v", r, err)
	}
}

func TestPersistenceFailureDoesNotPromote(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "blocked")
	e := engine(t, dir)
	if err := os.WriteFile(dir, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Run(context.Background(), benchmark()); err == nil {
		t.Fatal("activation survived failed persistence")
	}
	if e.Snapshot().Active != "base" || len(e.Snapshot().Reports) != 0 {
		t.Fatal("failed commit mutated state")
	}
}

func TestCorruptAndSymlinkStateFailClosed(t *testing.T) {
	for _, mode := range []string{"invalid-json", "unknown-field", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, stateFile)
			if mode == "symlink" {
				target := filepath.Join(t.TempDir(), "state")
				if err := os.WriteFile(target, []byte(`{"version":1,"active":"base"}`), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Skip(err)
				}
			} else {
				raw := `{"version":1,"active":"base","override_permissions":true}`
				if mode == "invalid-json" {
					raw = "{"
				}
				if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := New(config(dir)); err == nil {
				t.Fatal("accepted invalid state")
			}
		})
	}
}

func TestConcurrentRunsDoNotDuplicateExperiences(t *testing.T) {
	e := engine(t, t.TempDir())
	b := benchmark()
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.Run(context.Background(), b); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(e.Snapshot().Reports) != 1 {
		t.Fatal("duplicate evaluation reports")
	}
}

func TestPairedProbabilityAndVersionInvalidation(t *testing.T) {
	cases := []Case{}
	a, b := map[string]Outcome{}, map[string]Outcome{}
	for n := 0; n < 5; n++ {
		id := fmt.Sprint(n)
		cases = append(cases, Case{ID: id, Split: Holdout})
		a[id] = Outcome{Cost: 1}
		b[id] = Outcome{Passed: true, Cost: 1}
	}
	if got := compareCases(cases, a, b, Holdout); got.PValue != .03125 || got.Gain != 1 {
		t.Fatalf("paired probability: %+v", got)
	}
	e := engine(t, "")
	bench := benchmark()
	if _, err := e.Run(context.Background(), bench); err != nil {
		t.Fatal(err)
	}
	before := bench.calls["base/"+Train]
	bench.version = "test-v2"
	if _, err := e.Run(context.Background(), bench); err != nil || bench.calls["base/"+Train] == before {
		t.Fatalf("changed evaluator reused stale evidence: %v", err)
	}
}

func TestInvalidOutcomeAndForgedCatalogCannotActivate(t *testing.T) {
	b := benchmark()
	b.cost = func(string, Case) int64 { return -1 }
	e := engine(t, "")
	if _, err := e.Run(context.Background(), b); err == nil {
		t.Fatal("accepted invalid measurement")
	}
	if len(e.Snapshot().Reports) != 0 {
		t.Fatal("invalid measurements reached the journal")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stateFile), []byte(`{"version":1,"active":"execute-code"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(config(dir)); err == nil {
		t.Fatal("restored a strategy outside the code-owned catalog")
	}
}

func TestJournalRetentionIsBounded(t *testing.T) {
	e := engine(t, t.TempDir())
	b := benchmark()
	b.cases = b.cases[:1]
	for n := 0; n < MaxHistory+4; n++ {
		b.generation = fmt.Sprint(n)
		if _, err := e.Run(context.Background(), b); err != nil {
			t.Fatal(err)
		}
	}
	if len(e.Snapshot().Reports) != MaxHistory {
		t.Fatal("unbounded journal")
	}
}
