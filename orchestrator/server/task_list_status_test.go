package server

import (
	"testing"
)

// Listings derive every task's status with listReadiness instead of
// engine.ReadyFor, which read the role bindings from the database once per task
// (~1k queries, ~300 ms for GET /api/tasks). The statuses must not change, and
// a listing must read each role binding at most once.
func TestListReadinessMatchesEngineAndReadsBindingsOnce(t *testing.T) {
	s, _, request := trafficEvidenceServer(t)
	started, err := s.m.CreateTask("list status: started", "", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.m.DeleteTask(started.ID, DeleteTaskOptions{}) })
	if _, loaded := s.engine.started.LoadOrStore(started.ID, true); !loaded {
		t.Cleanup(func() { s.engine.started.Delete(started.ID) })
	}

	reads := map[string]int{}
	ready := s.listReadiness(func(agentKey string) bool {
		reads[agentKey]++
		return s.agentHasBinding(agentKey)
	})
	tasks := s.m.List()
	if len(tasks) < 2 {
		t.Fatalf("need several tasks to compare, got %d", len(tasks))
	}
	for _, task := range tasks {
		if got, want := ready(task), s.engine.ReadyFor(task); got != want {
			t.Fatalf("task %s: listReadiness=%v, engine.ReadyFor=%v", task.ID, got, want)
		}
		if got, want := s.resolvedTaskStatusWith(task, ready), s.resolvedTaskStatus(task); got != want {
			t.Fatalf("task %s: status %q, want %q", task.ID, got, want)
		}
	}
	for key, n := range reads {
		if n != 1 {
			t.Errorf("binding %q read %d times for %d tasks, want 1", key, n, len(tasks))
		}
	}

	if rec := request("GET", "/api/tasks", ""); rec.Code != 200 {
		t.Fatalf("GET /api/tasks = %d", rec.Code)
	}
}
