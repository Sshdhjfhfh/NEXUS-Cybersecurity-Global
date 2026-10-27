// Reliability middleware for the HTTP API: panic recovery, request IDs, access
// logging (errors + slow requests), and in-flight load shedding. These keep the
// service up under bad input and bounded under load, without touching the
// long-lived SSE streams (which are exempted from the in-flight cap).
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type ctxKey int

const reqIDKey ctxKey = iota

// withReliability wraps the API handler with the full reliability chain.
// Outermost first: recover panics, attach a request id, log, shed load.
func withReliability(h http.Handler) http.Handler {
	return recoverer(requestID(accessLog(loadShed(maxInFlight(), h))))
}

// recoverer turns a panic in any downstream handler into a 500 (or a clean
// connection close if the response already started) instead of taking the
// process down. net/http recovers per-connection too, but this adds a proper
// error body, a logged stack, and the request id.
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v) // intentional abort — let net/http handle it
			}
			log.Printf("[panic] %s %s id=%s: %v\n%s", r.Method, r.URL.Path, reqIDFrom(r.Context()), v, debug.Stack())
			// Best effort: only works if headers were not sent yet.
			func() {
				defer func() { _ = recover() }()
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			}()
		}()
		next.ServeHTTP(w, r)
	})
}

// requestID attaches a short request id (honoring an inbound X-Request-ID) to
// the context and the response, so logs and errors can be correlated.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" || len(id) > 64 {
			id = newReqID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), reqIDKey, id)))
	})
}

func newReqID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "na"
	}
	return hex.EncodeToString(b[:])
}

func reqIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(reqIDKey).(string); ok {
		return v
	}
	return "-"
}

// statusWriter captures the status code and byte count while preserving the
// ResponseWriter's streaming capabilities (Flush) and letting
// http.ResponseController unwrap to the real writer (deadlines, Hijack, etc.).
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusWriter) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// accessLog records only the requests worth seeing: any error (>=400) or any
// request slower than a second. This gives useful signal without flooding logs
// (and disk) under high traffic.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		dur := time.Since(start)
		if status >= 400 || dur > time.Second {
			log.Printf("[http] %s %s %d %dB %s id=%s", r.Method, r.URL.Path, status, sw.bytes,
				dur.Round(time.Millisecond), reqIDFrom(r.Context()))
		}
	})
}

// loadShed bounds the number of concurrent (non-streaming) requests. Beyond the
// cap it returns 503 with Retry-After instead of piling up and exhausting
// memory/DB connections. Long-lived SSE streams are exempt so they never hold a
// slot. max<=0 disables shedding.
func loadShed(max int, next http.Handler) http.Handler {
	if max <= 0 {
		return next
	}
	sem := make(chan struct{}, max)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isStreaming(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Retry-After", "2")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"server busy, please retry shortly"}`))
			log.Printf("[loadshed] 503 %s %s id=%s (in-flight cap %d reached)",
				r.Method, r.URL.Path, reqIDFrom(r.Context()), max)
		}
	})
}

// isStreaming matches the SSE routes (…/stream and …/events) which must never be
// counted against the in-flight cap.
func isStreaming(path string) bool {
	return strings.Contains(path, "/stream") || strings.HasSuffix(path, "/events")
}

// maxInFlight reads the in-flight request cap from NEXUS_MAX_INFLIGHT (default
// 512). 0 disables shedding.
func maxInFlight() int {
	if v := strings.TrimSpace(os.Getenv("NEXUS_MAX_INFLIGHT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return 512
}
