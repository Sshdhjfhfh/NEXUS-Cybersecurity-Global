package level12

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/agents/blue"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/lab/network"
	"github.com/nexus-ia/nexus/swarm"
)

type memoryRepo struct {
	mu       sync.Mutex
	runs     map[string]Run
	outcomes []Outcome
}

func (r *memoryRepo) SaveRun(ctx context.Context, run Run) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = run
	return nil
}
func (r *memoryRepo) GetRun(_ context.Context, task int64, id string) (*Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[id]
	if !ok || run.TaskID != task {
		return nil, ErrNotFound
	}
	return &run, nil
}
func (r *memoryRepo) ListRuns(_ context.Context, task int64, limit int) ([]Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Run{}
	for _, run := range r.runs {
		if run.TaskID == task && len(out) < limit {
			out = append(out, run)
		}
	}
	return out, nil
}
func (r *memoryRepo) AddFeedback(_ context.Context, task int64, id string, f Feedback) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runs[id].TaskID != task {
		return ErrNotFound
	}
	for _, o := range r.outcomes {
		if o.RunID == id && o.Class == f.Class {
			return ErrConflict
		}
	}
	r.outcomes = append(r.outcomes, Outcome{id, f.Class, f.Confirmed})
	return nil
}
func (r *memoryRepo) Outcomes(_ context.Context, task int64) ([]Outcome, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Outcome{}
	for _, o := range r.outcomes {
		if r.runs[o.RunID].TaskID == task {
			out = append(out, o)
		}
	}
	return out, nil
}

type sourceFunc func(context.Context, int64, string, []string) ([]blue.Observation, error)

func (f sourceFunc) Load(ctx context.Context, id int64, target string, ids []string) ([]blue.Observation, error) {
	return f(ctx, id, target, ids)
}
func fixture(t *testing.T, check lab.Check) (*Service, *memoryRepo) {
	t.Helper()
	scope, err := network.NewScope("10.20.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if check == nil {
		check = func(context.Context, lab.Action) error { return nil }
	}
	gate, err := lab.NewGate(scope, lab.Checks{Identity: check, OPSEC: check, Ethics: check, Monitor: check})
	if err != nil {
		t.Fatal(err)
	}
	repo := &memoryRepo{runs: map[string]Run{}}
	ids := map[swarm.Role]string{}
	for _, role := range swarm.Roles() {
		ids[role] = "test-" + string(role)
	}
	service, err := New(Config{Repository: repo, Gate: gate, AgentIDs: ids, Source: sourceFunc(func(ctx context.Context, task int64, target string, ids []string) ([]blue.Observation, error) {
		return []blue.Observation{{ID: ids[0], Target: target, Scheme: "http", Status: 200, ObservedAt: time.Now(), Headers: map[string][]string{"Content-Type": {"text/html"}, "Set-Cookie": {"session=secret; HttpOnly"}}}}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	return service, repo
}
func TestPipelinePersistsReportsAndRebuildsPredictions(t *testing.T) {
	s, repo := fixture(t, nil)
	run, err := s.Analyze(t.Context(), 7, Request{"10.20.0.3", []string{"capture-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Workers) != 8 || len(run.Findings) < 3 || len(run.EvidenceDigest) != 64 || len(repo.runs) != 1 {
		t.Fatalf("incomplete report: %+v", run)
	}
	raw, _ := json.Marshal(run)
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "Set-Cookie") {
		t.Fatal("raw credentials persisted")
	}
	f := Feedback{Class: "cleartext", EvidenceID: "capture-1", Confirmed: true}
	if err = s.Feedback(t.Context(), 7, run.ID, f); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(s.Feedback(t.Context(), 7, run.ID, f), ErrConflict) {
		t.Fatal("duplicate feedback accepted")
	}
	if !errors.Is(s.Feedback(t.Context(), 8, run.ID, f), ErrNotFound) {
		t.Fatal("cross-task feedback accepted")
	}
	if !errors.Is(s.Feedback(t.Context(), 7, run.ID, Feedback{Class: "cleartext", EvidenceID: "invented"}), ErrInvalid) {
		t.Fatal("unbound feedback accepted")
	}
	// A new service rebuilds the model from durable outcomes rather than counters.
	fresh, err := New(Config{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	rank, err := fresh.Predictions(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rank {
		if p.Class == "cleartext" && (p.Samples != 1 || p.Ready || p.Probability <= .5) {
			t.Fatalf("invalid prediction: %+v", p)
		}
	}
}
func TestAuthorizationFailsBeforeEvidenceAndPersistence(t *testing.T) {
	s, repo := fixture(t, func(context.Context, lab.Action) error { return errors.New("identity revoked") })
	loaded := false
	s.cfg.Source = sourceFunc(func(context.Context, int64, string, []string) ([]blue.Observation, error) {
		loaded = true
		return nil, nil
	})
	_, err := s.Analyze(t.Context(), 7, Request{"10.20.0.3", []string{"capture-1"}})
	if !errors.Is(err, ErrDenied) || loaded || len(repo.runs) != 0 {
		t.Fatalf("authorization bypass: %v loaded=%v", err, loaded)
	}
}
func TestInvalidEvidenceAndCancellationNeverPersist(t *testing.T) {
	for _, target := range []string{"8.8.8.8", "localhost", "::ffff:10.20.0.3", "10.21.0.3"} {
		s, r := fixture(t, nil)
		if _, err := s.Analyze(t.Context(), 7, Request{target, []string{"capture-1"}}); err == nil || len(r.runs) != 0 {
			t.Fatalf("accepted %s", target)
		}
	}
	s, r := fixture(t, nil)
	if _, err := s.Analyze(t.Context(), 7, Request{"10.20.0.3", []string{"same", "same"}}); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	s.cfg.Source = sourceFunc(func(ctx context.Context, task int64, target string, ids []string) ([]blue.Observation, error) {
		return []blue.Observation{{ID: "different", Target: target}}, nil
	})
	if _, err := s.Analyze(t.Context(), 7, Request{"10.20.0.3", []string{"capture-1"}}); !errors.Is(err, ErrInvalid) || len(r.runs) != 0 {
		t.Fatal("mismatched evidence accepted", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.Analyze(ctx, 7, Request{"10.20.0.3", []string{"capture-1"}}); err == nil || len(r.runs) != 0 {
		t.Fatal("canceled run persisted")
	}
}
func TestDefaultClosedAndConcurrentAnalyses(t *testing.T) {
	s, r := fixture(t, nil)
	disabled, err := New(Config{Repository: r})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status().Ready || len(disabled.Status().Blockers) != 2 {
		t.Fatal("missing guards reported ready")
	}
	if _, err := disabled.Analyze(t.Context(), 7, Request{"10.20.0.3", []string{"capture-1"}}); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := s.Analyze(t.Context(), 7, Request{"10.20.0.3", []string{"capture-1"}}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if len(r.runs) != 8 {
		t.Fatalf("lost reports: %d", len(r.runs))
	}
}
