// Package detectbench provides a deterministic detection benchmark for the
// evolve engine. "Generations" are detection rulesets (Sigma); "cases" are
// labeled telemetry events (malicious vs benign). It gives the honest,
// measurable basis for the claim "the blue agent improved": a new ruleset is
// only better if an independent train/holdout score shows it detects more
// attacks WITHOUT regressing on benign traffic.
//
// It runs no network, no LLM, and executes no generated code — it only scores
// rulesets against fixed labeled events, which is exactly what the evolve engine
// requires (strategies never see expected answers).
package detectbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		id := fmt.Sprintf("%s-%s-%d", split, kind, i)
		var ev sigma.Event
		var mal, critical bool
		switch kind {
		case "webshell": // malicious: exposed .php web shell
			ev = sigma.Event{"uri": fmt.Sprintf("/uploads/shell%d.php", i)}
			mal = true
		case "gitleak": // malicious: exposed .git directory (v1 misses this)
			ev = sigma.Event{"path": fmt.Sprintf("/srv/.git/config%d", i)}
			mal = true
		case "benign": // benign: a must-not-alert case (marked Critical)
			ev = sigma.Event{"uri": fmt.Sprintf("/index%d.html", i), "path": fmt.Sprintf("/home/u%d", i)}
			critical = true
		}
		b.cases = append(b.cases, evolve.Case{ID: id, Group: id, Split: split, Critical: critical})
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

	ids := make([]string, 0, len(b.cases))
	for _, c := range b.cases {
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	h := sha256.New()
	for _, id := range ids {
		fmt.Fprintf(h, "%s=%t;", id, b.malicious[id])
	}
	b.digest = hex.EncodeToString(h.Sum(nil))
	return b, nil
}

func (b *Bench) Version() string      { return "detectbench/1" }
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
func (b *Bench) Evaluate(_ context.Context, strategyID, caseID string) (evolve.Outcome, error) {
	rules, ok := b.rulesets[strategyID]
	if !ok {
		return evolve.Outcome{}, fmt.Errorf("detectbench: unknown strategy %q", strategyID)
	}
	ev, ok := b.events[caseID]
	if !ok {
		return evolve.Outcome{}, fmt.Errorf("detectbench: unknown case %q", caseID)
	}
	detected := false
	for _, r := range rules {
		m, err := r.Match(ev)
		if err != nil {
			return evolve.Outcome{}, err
		}
		if m {
			detected = true
			break
		}
	}
	passed := detected == b.malicious[caseID]
	sum := sha256.Sum256([]byte(strategyID + "|" + caseID + "|" + fmt.Sprint(detected)))
	return evolve.Outcome{Passed: passed, Cost: 1, EvidenceDigest: hex.EncodeToString(sum[:])}, nil
}
