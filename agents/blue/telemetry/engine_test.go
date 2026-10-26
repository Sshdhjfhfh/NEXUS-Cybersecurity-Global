package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/agents/blue/sigma"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/lab/network"
)

const fixtureRule = `title: Laboratory restricted path
id: nexus-lab-path
author: NEXUS original fixture
license: AGPL-3.0-or-later
logsource:
  category: webserver
detection:
  selection:
    uri|startswith: /restricted/
  condition: selection
level: medium
tags:
  - attack.t1190
`

func fixtureEngine(t *testing.T, check lab.Check) *Engine {
	t.Helper()
	if check == nil {
		check = func(_ context.Context, a lab.Action) error {
			if a.AgentID != "blue-fixture" || a.Name != "blue.analyze" || a.TaskID != "7" {
				return fmt.Errorf("unexpected fixture action")
			}
			return nil
		}
	}
	scope, err := network.NewScope("10.60.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	gate, err := lab.NewGate(scope, lab.Checks{Identity: check, Monitor: check, OPSEC: check, Ethics: check})
	if err != nil {
		t.Fatal(err)
	}
	r, err := sigma.Parse([]byte(fixtureRule))
	if err != nil {
		t.Fatal(err)
	}
	e, err := New("blue-fixture", gate, []sigma.Rule{r}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func fixtureEvent(id string, offset time.Duration) Event {
	when := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC).Add(offset)
	return Event{TenantID: "tenant-a", TaskID: "7", ID: id, Asset: "10.60.0.3", ObservedAt: when, ReceivedAt: when.Add(time.Second), Source: sigma.LogSource{Category: "webserver"}, Fields: sigma.Event{"uri": "/restricted/report"}}
}

func TestNormalizeDetectDeduplicateAndCorrelate(t *testing.T) {
	e := fixtureEngine(t, nil)
	a, b, c := fixtureEvent("a", 0), fixtureEvent("b", time.Second), fixtureEvent("c", time.Minute+time.Nanosecond)
	d := fixtureEvent("d", 2*time.Second)
	d.Fields = sigma.Event{"uri": "/public/report"}
	other := fixtureEvent("e", 3*time.Second)
	other.Source.Category = "authentication"
	data, _ := json.Marshal(a)
	normalized, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.Analyze(t.Context(), "tenant-a", "7", []Event{c, b, d, other, normalized, b})
	if err != nil {
		t.Fatal(err)
	}
	if r.InputEvents != 6 || r.UniqueEvents != 5 || r.DuplicateEvents != 1 || r.MatchedEvents != 3 || len(r.Alerts) != 2 {
		t.Fatalf("incorrect accounting: %+v", r)
	}
	if got := strings.Join(r.Alerts[0].EvidenceIDs, ","); got != "a,b" {
		t.Fatalf("wrong temporal grouping: %s", got)
	}
	alert := r.Alerts[0]
	if alert.Author != "NEXUS original fixture" || alert.License != "AGPL-3.0-or-later" || strings.Join(alert.Techniques, ",") != "T1190" || len(r.EvidenceDigest) != 64 {
		t.Fatal("lost provenance or taxonomy")
	}
	// Source tags classify the rule; this test does not emulate an ATT&CK attack.
}

func TestAttributionAndConflictingDuplicatesFailBeforeAuthorization(t *testing.T) {
	var calls atomic.Int32
	e := fixtureEngine(t, func(context.Context, lab.Action) error { calls.Add(1); return nil })
	for _, kind := range []string{"tenant", "task", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			a, b := fixtureEvent("a", 0), fixtureEvent("a", 0)
			switch kind {
			case "tenant":
				b.TenantID = "tenant-b"
			case "task":
				b.TaskID = "8"
			case "duplicate":
				b.Fields = sigma.Event{"uri": "changed"}
			}
			if _, err := e.Analyze(t.Context(), "tenant-a", "7", []Event{a, b}); err == nil {
				t.Fatal("invalid attribution/duplicate accepted")
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatal("partially processed invalid batch")
	}
}

func TestRevocationCancellationAndNoPartialReport(t *testing.T) {
	var calls atomic.Int32
	e := fixtureEngine(t, func(context.Context, lab.Action) error {
		if calls.Add(1) > 4 {
			return fmt.Errorf("fixture permission revoked")
		}
		return nil
	})
	r, err := e.Analyze(t.Context(), "tenant-a", "7", []Event{fixtureEvent("a", 0), fixtureEvent("b", time.Second)})
	if err == nil || len(r.Alerts) != 0 {
		t.Fatal("denied batch returned partial findings")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	before := calls.Load()
	if _, err := e.Analyze(ctx, "tenant-a", "7", []Event{fixtureEvent("a", 0)}); err != context.Canceled || calls.Load() != before {
		t.Fatal("cancelled request reached authorization")
	}
}

func TestDecodeRejectsUnsupportedAndConcatenatedEvents(t *testing.T) {
	e := fixtureEvent("a", 0)
	data, _ := json.Marshal(e)
	if _, err := Decode(append(append([]byte{}, data...), data...)); err == nil {
		t.Fatal("concatenated events accepted")
	}
	for _, replacement := range []string{`"tenant_id":"other","TENANT_ID":"tenant-a"`, `"tenant_id":"other","tenant_id":"tenant-a"`} {
		ambiguous := strings.Replace(string(data), `"tenant_id":"tenant-a"`, replacement, 1)
		if _, err := Decode([]byte(ambiguous)); err == nil {
			t.Fatal("duplicate attribution accepted")
		}
	}
	overflow := strings.Replace(string(data), `"fields":{"uri":"/restricted/report"}`, `"fields":{"value":1e999}`, 1)
	if _, err := Decode([]byte(overflow)); err == nil {
		t.Fatal("numeric overflow accepted")
	}
	e.Fields = sigma.Event{"nested": map[string]any{"uri": "/restricted/"}}
	data, _ = json.Marshal(e)
	if _, err := Decode(data); err == nil {
		t.Fatal("nested fields silently converted")
	}
	if _, err := Decode([]byte(strings.Repeat("x", MaxEventBytes+1))); err == nil {
		t.Fatal("oversized event accepted")
	}
}

func TestConcurrentRequestsDoNotShareIncidentState(t *testing.T) {
	e := fixtureEngine(t, nil)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := fixtureEvent("same-id", 0)
			event.TenantID = fmt.Sprintf("tenant-%d", i)
			r, err := e.Analyze(t.Context(), event.TenantID, "7", []Event{event})
			if err != nil || len(r.Alerts) != 1 || r.Alerts[0].TenantID != event.TenantID {
				t.Errorf("request state leaked: %v %+v", err, r)
			}
		}(i)
	}
	wg.Wait()
}
