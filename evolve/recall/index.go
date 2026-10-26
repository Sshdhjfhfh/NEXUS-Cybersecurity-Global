// Package recall adapts memory retrieval using bounded, declarative strategies.
// Its self-supervised benchmark measures title lookup, not factual truth.
package recall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode"

	"github.com/nexus-ia/nexus/core/memory"
	"golang.org/x/text/unicode/norm"
)

const (
	Legacy         = "legacy-v1"
	Terms          = "terms-v1"
	Phrase         = "phrase-v1"
	MaxDocuments   = 256
	MaxCorpusBytes = 2 << 20
)

func Strategies() []string { return []string{Legacy, Terms, Phrase} }

type document struct {
	memory memory.Memory
	name   string
	fields [3]map[string]bool
}

type Index struct {
	docs   []document
	digest string
}

func normalized(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(strings.ToLower(s)))
}

var stop = map[string]bool{"de": true, "del": true, "el": true, "la": true, "los": true, "las": true, "un": true, "una": true, "en": true, "y": true, "para": true, "con": true, "que": true, "the": true, "and": true, "of": true, "to": true, "in": true, "a": true}

func terms(s string) []string {
	parts := strings.FieldsFunc(normalized(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' })
	seen := map[string]bool{}
	out := []string{}
	for _, p := range parts {
		if !stop[p] && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range terms(s) {
		out[t] = true
	}
	return out
}

func NewIndex(mems []memory.Memory) (*Index, error) {
	copyMems, hash, err := corpusIdentity(mems)
	if err != nil {
		return nil, err
	}
	idx := &Index{digest: hash}
	for _, m := range copyMems {
		idx.docs = append(idx.docs, document{memory: m, name: normalized(m.Name), fields: [3]map[string]bool{tokenSet(m.Name + " " + string(m.Type)), tokenSet(m.Description), tokenSet(m.Content)}})
	}
	return idx, nil
}

func corpusIdentity(mems []memory.Memory) ([]memory.Memory, string, error) {
	if len(mems) > MaxDocuments {
		return nil, "", errors.New("recall: document budget exceeded")
	}
	copyMems := append([]memory.Memory(nil), mems...)
	sort.Slice(copyMems, func(i, j int) bool { return copyMems[i].Name < copyMems[j].Name })
	total := 0
	for _, m := range copyMems {
		total += len(m.Name) + len(m.Description) + len(m.Content)
		if total > MaxCorpusBytes {
			return nil, "", errors.New("recall: corpus budget exceeded")
		}
	}
	raw, err := json.Marshal(copyMems)
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(raw)
	return copyMems, hex.EncodeToString(h[:]), nil
}

func (i *Index) Digest() string { return i.digest }

// evaluableTitle keeps evaluation eligibility and rollout eligibility aligned.
func evaluableTitle(name string) bool {
	return len(name) <= 256 && len(terms(name)) > 0
}

// Covers limits rollout to the query family the evaluator actually measures.
// A title-lookup benchmark cannot validate arbitrary natural-language queries.
func (i *Index) Covers(query string) bool {
	count := 0
	name := normalized(strings.TrimSpace(query))
	for _, d := range i.docs {
		if d.name == name {
			if !evaluableTitle(d.memory.Name) {
				return false
			}
			count++
		}
	}
	return count == 1
}

func (i *Index) Search(strategy, query string, limit int) []memory.Memory {
	if strategy == Legacy {
		mems := make([]memory.Memory, len(i.docs))
		for n, d := range i.docs {
			mems[n] = d.memory
		}
		sort.Slice(mems, func(a, b int) bool {
			if mems[a].ModTime.Equal(mems[b].ModTime) {
				return mems[a].Name < mems[b].Name
			}
			return mems[a].ModTime.After(mems[b].ModTime)
		})
		return memory.RankLegacy(mems, query, limit)
	}
	if strategy != Terms && strategy != Phrase {
		return nil
	}
	tokens := terms(query)
	if len(tokens) == 0 {
		return nil
	}
	type hit struct{ index, score int }
	hits := []hit{}
	for n, d := range i.docs {
		score := 0
		for f, weight := range []int{6, 2, 1} {
			for _, t := range tokens {
				if d.fields[f][t] {
					score += weight
				}
			}
		}
		if strategy == Phrase && d.name == normalized(strings.TrimSpace(query)) {
			score += 6 * len(tokens)
		}
		if score > 0 {
			hits = append(hits, hit{n, score})
		}
	}
	sort.Slice(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score > hits[b].score
		}
		x, y := i.docs[hits[a].index].memory, i.docs[hits[b].index].memory
		if !x.ModTime.Equal(y.ModTime) {
			return x.ModTime.After(y.ModTime)
		}
		return x.Name < y.Name
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]memory.Memory, 0, len(hits))
	for _, h := range hits {
		out = append(out, i.docs[h.index].memory)
	}
	return out
}
