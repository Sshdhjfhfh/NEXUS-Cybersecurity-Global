package recall

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	"github.com/nexus-ia/nexus/evolve"
)

// Benchmark keeps answer keys on the evaluator side. One note is one group:
// generating paraphrases must not inflate the number of independent cases.
type Benchmark struct {
	index *Index
	cases []evolve.Case
	names map[string]string
}

func NewBenchmark(index *Index) *Benchmark {
	b := &Benchmark{index: index, names: map[string]string{}}
	counts := map[string]int{}
	for _, d := range index.docs {
		counts[d.name]++
	}
	for _, d := range index.docs {
		if counts[d.name] != 1 || !evaluableTitle(d.memory.Name) {
			continue
		}
		h := sha256.Sum256([]byte(d.name))
		id := hex.EncodeToString(h[:])
		split := evolve.Train
		if h[0]%3 == 0 {
			split = evolve.Holdout
		}
		b.cases = append(b.cases, evolve.Case{ID: id, Group: id, Split: split})
		b.names[id] = d.memory.Name
	}
	sort.Slice(b.cases, func(i, j int) bool { return b.cases[i].ID < b.cases[j].ID })
	// Bound evaluation work independently of the number of searchable notes.
	if len(b.cases) > 96 {
		b.cases = b.cases[:96]
	}
	return b
}

func (b *Benchmark) Version() string      { return "recall-title-lookup-v1" }
func (b *Benchmark) Digest() string       { return b.index.Digest() }
func (b *Benchmark) Cases() []evolve.Case { return append([]evolve.Case(nil), b.cases...) }

func (b *Benchmark) Evaluate(ctx context.Context, strategy, id string) (evolve.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return evolve.Outcome{}, err
	}
	name, ok := b.names[id]
	if !ok {
		return evolve.Outcome{}, errors.New("recall: unknown evaluation case")
	}
	if strategy != Legacy && strategy != Terms && strategy != Phrase {
		return evolve.Outcome{}, errors.New("recall: unknown strategy")
	}
	result := b.index.Search(strategy, name, 1)
	selected := ""
	cost := int64(0)
	if len(result) > 0 {
		selected = result[0].Name
		cost = int64(len(result[0].Content) + len(result[0].Description) + len(result[0].Name))
	}
	raw, _ := json.Marshal([]string{b.index.Digest(), strategy, id, selected})
	h := sha256.Sum256(raw)
	return evolve.Outcome{Passed: selected == name, Cost: cost, EvidenceDigest: hex.EncodeToString(h[:])}, nil
}
