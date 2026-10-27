// Security hardening for the HTTP API: defense-in-depth response headers,
// allowlist CORS (no wildcard), and a login brute-force guard. The real
// authorization boundary is requireAuth (server-side JWT): tampering with
// client-side routes never grants API access.
package server

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// secureHeaders adds defensive headers to every API response. The API serves
// only JSON/SSE and must never be framed, MIME-sniffed or cached.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// corsHandler replaces permissive wildcard CORS with an allowlist. Browsers only
// reach the API same-origin (through the web proxy), so by default NO cross-origin
// access is granted. Set NEXUS_CORS_ORIGINS (comma-separated) to allow specific
// origins; credentials are never allowed cross-origin.
func corsHandler(next http.Handler) http.Handler {
	allowed := corsAllowlist()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsAllowlist() map[string]bool {
	m := map[string]bool{}
	for _, o := range strings.Split(os.Getenv("NEXUS_CORS_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			m[o] = true
		}
	}
	return m
}

// --- login brute-force guard -------------------------------------------------

type loginAttempts struct {
	fails       int
	windowStart time.Time
	lockedUntil time.Time
}

type bruteGuard struct {
	mu       sync.Mutex
	byIP     map[string]*loginAttempts
	maxFails int
	window   time.Duration
	lockout  time.Duration
}

// loginBrute locks out an IP after 5 failed logins within 15 min, for 15 min.
// This blunts credential brute-forcing without affecting legitimate users.
var loginBrute = &bruteGuard{
	byIP:     make(map[string]*loginAttempts),
	maxFails: 5,
	window:   15 * time.Minute,
	lockout:  15 * time.Minute,
}

func (g *bruteGuard) blocked(ip string) (bool, time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.byIP[ip]
	if a != nil && time.Now().Before(a.lockedUntil) {
		return true, time.Until(a.lockedUntil)
	}
	return false, 0
}

func (g *bruteGuard) fail(ip string) {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.byIP[ip]
	if a == nil || now.Sub(a.windowStart) > g.window {
		a = &loginAttempts{windowStart: now}
		g.byIP[ip] = a
	}
	a.fails++
	if a.fails >= g.maxFails {
		a.lockedUntil = now.Add(g.lockout)
	}
}

func (g *bruteGuard) success(ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.byIP, ip)
}
