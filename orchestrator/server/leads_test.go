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
		t.Fatalf("clientIP = %q, want first XFF hop", got)
	}
}

func TestClientIPNoXFF(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/public/leads", nil)
	r.RemoteAddr = "[::1]:6666"
	if got := clientIP(r); got != "::1" {
		t.Fatalf("clientIP = %q, want ::1", got)
	}
}
