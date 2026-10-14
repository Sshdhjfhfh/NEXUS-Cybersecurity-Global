package db

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/nexus-ia/nexus/agents/blue"
	"github.com/nexus-ia/nexus/orchestrator/level12"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLabReportsPersistAndFeedbackIsAtomic(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	task, err := d.CreateTask("lab persistence", "passive", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(task.ID)
	other, err := d.CreateTask("lab isolation", "passive", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(other.ID)
	now := time.Now().UTC()
	run := level12.Run{ID: uuid.NewString(), TaskID: task.ID, Target: "10.20.0.3", EvidenceIDs: []string{"evidence-1"}, EvidenceDigest: "digest", StartedAt: now, FinishedAt: now, Findings: []blue.Finding{{Rule: "cleartext", EvidenceID: "evidence-1", Target: "10.20.0.3"}}}
	store := d.Level12()
	if err := store.SaveRun(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	replay := run
	replay.ID = uuid.NewString()
	if err := store.SaveRun(t.Context(), replay); !errors.Is(err, level12.ErrConflict) {
		t.Fatalf("evidence replay: %v", err)
	}
	if _, err := store.GetRun(t.Context(), other.ID, run.ID); !errors.Is(err, level12.ErrNotFound) {
		t.Fatalf("cross-task read: %v", err)
	}
	feedback := level12.Feedback{Class: "cleartext", EvidenceID: "evidence-1", Confirmed: true}
	if err := store.AddFeedback(t.Context(), other.ID, run.ID, feedback); !errors.Is(err, level12.ErrNotFound) {
		t.Fatal("cross-task feedback", err)
	}
	if err := store.AddFeedback(t.Context(), task.ID, run.ID, level12.Feedback{Class: "invented", EvidenceID: "evidence-1"}); !errors.Is(err, level12.ErrInvalid) {
		t.Fatal("unbound feedback", err)
	}
	var recorded atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			err := store.AddFeedback(t.Context(), task.ID, run.ID, feedback)
			if err == nil {
				recorded.Add(1)
			} else if !errors.Is(err, level12.ErrConflict) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if recorded.Load() != 1 {
		t.Fatalf("feedback accepted %d times", recorded.Load())
	}
	// A new pool and service must reconstruct predictions from stored feedback.
	reopened, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	service, err := level12.New(level12.Config{Repository: reopened.Level12()})
	if err != nil {
		t.Fatal(err)
	}
	restored, err := service.Get(t.Context(), task.ID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(restored)
	if restored.EvidenceDigest != run.EvidenceDigest || len(raw) == 0 {
		t.Fatal("report not restored")
	}
	rank, err := service.Predictions(t.Context(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range rank {
		if p.Class == "cleartext" && p.Samples != 1 {
			t.Fatal("outcome not restored", p)
		}
	}
	if err := d.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	outcomes, err := store.Outcomes(t.Context(), task.ID)
	if err != nil || len(outcomes) != 0 {
		t.Fatal("feedback did not cascade", err)
	}
}
