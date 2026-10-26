package agent

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	agentcore "github.com/nexus-ia/nexus/core/agent"
	"github.com/nexus-ia/nexus/core/memory"
	"github.com/nexus-ia/nexus/evolve/recall"
)

// scopedMemory shares durable notes within a task, or within one chat. It never
// exposes another task's data. Existing deployments can explicitly opt out.
func scopedMemory(workDir string) *agentcore.MemoryOptions {
	if workDir == "" || strings.EqualFold(os.Getenv("NEXUS_MEMORY_ENABLED"), "false") || os.Getenv("NEXUS_MEMORY_ENABLED") == "0" {
		return nil
	}
	store := memory.NewStore(filepath.Join(workDir, "memory"))
	if !strings.EqualFold(os.Getenv("NEXUS_EVOLVE_ENABLED"), "false") && os.Getenv("NEXUS_EVOLVE_ENABLED") != "0" {
		store.Ranker = recall.NewAdaptive(filepath.Join(workDir, "evolve", "recall"), func(msg string) { log.Printf("[evolve] %s", msg) })
	}
	return &agentcore.MemoryOptions{
		Store:                store,
		AutoInject:           true,
		MaxInject:            3,
		IncludeIndexInPrompt: true,
	}
}
