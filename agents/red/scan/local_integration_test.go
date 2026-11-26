package scan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/lab/engagement"
)

// This integration runs the actual installed Nuclei process against an owned
// loopback fixture, using one local GET-only template, before and after a fix.
// It does not use public targets or community template downloads.
func TestNucleiLocalExposureAndRetest(t *testing.T) {
	if _, err := exec.LookPath("nuclei"); err != nil {
		t.Skip("nuclei is not installed; real-scanner integration not verified")
	}
	var secured atomic.Bool
	var observed atomic.Int64
	var nonRead atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			nonRead.Add(1)
			http.Error(w, "method denied", 405)
			return
		}
		if r.URL.Path != "/debug/config" || secured.Load() {
			http.NotFound(w, r)
			return
		}
		observed.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"debug":true,"environment":"NEXUS-SYNTHETIC-INTEGRATION"}`)
	}))
	defer target.Close()
	gate, err := engagement.NewAuthorizer(engagement.Scope{Engagement: "nexus-local-test", AuthorizedTargets: []string{target.URL}})
	if err != nil {
		t.Fatal(err)
	}
	scanner, err := New(gate, "nexus-test")
	if err != nil {
		t.Fatal(err)
	}
	templates := t.TempDir()
	template := `id: nexus-local-debug-exposure
info:
  name: NEXUS local synthetic debug exposure
  author: nexus
  severity: high
  tags: exposure,local-test
http:
  - method: GET
    path:
      - "{{BaseURL}}/debug/config"
    matchers-condition: and
    matchers:
      - type: status
        status:
          - 200
      - type: word
        part: body
        words:
          - NEXUS-SYNTHETIC-INTEGRATION
`
	if err := os.WriteFile(filepath.Join(templates, "exposure.yaml"), []byte(template), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	before, err := scanner.Nuclei(ctx, "local-test", target.URL, NucleiOptions{TemplatesDir: templates})
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 1 || before[0].TemplateID != "nexus-local-debug-exposure" || observed.Load() == 0 {
		t.Fatalf("expected actual evidence, findings=%+v requests=%d", before, observed.Load())
	}
	secured.Store(true)
	after, err := scanner.Nuclei(ctx, "local-test", target.URL, NucleiOptions{TemplatesDir: templates})
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 0 || nonRead.Load() != 0 {
		t.Fatalf("unexpected result: findings=%+v non-read=%d", after, nonRead.Load())
	}
	t.Logf("Real Nuclei process: %d confirmed finding before, %d after, %d exposure requests observed, %d write methods", len(before), len(after), observed.Load(), nonRead.Load())
}
