package recall

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/core/memory"
	"github.com/nexus-ia/nexus/evolve"
	"github.com/nexus-ia/nexus/lab/benchmark/recallfixture"
)

func TestAdaptiveRetrievalPromotesAndSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	notes := recallfixture.Memories()
	baseline := memory.RankLegacy(notes, "Red 001", 1)
	if len(baseline) != 1 || baseline[0].Name != "Redactor 001" {
		t.Fatalf("fixture must expose substring collision: %+v", baseline)
	}
	a := NewAdaptive(dir, func(msg string) { t.Error(msg) })
	got := a.Rank(context.Background(), notes, "Red 001", 1)
	if len(got) != 1 || got[0].Name != "Red 001" {
		t.Fatalf("adaptive recall: %+v", got)
	}
	r := a.Report()
	if r.Decision != "promoted" || r.Holdout.CandidatePassed <= r.Holdout.BaselinePassed {
		t.Fatalf("unmeasured improvement: %+v", r)
	}
	before, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Rank(context.Background(), notes, "Red 002", 1)
	after, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if string(before) != string(after) {
		t.Fatal("unchanged corpus repeated the experiment")
	}
	if strings.Contains(string(after), "Dato de laboratorio") || strings.Contains(string(after), "Redactor 001") {
		t.Fatal("journal contains raw memory text")
	}
	reopened := NewAdaptive(dir, nil)
	if got := reopened.Rank(context.Background(), notes, "Red 001", 1); len(got) != 1 || got[0].Name != "Red 001" {
		t.Fatalf("reopen: %+v", got)
	}
}

func TestChangedCorpusWaitsThenReevaluates(t *testing.T) {
	a := NewAdaptive(t.TempDir(), nil)
	notes := recallfixture.Memories()
	ctx := context.Background()
	_ = a.Rank(ctx, notes, "Red 001", 1)
	first := a.Report()
	notes[0].Content += " Información nueva."
	_ = a.Rank(ctx, notes, "Red 001", 1)
	if a.active != Legacy || a.evaluated != "" {
		t.Fatal("changed corpus used a stale strategy")
	}
	a.checked = time.Now().Add(-2 * time.Minute)
	_ = a.Rank(ctx, notes, "Red 001", 1)
	if a.Report().Dataset == first.Dataset || a.active == Legacy {
		t.Fatal("pending generation was never re-evaluated")
	}
	// A small or ambiguous corpus cannot sustain an old promotion.
	a.checked = time.Now().Add(-2 * time.Minute)
	_ = a.Rank(ctx, notes[:1], "Red 000", 1)
	if a.active != Legacy || a.Report().Decision != "rolled_back" {
		t.Fatal("insufficient new evidence did not restore baseline")
	}
}

func TestCancellationCorruptionAndBudgetsFallBack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := filepath.Join(t.TempDir(), "not-created")
	a := NewAdaptive(dir, nil)
	if got := a.Rank(ctx, recallfixture.Memories(), "Red", 1); len(got) != 0 {
		t.Fatal("canceled retrieval ran")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("canceled retrieval wrote state")
	}
	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	warned := false
	a = NewAdaptive(dir, func(string) { warned = true })
	got := a.Rank(context.Background(), recallfixture.Memories(), "Red 001", 1)
	if !warned || len(got) != 1 || got[0].Name != "Redactor 001" {
		t.Fatal("corrupt state did not retain baseline retrieval")
	}
	oversize := make([]memory.Memory, MaxDocuments+1)
	oversize[0] = memory.Memory{Name: "Referencia", Content: "evidencia"}
	if got := NewAdaptive(t.TempDir(), nil).Rank(context.Background(), oversize, "evidencia", 1); len(got) != 1 {
		t.Fatal("budget limit dropped baseline results")
	}
}

func TestSpanishMatchingAndAmbiguousCases(t *testing.T) {
	notes := []memory.Memory{{Name: "Autenticación", Content: "ámbito"}, {Name: "Autenticacion", Content: "otro ámbito"}, {Name: "Sesión nueva", Content: "nuevo"}}
	idx, err := NewIndex(notes)
	if err != nil {
		t.Fatal(err)
	}
	if got := idx.Search(Phrase, "sesion nueva", 1); len(got) != 1 || got[0].Name != "Sesión nueva" {
		t.Fatalf("accent normalization: %+v", got)
	}
	b := NewBenchmark(idx)
	if len(b.Cases()) != 1 {
		t.Fatal("ambiguous titles inflated the evaluation")
	}
	if _, err := b.Evaluate(context.Background(), "invented-policy", b.Cases()[0].ID); err == nil {
		t.Fatal("accepted an unregistered strategy")
	}
}

func TestPromotionOnlyAppliesToEvaluatedQueryFamily(t *testing.T) {
	notes := recallfixture.Memories()
	a := NewAdaptive(t.TempDir(), nil)
	_ = a.Rank(context.Background(), notes, "Red 001", 1)
	if a.Report().Decision != "promoted" {
		t.Fatal("fixture was not promoted")
	}
	query := "Recuerda los datos de red y las notas del laboratorio"
	got := a.Rank(context.Background(), notes, query, 3)
	want := memory.RankLegacy(notes, query, 3)
	if len(got) != len(want) {
		t.Fatal("unvalidated query family changed behavior")
	}
	for n := range want {
		if got[n].Name != want[n].Name {
			t.Fatal("unvalidated query family used candidate")
		}
	}
}

func TestConcurrentScopeInstancesKeepJournalReadable(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := NewAdaptive(dir, nil)
			if got := a.Rank(context.Background(), recallfixture.Memories(), "Red 001", 1); len(got) != 1 || got[0].Name != "Red 001" {
				t.Error("concurrent adaptation failed")
			}
		}()
	}
	wg.Wait()
	e, err := evolve.New(evolve.Config{Dir: dir, Baseline: Legacy, Candidates: []string{Terms, Phrase}, Policy: evolve.DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Snapshot().Reports) != 4 {
		t.Fatal("concurrent scope instances lost journal entries")
	}
}
