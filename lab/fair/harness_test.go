package fair

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nexus-ia/nexus/lab"
)

func newTestHarness(t *testing.T) *Harness {
	t.Helper()
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

func TestCompleteCyclePreservesAttemptsButPreventsAccess(t *testing.T) {
	h := newTestHarness(t)
	for _, secured := range []bool{false, true, false} {
		run, err := h.Run(context.Background(), secured)
		if err != nil {
			t.Fatal(err)
		}
		want := 2
		if secured {
			want = 0
		}
		if len(run.Findings) != want {
			t.Fatalf("secured=%v findings=%d", secured, len(run.Findings))
		}
		if run.Metrics != (Metrics{TruePositive: 2, TrueNegative: 2}) || len(run.Detection.Alerts) != 2 {
			t.Fatalf("unexpected detection: %+v", run)
		}
		for _, observation := range run.Observations {
			hash := sha256.Sum256([]byte(observation.Body))
			if observation.SHA256 != hex.EncodeToString(hash[:]) {
				t.Fatal("body evidence hash mismatch")
			}
		}
		if secured && (run.Observations[2].Status != 404 || run.Observations[3].Status != 403) {
			t.Fatal("remediation did not change real HTTP responses")
		}
	}
	state := h.State()
	state.Runs[0].Observations[0].Body = "mutated"
	if h.State().Runs[0].Observations[0].Body == "mutated" {
		t.Fatal("caller mutated stored evidence")
	}
	if !strings.Contains(Markdown(h.State()), "FICTICIO") {
		t.Fatal("report missing response evidence")
	}
}

func TestActiveScopeCannotBeChanged(t *testing.T) {
	h := newTestHarness(t)
	for _, path := range []string{"https://example.org/", "//example.org/", "/../.env", "/.env?url=http://example.org", "/admin"} {
		if _, _, err := h.probe(context.Background(), path); err == nil {
			t.Fatalf("accepted %q", path)
		}
	}
	transport := h.client.Transport.(*http.Transport)
	if _, err := transport.DialContext(context.Background(), "tcp", "127.0.0.1:1"); err == nil {
		t.Fatal("transport accepted a different port")
	}
	if h.client.CheckRedirect(&http.Request{}, nil) == nil {
		t.Fatal("redirect accepted")
	}
	a := fixtureAuthorizer{"1"}
	for _, action := range []lab.Action{{AgentID: "fair-blue", TaskID: "1", Name: "blue.analyze", Target: "10.77.0.3"}, {AgentID: "other", TaskID: "1", Name: "blue.analyze", Target: asset}, {AgentID: "fair-blue", TaskID: "2", Name: "blue.analyze", Target: asset}, {AgentID: "fair-blue", TaskID: "1", Name: "execute", Target: asset}} {
		if a.Authorize(context.Background(), action) == nil {
			t.Fatal("unexpected authorization")
		}
	}
}

func TestCancellationDoesNotPublishPartialRun(t *testing.T) {
	h := newTestHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.Run(ctx, true); err == nil {
		t.Fatal("cancelled run succeeded")
	}
	if len(h.State().Runs) != 0 || h.State().Secured {
		t.Fatal("partial run published or fixture mode changed")
	}
}

func TestEvaluationPromotesGainAndRejectsNoise(t *testing.T) {
	h := newTestHarness(t)
	result, err := h.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Improvement.Decision != "promoted" || result.Improvement.Holdout.CandidatePassed <= result.Improvement.Holdout.BaselinePassed {
		t.Fatal("improvement not verified")
	}
	if result.Noisy.Active != "v1" || result.Noisy.Decision == "promoted" {
		t.Fatal("false positives promoted")
	}
}

func TestHTTPBoundaryAndExports(t *testing.T) {
	h := newTestHarness(t)
	host := "127.0.0.1:8795"
	handler, err := h.Handler(host)
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, body, origin string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://"+host+path, strings.NewReader(body))
		r.Host = host
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if call("GET", "/api/state", "", "", nil).Code != 401 {
		t.Fatal("state exposed without session")
	}
	page := call("GET", "/", "", "", nil)
	if page.Code != 200 || len(page.Result().Cookies()) != 1 || !page.Result().Cookies()[0].HttpOnly {
		t.Fatal("page session missing")
	}
	cookie := page.Result().Cookies()[0]
	if call("GET", "/api/report.md", "", "", cookie).Code != 409 {
		t.Fatal("export before run should be unavailable")
	}
	if call("POST", "/api/run", `{"mode":"baseline"}`, "https://other.example", cookie).Code != 403 {
		t.Fatal("cross origin write accepted")
	}
	if call("POST", "/api/run", `{"mode":"baseline","target":"http://example.org"}`, "http://"+host, cookie).Code != 400 {
		t.Fatal("external target parameter accepted")
	}
	if call("POST", "/api/run", `{"mode":"baseline"}{}`, "http://"+host, cookie).Code != 400 {
		t.Fatal("trailing JSON accepted")
	}
	if call("POST", "/api/run", `{"mode":"baseline"}`, "http://"+host, cookie).Code != 200 {
		t.Fatal("authorized run failed")
	}
	if call("POST", "/api/run", `{"mode":"secured"}`, "http://"+host, cookie).Code != 200 {
		t.Fatal("retest failed")
	}
	response := call("GET", "/api/report.json", "", "", cookie)
	var state State
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &state) != nil || len(state.Runs) != 2 {
		t.Fatal("export does not contain executed runs")
	}
	markdown := call("GET", "/api/report.md", "", "", cookie)
	if markdown.Code != 200 || !strings.Contains(markdown.Body.String(), "403") {
		t.Fatal("report missing correction evidence")
	}
	for _, path := range []string{"/app.css", "/app.js"} {
		response := call("GET", path, "", "", nil)
		if response.Code != 200 {
			t.Fatal("missing asset", path)
		}
		body, _ := io.ReadAll(response.Result().Body)
		if len(body) == 0 {
			t.Fatal("empty asset")
		}
	}
	r := httptest.NewRequest("GET", "http://evil.example/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("DNS rebinding host accepted")
	}
	r = httptest.NewRequest("GET", "http://"+host+"/", nil)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-site entry accepted")
	}
}
