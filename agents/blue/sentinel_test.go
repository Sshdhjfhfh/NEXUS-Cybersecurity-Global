package blue

import (
	"context"
	"errors"
	"github.com/nexus-ia/nexus/lab"
	"reflect"
	"testing"
	"time"
)

type gateFunc func(context.Context, lab.Action) error

func (f gateFunc) Authorize(c context.Context, a lab.Action) error { return f(c, a) }
func TestSentinelUsesEvidenceAndDoesNotModifyObservation(t *testing.T) {
	calls := 0
	s, _ := New("blue-1", gateFunc(func(_ context.Context, a lab.Action) error {
		calls++
		if a.AgentID != "blue-1" || a.Name != "blue.analyze" {
			t.Fatal(a)
		}
		return nil
	}))
	observed := Observation{ID: "e1", Target: "172.30.4.2", Scheme: "https", Status: 200, ObservedAt: time.Now(), FailedLogins: 6, Headers: map[string][]string{"content-type": {"text/html"}, "set-cookie": {"sid=test; Path=/"}}}
	before := append([]string(nil), observed.Headers["set-cookie"]...)
	findings, err := s.Analyze(context.Background(), "task-1", []Observation{observed})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 5 || calls != 1 {
		t.Fatalf("findings=%d calls=%d", len(findings), calls)
	}
	for _, f := range findings {
		if f.EvidenceID != "e1" || f.CWE == "" || f.Recommendation == "" {
			t.Fatal("finding without traceable evidence")
		}
	}
	if !reflect.DeepEqual(before, observed.Headers["set-cookie"]) {
		t.Fatal("evidence mutated")
	}
}
func TestDeniedOrInvalidEvidenceCannotProduceFindings(t *testing.T) {
	denied := errors.New("ethical STOP")
	s, _ := New("blue-1", gateFunc(func(context.Context, lab.Action) error { return denied }))
	observed := Observation{ID: "e1", Target: "172.30.4.2", Scheme: "https", Status: 200, ObservedAt: time.Now()}
	if findings, err := s.Analyze(context.Background(), "task-1", []Observation{observed}); !errors.Is(err, denied) || findings != nil {
		t.Fatal("denied analysis produced findings")
	}
	observed.Status = 0
	if _, err := s.Analyze(context.Background(), "task-1", []Observation{observed}); err == nil {
		t.Fatal("invalid evidence accepted")
	}
	if _, err := New("", nil); err == nil {
		t.Fatal("missing identity accepted")
	}
}
