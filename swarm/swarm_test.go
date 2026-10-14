package swarm

import (
	"context"
	"errors"
	"github.com/nexus-ia/nexus/lab"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type worker struct {
	id  string
	run func(context.Context, Job) ([]byte, error)
}

func (w worker) ID() string                                         { return w.id }
func (w worker) Execute(ctx context.Context, j Job) ([]byte, error) { return w.run(ctx, j) }

type gateFunc func(context.Context, lab.Action) error

func (f gateFunc) Authorize(ctx context.Context, a lab.Action) error { return f(ctx, a) }
func setup(t *testing.T, parallel int, run func(context.Context, Job) ([]byte, error), authorize gateFunc) *Swarm {
	t.Helper()
	workers := map[Role]Worker{}
	for _, role := range Roles() {
		workers[role] = worker{string(role), run}
	}
	s, err := New(authorize, workers, parallel)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func allow(context.Context, lab.Action) error { return nil }
func job(id string, role Role, deps ...string) Job {
	return Job{ID: id, TaskID: "task", Role: role, Target: "172.30.4.2", Dependencies: deps}
}
func TestDependencyFailuresSkipDescendantsAndPreserveOrder(t *testing.T) {
	var mu sync.Mutex
	ran := []string{}
	s := setup(t, 3, func(_ context.Context, j Job) ([]byte, error) {
		mu.Lock()
		ran = append(ran, j.ID)
		mu.Unlock()
		if j.ID == "fail" {
			return nil, errors.New("failed evidence validation")
		}
		return []byte(j.ID), nil
	}, allow)
	jobs := []Job{job("report", Reporter, "fail"), job("ok", Recon), job("fail", Validator, "ok")}
	results, err := s.Run(context.Background(), jobs)
	if err == nil || !results[0].Skipped || results[1].Err != nil || results[2].Err == nil {
		t.Fatal(results, err)
	}
	if !reflect.DeepEqual(ran, []string{"ok", "fail"}) {
		t.Fatal(ran)
	}
	if results[0].JobID != "report" || !reflect.DeepEqual(Summaries(results), []string{"ok"}) {
		t.Fatal("result ordering changed")
	}
}
func TestDAGValidationAndAuthorizationBeforeExecution(t *testing.T) {
	var calls atomic.Int32
	s := setup(t, 2, func(context.Context, Job) ([]byte, error) { calls.Add(1); return nil, nil }, func(context.Context, lab.Action) error { return errors.New("STOP") })
	for _, jobs := range [][]Job{{job("a", Recon, "b"), job("b", Mapper, "a")}, {job("a", Recon, "missing")}, {job("a", Recon), job("a", Mapper)}} {
		if _, err := s.Run(context.Background(), jobs); err == nil {
			t.Fatal("invalid graph accepted")
		}
	}
	if _, err := s.Run(context.Background(), []Job{job("a", Recon)}); err == nil || calls.Load() != 0 {
		t.Fatal("unauthorized worker executed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err := s.Run(ctx, []Job{job("a", Recon)})
	if !errors.Is(err, context.Canceled) || !results[0].Skipped || calls.Load() != 0 {
		t.Fatal("cancellation ignored")
	}
}
func TestConcurrencyBoundAndRoleSerialization(t *testing.T) {
	started := make(chan Role, 8)
	release := make(chan struct{})
	var active atomic.Int32
	var peak atomic.Int32
	s := setup(t, 2, func(ctx context.Context, j Job) ([]byte, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		started <- j.Role
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return nil, nil
	}, allow)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := s.Run(ctx, []Job{job("a", Recon), job("b", Mapper), job("c", Analyzer)})
		done <- err
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("workers did not start")
		}
	}
	select {
	case <-started:
		t.Fatal("concurrency bound exceeded")
	default:
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if peak.Load() != 2 {
		t.Fatal(peak.Load())
	}
	// A worker is stateful; two jobs for its role must never overlap.
	var sameRole atomic.Int32
	s = setup(t, 8, func(context.Context, Job) ([]byte, error) {
		if sameRole.Add(1) != 1 {
			t.Error("same worker ran concurrently")
		}
		defer sameRole.Add(-1)
		return nil, nil
	}, allow)
	if _, err := s.Run(ctx, []Job{job("x", Recon), job("y", Recon)}); err != nil {
		t.Fatal(err)
	}
}
func TestWorkerPanicDoesNotReleaseDependentWork(t *testing.T) {
	s := setup(t, 2, func(context.Context, Job) ([]byte, error) { panic("worker failure") }, allow)
	results, err := s.Run(context.Background(), []Job{job("a", Recon), job("b", Mapper, "a")})
	if err == nil || !results[1].Skipped {
		t.Fatal("panic did not block dependent job")
	}
}
