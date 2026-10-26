package recall

import (
	"context"
	"hash/fnv"
	"path/filepath"
	"sync"
	"time"

	"github.com/nexus-ia/nexus/core/memory"
	"github.com/nexus-ia/nexus/evolve"
)

var scopeLocks [64]sync.Mutex

// Adaptive is owned by the host, never exposed as an agent tool. Files record
// decisions, not executable prompts; restored choices must pass evaluation again.
type Adaptive struct {
	mu        sync.Mutex
	dir       string
	warn      func(string)
	index     *Index
	active    string
	last      evolve.Report
	checked   time.Time
	evaluated string
}

func NewAdaptive(dir string, warn func(string)) *Adaptive {
	return &Adaptive{dir: dir, warn: warn, active: Legacy}
}

func (a *Adaptive) Rank(ctx context.Context, mems []memory.Memory, query string, limit int) []memory.Memory {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return nil
	}
	_, hash, err := corpusIdentity(mems)
	if err != nil {
		return memory.RankLegacy(mems, query, limit)
	}
	if a.index == nil || a.index.Digest() != hash {
		a.index, err = NewIndex(mems)
		if err != nil {
			return memory.RankLegacy(mems, query, limit)
		}
		a.active = Legacy
		a.evaluated = ""
	}
	// A rapidly changing corpus must not cause continuous benchmarking. A
	// skipped generation stays pending and is retried on a later retrieval.
	if a.evaluated != hash && time.Since(a.checked) >= time.Minute {
		a.checked = time.Now()
		path, _ := filepath.Abs(a.dir)
		h := fnv.New32a()
		_, _ = h.Write([]byte(path))
		lock := &scopeLocks[h.Sum32()%uint32(len(scopeLocks))]
		lock.Lock()
		engine, openErr := evolve.New(evolve.Config{Dir: a.dir, Baseline: Legacy, Candidates: []string{Terms, Phrase}, Policy: evolve.DefaultPolicy()})
		if openErr == nil {
			a.last, openErr = engine.Run(ctx, NewBenchmark(a.index))
		}
		lock.Unlock()
		if openErr == nil {
			a.active = a.last.Active
			a.evaluated = hash
		} else if a.warn != nil {
			a.warn("adaptive recall unavailable; using baseline: " + openErr.Error())
		}
	}
	if ctx.Err() != nil {
		return nil
	}
	// Preserve the caller's tie order whenever adaptation is inactive; the
	// benchmark index uses its own deterministic ordering.
	if a.active == Legacy || !a.index.Covers(query) {
		return memory.RankLegacy(mems, query, limit)
	}
	return a.index.Search(a.active, query, limit)
}

func (a *Adaptive) Report() evolve.Report {
	a.mu.Lock()
	defer a.mu.Unlock()
	r := a.last
	r.Trials = append([]evolve.Trial(nil), r.Trials...)
	return r
}

var _ memory.Ranker = (*Adaptive)(nil)
