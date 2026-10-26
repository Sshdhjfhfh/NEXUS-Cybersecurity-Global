package scan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/nexus-ia/nexus/lab"
)

// --- test gates (implement lab.Authorizer) ---

type allowGate struct{ calls []lab.Action }

func (g *allowGate) Authorize(_ context.Context, a lab.Action) error {
	g.calls = append(g.calls, a)
	return nil
}

type denyGate struct{}

func (denyGate) Authorize(_ context.Context, _ lab.Action) error {
	return fmt.Errorf("outside laboratory scope")
}

func TestNewRequiresGateAndID(t *testing.T) {
	if _, err := New(nil, "red-1"); err == nil {
		t.Error("expected error without gate")
	}
	if _, err := New(&allowGate{}, ""); err == nil {
		t.Error("expected error without agent id")
	}
}

func TestHTTPXDeniedTargetNeverRuns(t *testing.T) {
	s, _ := New(denyGate{}, "red-1")
	// A bogus binary name proves the denial happens BEFORE any exec attempt.
	s.HTTPXBin = "this-binary-does-not-exist"
	_, err := s.HTTPX(context.Background(), "task-1", []string{"http://127.0.0.1:9/"})
	if err == nil {
		t.Fatal("expected authorization denial")
	}
}

func TestHTTPXAuthorizesThenProbes(t *testing.T) {
	if _, err := exec.LookPath("httpx"); err != nil {
		t.Skip("httpx not installed; skipping integration probe")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>Lab Target</title></head><body>ok</body></html>"))
	}))
	defer srv.Close()

	gate := &allowGate{}
	s, _ := New(gate, "red-1")
	probes, err := s.HTTPX(context.Background(), "task-1", []string{srv.URL})
	if err != nil {
		t.Fatalf("httpx: %v", err)
	}
	if len(gate.calls) != 1 || gate.calls[0].Name != "red.httpx" || gate.calls[0].Target != srv.URL {
		t.Fatalf("authorization not recorded correctly: %+v", gate.calls)
	}
	if len(probes) == 0 || probes[0].Status != 200 {
		t.Fatalf("expected a 200 probe, got %+v", probes)
	}
}

// TestParseNucleiWebshellAndCVE validates parsing of nuclei JSONL, including an
// exposed PHP web shell detection and a CVE finding — the kinds of issues we
// report to the audited organization so they can remediate.
func TestParseNucleiWebshellAndCVE(t *testing.T) {
	jsonl := `{"template-id":"php-webshell-detect","matched-at":"http://lab.target/uploads/c99.php","host":"lab.target","info":{"name":"Exposed PHP WebShell (c99)","severity":"critical","tags":["webshell","backdoor","intrusive"],"reference":["https://example.test/webshell"],"description":"An accessible PHP web shell was found","classification":{"cwe-id":["CWE-506"]}}}
{"template-id":"CVE-2021-41773","matched-at":"http://lab.target/cgi-bin/","host":"lab.target","info":{"name":"Apache 2.4.49 Path Traversal","severity":"high","tags":["cve","apache","lfi"],"classification":{"cve-id":["CVE-2021-41773"],"cwe-id":["CWE-22"]}}}
`
	fs, err := parseNuclei([]byte(jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(fs) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(fs))
	}
	if fs[0].TemplateID != "php-webshell-detect" || fs[0].Severity != "critical" || fs[0].CWE != "CWE-506" {
		t.Errorf("webshell finding parsed wrong: %+v", fs[0])
	}
	if fs[1].CVE != "CVE-2021-41773" || fs[1].Severity != "high" || fs[1].MatchedAt == "" {
		t.Errorf("cve finding parsed wrong: %+v", fs[1])
	}
}
