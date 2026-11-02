package server

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPIgnoresSpoofedXFFFromRemotePeer(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/public/leads", nil)
	r.RemoteAddr = "203.0.113.7:4444"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := clientIP(r); got != "203.0.113.7" {
		t.Fatalf("clientIP = %q, want peer address", got)
	}
}

func TestClientIPHonorsXFFFromLoopbackProxy(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/public/leads", nil)
	r.RemoteAddr = "127.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "198.51.100.9, 127.0.0.1")
	if got := clientIP(r); got != "198.51.100.9" {
		t.Fatalf("clientIP = %q, want the last non-loopback hop", got)
	}
}

func TestClientIPNoXFF(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/public/leads", nil)
	r.RemoteAddr = "[::1]:6666"
	if got := clientIP(r); got != "::1" {
		t.Fatalf("clientIP = %q, want ::1", got)
	}
}

func TestClientIPIgnoresForgedLeftmostXFF(t *testing.T) {
	// The client sent "X-Forwarded-For: 1.2.3.4"; the local proxy appended the
	// real address. The forged entry must not be used for limits or allowlists.
	r := httptest.NewRequest("POST", "/api/portal/login", nil)
	r.RemoteAddr = "127.0.0.1:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.9")
	if got := clientIP(r); got != "198.51.100.9" {
		t.Fatalf("clientIP = %q, want the proxy-appended address", got)
	}
	r.Header.Set("X-Forwarded-For", "1.2.3.4, garbage")
	if got := clientIP(r); got == "1.2.3.4" {
		t.Fatal("an unreadable hop must not let the forged entry through")
	}
}

func TestClientIPBehindTrustedProxy(t *testing.T) {
	trusted := parseTrustedProxies("172.30.0.0/16, 10.9.8.7, bogus")
	if len(trusted) != 2 {
		t.Fatalf("parsed %d trusted proxies, want 2 (invalid entry skipped)", len(trusted))
	}
	r := httptest.NewRequest("POST", "/api/portal/login", nil)
	// The HTTPS proxy container (172.30.0.5) appends the real client address.
	r.RemoteAddr = "172.30.0.5:40000"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")
	if got := clientIPWith(r, trusted); got != "203.0.113.50" {
		t.Fatalf("clientIP = %q, want the client behind the trusted proxy", got)
	}
	// A forged left-most entry is still ignored.
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.50")
	if got := clientIPWith(r, trusted); got != "203.0.113.50" {
		t.Fatalf("clientIP = %q, want the proxy-appended address", got)
	}
	// Chained trusted proxies are skipped from the right.
	r.Header.Set("X-Forwarded-For", "203.0.113.50, 10.9.8.7")
	if got := clientIPWith(r, trusted); got != "203.0.113.50" {
		t.Fatalf("clientIP = %q, want the first untrusted hop", got)
	}
	// A peer outside the trusted networks cannot set its address.
	r.RemoteAddr = "198.51.100.77:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.50")
	if got := clientIPWith(r, trusted); got != "198.51.100.77" {
		t.Fatalf("clientIP = %q, want the untrusted peer itself", got)
	}
	// Without configuration only loopback is trusted (previous behavior).
	r.RemoteAddr = "172.30.0.5:40000"
	if got := clientIPWith(r, nil); got != "172.30.0.5" {
		t.Fatalf("clientIP = %q, want the peer when no proxy is configured", got)
	}
}
