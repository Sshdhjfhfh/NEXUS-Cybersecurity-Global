// Package telemetry provides original NEXUS passive event normalization,
// source-routed Sigma detection and bounded temporal alert aggregation.
// It never opens a target connection or applies containment actions.
package telemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/agents/blue/sigma"
	"github.com/nexus-ia/nexus/lab"
)

const MaxEventBytes = 256 << 10

var identifier = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)
var technique = regexp.MustCompile(`(?i)^attack\.(t[0-9]{4}(?:\.[0-9]{3})?)$`)

// Event carries attribution supplied by a trusted ingestion boundary. Analyze
// requires tenant/task to match its independent request and reauthorizes assets.
type Event struct {
	TenantID   string          `json:"tenant_id"`
	TaskID     string          `json:"task_id"`
	ID         string          `json:"id"`
	Asset      string          `json:"asset"`
	ObservedAt time.Time       `json:"observed_at"`
	ReceivedAt time.Time       `json:"received_at"`
	Source     sigma.LogSource `json:"source"`
	Fields     sigma.Event     `json:"fields"`
}

// Decode accepts a single bounded JSON event. Nested field objects, non-finite
// values, unknown envelope fields and concatenated documents are rejected.
func Decode(raw []byte) (Event, error) {
	if len(raw) > MaxEventBytes {
		return Event{}, fmt.Errorf("event size limit exceeded")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	var e Event
	if err := d.Decode(&e); err != nil {
		return Event{}, err
	}
	if err := checkJSONKeys(raw); err != nil {
		return Event{}, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return Event{}, fmt.Errorf("exactly one event is required")
	}
	if err := validate(e); err != nil {
		return Event{}, err
	}
	e.Source = normalizeSource(e.Source)
	e.ObservedAt = e.ObservedAt.UTC()
	e.ReceivedAt = e.ReceivedAt.UTC()
	return e, nil
}

func validate(e Event) error {
	task, err := strconv.ParseInt(e.TaskID, 10, 64)
	asset, assetErr := netip.ParseAddr(e.Asset)
	if !identifier.MatchString(e.TenantID) || !identifier.MatchString(e.ID) || err != nil || task <= 0 || assetErr != nil || !asset.IsPrivate() || asset.Is4In6() || asset.Zone() != "" || e.ObservedAt.IsZero() || e.ReceivedAt.IsZero() || len(e.Fields) == 0 || len(e.Fields) > 256 {
		return fmt.Errorf("invalid event attribution or shape")
	}
	for _, s := range []string{e.Source.Product, e.Source.Service, e.Source.Category} {
		if len(s) > 128 {
			return fmt.Errorf("source size limit exceeded")
		}
	}
	if normalizeSource(e.Source) == (sigma.LogSource{}) {
		return fmt.Errorf("event source is required")
	}
	validScalar := func(v any) bool {
		switch n := v.(type) {
		case nil, bool, int, int64, uint64:
			return true
		case float64:
			return !math.IsNaN(n) && !math.IsInf(n, 0)
		case json.Number:
			if !strings.ContainsAny(string(n), ".eE") {
				if _, err := n.Int64(); err == nil {
					return true
				}
				_, err := strconv.ParseUint(string(n), 10, 64)
				return err == nil
			}
			f, err := n.Float64()
			return err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
		case string:
			return len(n) <= 8192
		default:
			return false
		}
	}
	for key, v := range e.Fields {
		if key == "" || len(key) > 128 {
			return fmt.Errorf("invalid field name")
		}
		switch values := v.(type) {
		case []any:
			if len(values) > 256 {
				return fmt.Errorf("field list limit exceeded")
			}
			for _, item := range values {
				if !validScalar(item) {
					return fmt.Errorf("unsupported field value")
				}
			}
		case []string:
			if len(values) > 256 {
				return fmt.Errorf("field list limit exceeded")
			}
			for _, item := range values {
				if !validScalar(item) {
					return fmt.Errorf("field size limit exceeded")
				}
			}
		default:
			if !validScalar(v) {
				return fmt.Errorf("unsupported field value")
			}
		}
	}
	return nil
}

func normalizeSource(s sigma.LogSource) sigma.LogSource {
	return sigma.LogSource{Product: strings.ToLower(strings.TrimSpace(s.Product)), Service: strings.ToLower(strings.TrimSpace(s.Service)), Category: strings.ToLower(strings.TrimSpace(s.Category))}
}
func sourceMatches(rule, event sigma.LogSource) bool {
	return (rule.Product == "" || rule.Product == event.Product) && (rule.Service == "" || rule.Service == event.Service) && (rule.Category == "" || rule.Category == event.Category)
}

type Alert struct {
	TenantID    string    `json:"tenant_id"`
	TaskID      string    `json:"task_id"`
	Asset       string    `json:"asset"`
	RuleID      string    `json:"rule_id"`
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`
	Author      string    `json:"author,omitempty"`
	License     string    `json:"license,omitempty"`
	References  []string  `json:"references,omitempty"`
	Techniques  []string  `json:"techniques"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	EvidenceIDs []string  `json:"evidence_ids"`
}
type Report struct {
	InputEvents     int     `json:"input_events"`
	UniqueEvents    int     `json:"unique_events"`
	DuplicateEvents int     `json:"duplicate_events"`
	MatchedEvents   int     `json:"matched_events"`
	EvidenceDigest  string  `json:"evidence_digest"`
	ElapsedNS       int64   `json:"elapsed_ns"`
	Alerts          []Alert `json:"alerts"`
}
type compiledRule struct {
	predicate *sigma.CompiledRule
	source    sigma.LogSource
	alert     Alert
}
type Engine struct {
	agentID string
	gate    lab.Authorizer
	window  time.Duration
	rules   []compiledRule
}

// New compiles rules once. Callers supply real host authorization; no permissive
// default, process execution, tenant-global cache or shared incident state exists.
func New(agentID string, gate lab.Authorizer, rules []sigma.Rule, window time.Duration) (*Engine, error) {
	if agentID == "" || gate == nil || window <= 0 || window > 24*time.Hour || len(rules) == 0 || len(rules) > 1000 {
		return nil, fmt.Errorf("invalid detection engine configuration")
	}
	e := &Engine{agentID: agentID, gate: gate, window: window}
	seen := map[string]bool{}
	for _, r := range rules {
		if !identifier.MatchString(r.ID) || seen[r.ID] || normalizeSource(r.LogSource) == (sigma.LogSource{}) {
			return nil, fmt.Errorf("unique rule IDs and logsource are required")
		}
		seen[r.ID] = true
		predicate, err := r.Compile()
		if err != nil {
			return nil, err
		}
		a := Alert{RuleID: r.ID, Title: r.Title, Severity: r.Level, Author: r.Author, License: r.License, References: append([]string(nil), r.References...), Techniques: []string{}}
		for _, tag := range r.Tags {
			if m := technique.FindStringSubmatch(tag); m != nil {
				a.Techniques = append(a.Techniques, strings.ToUpper(m[1]))
			}
		}
		sort.Strings(a.Techniques)
		e.rules = append(e.rules, compiledRule{predicate: predicate, source: normalizeSource(r.LogSource), alert: a})
	}
	sort.Slice(e.rules, func(i, j int) bool { return e.rules[i].alert.RuleID < e.rules[j].alert.RuleID })
	return e, nil
}

// Analyze is a passive, request-local replay. The request tenant/task must be
// independently authenticated by the caller. Correlation uses fixed windows
// starting at the first event, scoped to one rule and asset. Different content
// under the same event ID is rejected rather than silently deduplicated.
func (e *Engine) Analyze(ctx context.Context, tenant, task string, events []Event) (Report, error) {
	started := time.Now()
	taskID, taskErr := strconv.ParseInt(task, 10, 64)
	if e == nil || !identifier.MatchString(tenant) || taskErr != nil || taskID <= 0 || len(events) > 10000 {
		return Report{}, fmt.Errorf("invalid detection request")
	}
	unique := make([]Event, 0, len(events))
	seen := map[string]string{}
	total := 0
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		if event.TenantID != tenant || event.TaskID != task {
			return Report{}, fmt.Errorf("event attribution does not match request")
		}
		if err := validate(event); err != nil {
			return Report{}, err
		}
		raw, err := json.Marshal(event)
		if err != nil {
			return Report{}, err
		}
		total += len(raw)
		if len(raw) > MaxEventBytes || total > 16<<20 {
			return Report{}, fmt.Errorf("event batch size limit exceeded")
		}
		copy, err := Decode(raw)
		if err != nil {
			return Report{}, err
		}
		canonical, _ := json.Marshal(copy)
		hash := sha256.Sum256(canonical)
		digest := hex.EncodeToString(hash[:])
		if prior, exists := seen[copy.ID]; exists {
			if prior != digest {
				return Report{}, fmt.Errorf("conflicting duplicate event ID")
			}
			continue
		}
		seen[copy.ID] = digest
		unique = append(unique, copy)
	}
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].ObservedAt.Equal(unique[j].ObservedAt) {
			return unique[i].ID < unique[j].ID
		}
		return unique[i].ObservedAt.Before(unique[j].ObservedAt)
	})
	manifest, _ := json.Marshal(unique)
	digest := sha256.Sum256(manifest)
	report := Report{InputEvents: len(events), UniqueEvents: len(unique), DuplicateEvents: len(events) - len(unique), EvidenceDigest: hex.EncodeToString(digest[:]), Alerts: []Alert{}}
	groups := map[string]int{}
	for _, event := range unique {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		if err := e.gate.Authorize(ctx, lab.Action{AgentID: e.agentID, TaskID: task, Name: "blue.analyze", Target: event.Asset}); err != nil {
			return Report{}, fmt.Errorf("event analysis denied: %w", err)
		}
		matched := false
		for _, rule := range e.rules {
			if !sourceMatches(rule.source, event.Source) || !rule.predicate.Match(event.Fields) {
				continue
			}
			matched = true
			key := rule.alert.RuleID + "\x00" + event.Asset
			index, exists := groups[key]
			if !exists || event.ObservedAt.Sub(report.Alerts[index].FirstSeen) > e.window {
				a := rule.alert
				a.TenantID = tenant
				a.TaskID = task
				a.Asset = event.Asset
				a.FirstSeen = event.ObservedAt
				a.LastSeen = event.ObservedAt
				a.Techniques = append([]string{}, a.Techniques...)
				a.References = append([]string(nil), a.References...)
				a.EvidenceIDs = []string{}
				index = len(report.Alerts)
				report.Alerts = append(report.Alerts, a)
				groups[key] = index
			}
			report.Alerts[index].LastSeen = event.ObservedAt
			report.Alerts[index].EvidenceIDs = append(report.Alerts[index].EvidenceIDs, event.ID)
		}
		if matched {
			report.MatchedEvents++
		}
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	report.ElapsedNS = time.Since(started).Nanoseconds()
	return report, nil
}
