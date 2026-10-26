// Package detectbench provides a deterministic detection benchmark for the
// evolve engine. "Generations" are detection rulesets (Sigma); "cases" are
// labeled synthetic telemetry events. These fixtures verify the mechanics of
// selection and regression gates; they do not establish real-world detection
// accuracy, independent population samples or general agent capability.
//
// It runs no network, no LLM, and executes no generated code — it only scores
// rulesets against fixed labeled events, which is exactly what the evolve engine
// requires (strategies never see expected answers).
package detectbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nexus-ia/nexus/agents/blue/sigma"
	"github.com/nexus-ia/nexus/evolve"
)

// Built-in detection rules (original NEXUS content, not third-party rules).
const (
	rulePHP = `
title: PHP file accessed
id: 00000000-0000-0000-0000-0000000000a1
logsource:
    category: webserver
detection:
    sel:
        uri|endswith: .php
    condition: sel
level: medium
tags:
    - attack.persistence
    - attack.t1505.003
`
	ruleGit = `
title: Git directory exposed
id: 00000000-0000-0000-0000-0000000000a2
logsource:
    category: webserver
detection:
    sel:
        path|contains: /.git
    condition: sel
level: high
`
	ruleNoisyURI = `
title: Any URI (overbroad)
id: 00000000-0000-0000-0000-0000000000a3
detection:
    sel:
        uri|contains: /
    condition: sel
level: low
`
	ruleNoisyPath = `
title: Any path (overbroad)
id: 00000000-0000-0000-0000-0000000000a4
detection:
    sel:
        path|contains: /
    condition: sel
level: low
`
)

// Bench implements evolve.Benchmark for detection rulesets.
type Bench struct {
	rulesets  map[string][]sigma.Rule
	compiled  map[string][]*sigma.CompiledRule
	cases     []evolve.Case
	events    map[string]sigma.Event
	malicious map[string]bool
	digest    string
}

// New builds the benchmark with three strategies:
//   - "v1":    incumbent, detects exposed .php only.
//   - "v2":    improved, also detects exposed .git directories (a real gain).
//   - "noisy": overbroad, flags everything (false positives on benign traffic).
func New() (*Bench, error) {
	return NewGeneration(0)
}

// NewGeneration supplies fresh, deterministic fixture inputs for an isolated
// generation. Families are still synthetic and related; distinct hashes are
// not evidence of statistical independence or broad skill transfer.
func NewGeneration(generation int) (*Bench, error) {
	if generation < 0 || generation > 1000000 {
		return nil, fmt.Errorf("invalid fixture generation")
	}
	parse := func(y string) (sigma.Rule, error) { return sigma.Parse([]byte(y)) }
	php, err := parse(rulePHP)
	if err != nil {
		return nil, err
	}
	git, err := parse(ruleGit)
	if err != nil {
		return nil, err
	}
	nURI, err := parse(ruleNoisyURI)
	if err != nil {
		return nil, err
	}
	nPath, err := parse(ruleNoisyPath)
	if err != nil {
		return nil, err
	}

	b := &Bench{
		rulesets: map[string][]sigma.Rule{
			"v1":    {php},
			"v2":    {php, git},
			"noisy": {nURI, nPath},
		},
		events:    map[string]sigma.Event{},
		malicious: map[string]bool{},
	}

	mk := func(split, kind string, i int) {
		id := fmt.Sprintf("g%d-%s-%s-%d", generation, split, kind, i)
		var ev sigma.Event
		var mal, critical bool
		switch kind {
		case "webshell": // malicious: exposed .php web shell
			ev = sigma.Event{"uri": fmt.Sprintf("/uploads/g%d/shell%d.php", generation, i)}
			if split == evolve.Holdout {
				ev = sigma.Event{"uri": fmt.Sprintf("/replay/g%d/artifact%d.php", generation, i)}
			}
			mal = true
		case "gitleak": // malicious: exposed .git directory (v1 misses this)
			ev = sigma.Event{"path": fmt.Sprintf("/srv/g%d/.git/config%d", generation, i)}
			if split == evolve.Holdout {
				paths := []string{"HEAD", "logs/HEAD", "refs/heads/main", "objects/info/packs", "info/exclude", "hooks/pre-commit.sample"}
				ev = sigma.Event{"path": fmt.Sprintf("/replay/g%d/.git/%s", generation, paths[i])}
			}
			mal = true
		case "benign": // benign: a must-not-alert case (marked Critical)
			ev = sigma.Event{"uri": fmt.Sprintf("/g%d/index%d.html", generation, i), "path": fmt.Sprintf("/home/g%d/u%d", generation, i)}
			if split == evolve.Holdout {
				ev = sigma.Event{"uri": fmt.Sprintf("/g%d/help/%d", generation, i), "path": fmt.Sprintf("/var/g%d/content/%d", generation, i)}
			}
			critical = true
		}
		raw, _ := json.Marshal(ev)
		inputHash := sha256.Sum256(raw)
		b.cases = append(b.cases, evolve.Case{ID: id, InputDigest: hex.EncodeToString(inputHash[:]), Group: id, Split: split, Critical: critical})
		b.events[id] = ev
		b.malicious[id] = mal
	}

	for _, split := range []string{evolve.Train, evolve.Holdout} {
		for i := 0; i < 6; i++ {
			mk(split, "gitleak", i)
		}
		for i := 0; i < 2; i++ {
			mk(split, "webshell", i)
		}
		for i := 0; i < 2; i++ {
			mk(split, "benign", i)
		}
	}

	b.compiled = map[string][]*sigma.CompiledRule{}
	for id, rules := range b.rulesets {
		for _, rule := range rules {
			c, err := rule.Compile()
			if err != nil {
				return nil, err
			}
			b.compiled[id] = append(b.compiled[id], c)
		}
	}
	b.digest = b.fingerprint()
	return b, nil
}

func (b *Bench) fingerprint() string {
	raw, _ := json.Marshal(struct {
		Rules  map[string][]sigma.Rule
		Cases  []evolve.Case
		Events map[string]sigma.Event
		Labels map[string]bool
	}{b.rulesets, b.cases, b.events, b.malicious})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func (b *Bench) Version() string      { return "detectbench/2" }
func (b *Bench) Digest() string       { return b.digest }
func (b *Bench) Cases() []evolve.Case { return append([]evolve.Case(nil), b.cases...) }

// Strategies returns the available strategy ids (for configuring the engine).
func (b *Bench) Strategies() []string {
	out := make([]string, 0, len(b.rulesets))
	for id := range b.rulesets {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Evaluate runs a strategy's ruleset against one case's event and scores it:
// a case passes when the ruleset's detection verdict matches the true label.
func (b *Bench) Evaluate(ctx context.Context, strategyID, caseID string) (evolve.Outcome, error) {
	if err := ctx.Err(); err != nil {
		return evolve.Outcome{}, err
	}
	rules, ok := b.compiled[strategyID]
	if !ok {
		return evolve.Outcome{}, fmt.Errorf("detectbench: unknown strategy %q", strategyID)
	}
	ev, ok := b.events[caseID]
	if !ok {
		return evolve.Outcome{}, fmt.Errorf("detectbench: unknown case %q", caseID)
	}
	detected := false
	for _, r := range rules {
		if r.Match(ev) {
			detected = true
			break
		}
	}
	passed := detected == b.malicious[caseID]
	sum := sha256.Sum256([]byte(b.digest + "|" + strategyID + "|" + caseID + "|" + fmt.Sprint(detected)))
	return evolve.Outcome{Passed: passed, Cost: 1, EvidenceDigest: hex.EncodeToString(sum[:])}, nil
}
