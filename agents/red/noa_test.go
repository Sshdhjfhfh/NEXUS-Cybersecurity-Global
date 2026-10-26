package agent

import (
	"os"
	"path/filepath"
	"testing"

	agentcore "github.com/nexus-ia/nexus/core/agent"
	"github.com/nexus-ia/nexus/core/compaction"
	"github.com/nexus-ia/nexus/core/harness"
)

func TestAdvancedCompactionToggleAndFallback(t *testing.T) {
	standard := &compaction.Config{ContextWindow: 40000}
	opts := agentcore.Options{Compaction: standard}
	enableNoa(&opts, func() bool { return false }, t.TempDir(), "task", nil)
	if opts.Compaction != standard || opts.Compactor != nil {
		t.Fatal("disabled option changed compaction")
	}
	enableNoa(&opts, func() bool { return true }, t.TempDir(), "task", nil)
	if _, ok := opts.Compactor.(harness.ContextView); !ok || opts.Compaction != nil {
		t.Fatal("enabled option did not install advanced compaction")
	}
	found := false
	for _, tool := range opts.Tools {
		if tool.Name() == "Compress" {
			found = true
		}
	}
	if !found {
		t.Fatal("compression tool missing")
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts = agentcore.Options{Compaction: standard}
	warned := false
	enableNoa(&opts, func() bool { return true }, blocked, "task", func(string) { warned = true })
	if !warned || opts.Compaction != standard || opts.Compactor != nil {
		t.Fatal("archive failure did not preserve standard compaction")
	}
}
