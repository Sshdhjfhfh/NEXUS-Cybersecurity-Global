package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestRecovererReturns500(t *testing.T) {
	h := recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil)) // must not crash the test process
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
}

func TestRequestIDSetAndHonored(t *testing.T) {
	var seen string
	h := requestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = reqIDFrom(r.Context()) }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Header().Get("X-Request-ID") == "" || seen == "" || seen == "-" {
		t.Fatalf("missing generated request id (header=%q ctx=%q)", rec.Header().Get("X-Request-ID"), seen)
	}

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Request-ID", "abc123")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Header().Get("X-Request-ID") != "abc123" {
		t.Fatalf("inbound id not honored: %q", rec2.Header().Get("X-Request-ID"))
	}
}

func TestLoadShedShedsBeyondLimit(t *testing.T) {
	block := make(chan struct{})
	inflight := make(chan struct{})
	h := loadShed(1, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inflight <- struct{}{}
		<-block
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/a", nil))
	}()
	<-inflight // the one slot is now held

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/b", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 when saturated, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("503 must include Retry-After")
	}
	close(block)
	wg.Wait()
}

func TestLoadShedExemptsStreaming(t *testing.T) {
	block := make(chan struct{})
	inflight := make(chan struct{})
	h := loadShed(1, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hold" {
			inflight <- struct{}{}
			<-block
		}
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/hold", nil))
	}()
	<-inflight // slot held by the non-streaming request

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/logs/stream", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("streaming path must bypass the limiter, got %d", rec.Code)
	}
	close(block)
	wg.Wait()
}

func TestStatusWriterCapabilities(t *testing.T) {
	var sw http.ResponseWriter = &statusWriter{ResponseWriter: httptest.NewRecorder()}
	if _, ok := sw.(http.Flusher); !ok {
		t.Fatal("statusWriter must implement http.Flusher (SSE would break otherwise)")
	}
	rc := sw.(*statusWriter)
	rc.WriteHeader(http.StatusNotFound)
	if rc.status != http.StatusNotFound {
		t.Fatalf("status not captured: %d", rc.status)
	}
	if rc.Unwrap() == nil {
		t.Fatal("Unwrap must return the underlying writer")
	}
}

func TestIsStreaming(t *testing.T) {
	for _, p := range []string{"/api/logs/stream", "/api/update/stream", "/api/exploration/activity/stream", "/api/side-questions/x/events"} {
		if !isStreaming(p) {
			t.Fatalf("%s should be detected as streaming", p)
		}
	}
	if isStreaming("/api/leads") || isStreaming("/api/public/leads") {
		t.Fatal("non-streaming paths must not be exempt")
	}
}
