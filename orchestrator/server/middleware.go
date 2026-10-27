// Reliability middleware for the HTTP API: panic recovery, request IDs, access
// logging, Prometheus metrics, per-IP rate limiting and in-flight load shedding.
// These keep the service up under bad input and bounded under load, without
// touching the long-lived SSE streams (which are exempt from rate limiting and
// the in-flight cap).
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ctxKey int

const reqIDKey ctxKey = iota

// --- metrics -----------------------------------------------------------------

type metricStore struct {
	start       time.Time
	total       atomic.Int64
	c2xx        atomic.Int64
	c3xx        atomic.Int64
	c4xx        atomic.Int64
	c5xx        atomic.Int64
	durNanos    atomic.Int64
	inflight    atomic.Int64
	panics      atomic.Int64
	shed        atomic.Int64
	rateLimited atomic.Int64
}

var metrics = &metricStore{start: time.Now()}

func (m *metricStore) record(status int, dur time.Duration) {
	m.total.Add(1)
	m.durNanos.Add(int64(dur))
	switch {
	case status >= 500:
		m.c5xx.Add(1)
	case status >= 400:
		m.c4xx.Add(1)
	case status >= 300:
		m.c3xx.Add(1)
	default:
		m.c2xx.Add(1)
	}
}

// metricsHandler exposes Prometheus text-format metrics. It reveals only
// aggregate counters (no payloads); restrict it at the network layer in prod.
func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var b strings.Builder
	g := func(name, help, typ string, val any) {
		fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s %s\n%s %v\n", name, help, name, typ, name, val)
	}
	g("nexus_http_requests_total", "Total HTTP requests served.", "counter", metrics.total.Load())
	fmt.Fprintf(&b, "# HELP nexus_http_responses_total HTTP responses by status class.\n# TYPE nexus_http_responses_total counter\n")
	fmt.Fprintf(&b, "nexus_http_responses_total{class=\"2xx\"} %d\n", metrics.c2xx.Load())
	fmt.Fprintf(&b, "nexus_http_responses_total{class=\"3xx\"} %d\n", metrics.c3xx.Load())
	fmt.Fprintf(&b, "nexus_http_responses_total{class=\"4xx\"} %d\n", metrics.c4xx.Load())
	fmt.Fprintf(&b, "nexus_http_responses_total{class=\"5xx\"} %d\n", metrics.c5xx.Load())
	g("nexus_http_request_duration_seconds_sum", "Sum of request durations.", "counter", float64(metrics.durNanos.Load())/1e9)
	g("nexus_http_request_duration_seconds_count", "Count of measured requests.", "counter", metrics.total.Load())
	g("nexus_http_inflight_requests", "Requests currently being served.", "gauge", metrics.inflight.Load())
	g("nexus_http_panics_total", "Handler panics recovered.", "counter", metrics.panics.Load())
	g("nexus_http_loadshed_total", "Requests shed (503) due to the in-flight cap.", "counter", metrics.shed.Load())
	g("nexus_http_ratelimited_total", "Requests rejected (429) by the per-IP rate limiter.", "counter", metrics.rateLimited.Load())
	g("nexus_uptime_seconds", "Process uptime.", "gauge", int64(time.Since(metrics.start).Seconds()))
	_, _ = w.Write([]byte(b.String()))
}

// --- chain -------------------------------------------------------------------

// withReliability wraps the API handler with the full reliability chain.
func withReliability(h http.Handler) http.Handler {
	return recoverer(requestID(accessLog(rateLimit(loadShed(maxInFlight(), h)))))
}

// recoverer turns a panic in any downstream handler into a 500 (or a clean
// connection close if the response already started) instead of taking the
// process down.
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
			metrics.panics.Add(1)
			log.Printf("[panic] %s %s id=%s: %v\n%s", r.Method, r.URL.Path, reqIDFrom(r.Context()), v, debug.Stack())
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
// the context and the response.
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

// accessLog records metrics for every request and logs the ones worth seeing
// (errors >=400 or slower than a second), so logs stay useful without flooding.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metrics.inflight.Add(1)
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		metrics.inflight.Add(-1)
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		dur := time.Since(start)
		metrics.record(status, dur)
		if status >= 400 || dur > time.Second {
			log.Printf("[http] %s %s %d %dB %s id=%s", r.Method, r.URL.Path, status, sw.bytes,
				dur.Round(time.Millisecond), reqIDFrom(r.Context()))
		}
	})
}

// --- load shedding -----------------------------------------------------------

// loadShed bounds the number of concurrent (non-streaming) requests. Beyond the
// cap it returns 503 with Retry-After. Long-lived SSE streams are exempt.
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
			metrics.shed.Add(1)
			w.Header().Set("Retry-After", "2")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"server busy, please retry shortly"}`))
			log.Printf("[loadshed] 503 %s %s id=%s (in-flight cap %d reached)",
				r.Method, r.URL.Path, reqIDFrom(r.Context()), max)
		}
	})
}

// --- per-IP rate limiting ----------------------------------------------------

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type ipLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64
	burst   float64
}

func newIPLimiter(rps, burst float64) *ipLimiter {
	return &ipLimiter{buckets: make(map[string]*tokenBucket), rps: rps, burst: burst}
}

func (l *ipLimiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[ip]
	if b == nil {
		b = &tokenBucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rps
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// janitor periodically evicts idle buckets so memory stays bounded under churn.
func (l *ipLimiter) janitor() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		l.mu.Lock()
		for ip, b := range l.buckets {
			if b.last.Before(cutoff) {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

// rateLimit applies a per-IP token bucket (default 50 rps, burst 100) to bound
// abuse from any single client. Streaming endpoints are exempt. Disable with
// NEXUS_RATE_RPS=0. X-Forwarded-For is only trusted from a loopback peer (see
// clientIP), so a client cannot spoof its address.
func rateLimit(next http.Handler) http.Handler {
	rps, burst := rateParams()
	if rps <= 0 {
		return next
	}
	lim := newIPLimiter(rps, burst)
	go lim.janitor()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isStreaming(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !lim.allow(clientIP(r)) {
			metrics.rateLimited.Add(1)
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"too many requests, slow down"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateParams() (float64, float64) {
	rps := envFloat("NEXUS_RATE_RPS", 50)
	burst := envFloat("NEXUS_RATE_BURST", 100)
	if burst < rps {
		burst = rps
	}
	return rps, burst
}

func envFloat(key string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return def
}

// --- helpers -----------------------------------------------------------------

// isStreaming matches the SSE routes (…/stream and …/events) which must never be
// rate-limited or counted against the in-flight cap.
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
