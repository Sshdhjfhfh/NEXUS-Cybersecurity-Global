package agent

import (
	"path/filepath"
	"testing"

	"github.com/nexus-ia/nexus/core/memory"
	"github.com/nexus-ia/nexus/evolve/recall"
	"github.com/nexus-ia/nexus/lab/benchmark/recallfixture"
)

func TestTaskMemorySharesWithinScopeAndSurvivesReopen(t *testing.T) {
	t.Setenv("NEXUS_MEMORY_ENABLED", "true")
	root := t.TempDir()
	worker := scopedMemory(filepath.Join(root, "tasks", "1"))
	if err := worker.Store.Save(memory.Memory{Name: "Lección", Content: "evidencia de la tarea uno"}); err != nil {
		t.Fatal(err)
	}
	planner := scopedMemory(filepath.Join(root, "tasks", "1"))
	if got := planner.Store.Relevant("evidencia", 3); len(got) != 1 {
		t.Fatalf("planner did not recover worker memory: %+v", got)
	}
	other := scopedMemory(filepath.Join(root, "tasks", "2"))
	if got := other.Store.Relevant("evidencia", 3); len(got) != 0 {
		t.Fatalf("memory crossed task scope: %+v", got)
	}
	t.Setenv("NEXUS_MEMORY_ENABLED", "false")
	if scopedMemory(root) != nil {
		t.Fatal("memory opt-out ignored")
	}
}

func TestAgentMemoryAutomaticallyAdaptsWithinTask(t *testing.T) {
	t.Setenv("NEXUS_MEMORY_ENABLED", "true")
	t.Setenv("NEXUS_EVOLVE_ENABLED", "true")
	root := t.TempDir()
	opts := scopedMemory(filepath.Join(root, "tasks", "1"))
	for _, note := range recallfixture.Memories() {
		if err := opts.Store.Save(note); err != nil {
			t.Fatal(err)
		}
	}
	got := opts.Store.Relevant("Red 001", 1)
	if len(got) != 1 || got[0].Name != "Red 001" {
		t.Fatalf("agent did not use adaptive retrieval: %+v", got)
	}
	ranker, ok := opts.Store.Ranker.(*recall.Adaptive)
	if !ok || ranker.Report().Decision != "promoted" {
		t.Fatal("agent never validated the new strategy")
	}
	other := scopedMemory(filepath.Join(root, "tasks", "2"))
	if got := other.Store.Relevant("Red 001", 1); len(got) != 0 {
		t.Fatal("adaptive memory crossed task scope")
	}
	t.Setenv("NEXUS_EVOLVE_ENABLED", "false")
	disabled := scopedMemory(filepath.Join(root, "tasks", "1"))
	if disabled.Store.Ranker != nil {
		t.Fatal("adaptation opt-out ignored")
	}
	if got := disabled.Store.Relevant("Red 001", 1); len(got) != 1 || got[0].Name != "Redactor 001" {
		t.Fatal("baseline retrieval was not preserved")
	}
}
