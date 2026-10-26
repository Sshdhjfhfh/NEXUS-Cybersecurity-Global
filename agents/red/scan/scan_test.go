package scan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
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

func TestParseKatana(t *testing.T) {
	jsonl := `{"timestamp":"t","request":{"method":"GET","endpoint":"http://lab.target/"}}
{"timestamp":"t","request":{"method":"GET","endpoint":"http://lab.target/admin/login"}}
{"timestamp":"t","request":{"method":"GET","endpoint":"http://lab.target/"}}
{"url":"http://lab.target/api/v1/users"}
`
	urls := parseKatana([]byte(jsonl))
	if len(urls) != 3 {
		t.Fatalf("expected 3 unique urls, got %d: %v", len(urls), urls)
	}
	want := map[string]bool{
		"http://lab.target/":             true,
		"http://lab.target/admin/login":  true,
		"http://lab.target/api/v1/users": true,
	}
	for _, u := range urls {
		if !want[u] {
			t.Errorf("unexpected url %q", u)
		}
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

func TestParseGitleaksRedactsSecret(t *testing.T) {
	raw := `[{"RuleID":"aws-access-token","Description":"AWS Access Key","File":"config/.env","StartLine":3,"Secret":"AKIAIOSFODNN7EXAMPLE","Match":"aws_key=AKIAIOSFODNN7EXAMPLE"}]`
	secs, err := parseGitleaks([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(secs) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secs))
	}
	s := secs[0]
	if s.RuleID != "aws-access-token" || s.File != "config/.env" || s.StartLine != 3 {
		t.Errorf("secret parsed wrong: %+v", s)
	}
	// The raw secret must never appear in the preview we report.
	if strings.Contains(s.Preview, "AKIAIOSFODNN7EXAMPLE") || strings.Contains(s.Preview, "OSFODNN") {
		t.Errorf("preview leaks the secret: %q", s.Preview)
	}
	if !strings.HasPrefix(s.Preview, "AKIA") {
		t.Errorf("preview should keep a short fingerprint, got %q", s.Preview)
	}
}

func TestParseGitleaksEmpty(t *testing.T) {
	secs, err := parseGitleaks([]byte("  \n"))
	if err != nil || secs != nil {
		t.Fatalf("empty report should yield no secrets, got %v err=%v", secs, err)
	}
}

func TestParseOSVDedupesAliasesPerGroup(t *testing.T) {
	// osv-scanner lists the same flaw under both its PYSEC and GHSA ids. The
	// "groups" collapse those aliases, and parseOSV must emit ONE finding per
	// group (not one per alias), carrying the other ids as aliases.
	raw := `{"results":[{"source":{"path":"/repo/requirements.txt","type":"lockfile"},"packages":[{"package":{"name":"django","version":"2.2.0","ecosystem":"PyPI"},"vulnerabilities":[{"id":"GHSA-6r97-cj55-9hrq","aliases":["CVE-2019-14234","PYSEC-2019-13"],"summary":"SQL Injection in Django"},{"id":"PYSEC-2019-13","aliases":["CVE-2019-14234"],"summary":""}],"groups":[{"ids":["PYSEC-2019-13","GHSA-6r97-cj55-9hrq"],"aliases":["CVE-2019-14234","GHSA-6r97-cj55-9hrq","PYSEC-2019-13"],"max_severity":"9.8"}]}]}]}`
	vs, err := parseOSV([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected 1 deduped dep vuln, got %d: %+v", len(vs), vs)
	}
	v := vs[0]
	if v.Package != "django" || v.Version != "2.2.0" || v.Source != "/repo/requirements.txt" {
		t.Errorf("package/source parsed wrong: %+v", v)
	}
	if v.ID != "PYSEC-2019-13" {
		t.Errorf("representative id wrong: %q", v.ID)
	}
	if v.Severity != "9.8" {
		t.Errorf("severity should come from group max_severity, got %q", v.Severity)
	}
	if v.Summary != "SQL Injection in Django" {
		t.Errorf("summary should be pulled from a grouped advisory, got %q", v.Summary)
	}
	// Aliases must carry the other ids but never repeat the representative.
	joined := strings.Join(v.Aliases, ",")
	if !strings.Contains(joined, "GHSA-6r97-cj55-9hrq") || !strings.Contains(joined, "CVE-2019-14234") {
		t.Errorf("aliases missing cross-references: %+v", v.Aliases)
	}
	for _, a := range v.Aliases {
		if a == "PYSEC-2019-13" {
			t.Errorf("representative id must not appear in aliases: %+v", v.Aliases)
		}
	}
}

func TestParseOSVTwoGroups(t *testing.T) {
	// Two distinct groups → two findings.
	raw := `{"results":[{"source":{"path":"go.mod"},"packages":[{"package":{"name":"x","version":"1.0.0","ecosystem":"Go"},"vulnerabilities":[{"id":"GO-2024-1","summary":"a"},{"id":"GO-2024-2","summary":"b"}],"groups":[{"ids":["GO-2024-1"],"max_severity":"7.5"},{"ids":["GO-2024-2"],"max_severity":"3.1"}]}]}]}`
	vs, err := parseOSV([]byte(raw))
	if err != nil || len(vs) != 2 {
		t.Fatalf("parse: %v (%d)", err, len(vs))
	}
	if vs[0].Severity != "7.5" || vs[1].Severity != "3.1" {
		t.Errorf("group severities wrong: %+v", vs)
	}
}

func TestSecretsDeniedPathNeverRuns(t *testing.T) {
	s, _ := New(denyGate{}, "red-1")
	s.GitleaksBin = "this-binary-does-not-exist"
	if _, err := s.Secrets(context.Background(), "task-1", "/etc"); err == nil {
		t.Fatal("expected authorization denial before any exec")
	}
}

func TestSCADeniedPathNeverRuns(t *testing.T) {
	s, _ := New(denyGate{}, "red-1")
	s.OSVBin = "this-binary-does-not-exist"
	if _, err := s.SCA(context.Background(), "task-1", "/etc"); err == nil {
		t.Fatal("expected authorization denial before any exec")
	}
}
