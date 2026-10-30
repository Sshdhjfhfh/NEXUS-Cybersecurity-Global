package server

import (
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalizeAllowlist(t *testing.T) {
	got, err := normalizeAllowlist([]string{" 10.0.0.7/24 ", "1.2.3.4", "", "10.0.0.0/24", "2001:db8::1/32", "::ffff:5.6.7.8"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/24", "1.2.3.4/32", "2001:db8::/32", "5.6.7.8/32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, bad := range []string{"10.0.0.0/33", "intranet", "1.2.3", "10.0.0.0/"} {
		if _, err := normalizeAllowlist([]string{bad}); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	many := make([]string, maxAllowlistItems+1)
	for i := range many {
		many[i] = fmt.Sprintf("10.0.%d.%d", i/250, i%250)
	}
	if _, err := normalizeAllowlist(many); err == nil {
		t.Error("more than the maximum accepted")
	}
}

func TestIPAllowed(t *testing.T) {
	list := []string{"10.0.0.0/24", "2001:db8::/32"}
	cases := map[string]bool{
		"10.0.0.200": true, "10.0.1.1": false, "2001:db8::5": true, "::ffff:10.0.0.9": true,
		"": false, "garbage": false,
	}
	for ip, want := range cases {
		if got := ipAllowed(ip, list); got != want {
			t.Errorf("ipAllowed(%q) = %v, want %v", ip, got, want)
		}
	}
	if !ipAllowed("203.0.113.9", nil) {
		t.Error("empty allowlist must allow any address")
	}
}

func TestPolicyClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:5000"
	r.Header.Set("X-Forwarded-For", "10.0.0.5, 198.51.100.7")
	if got := policyClientIP(r); got != "198.51.100.7" {
		t.Errorf("behind the proxy: got %q, want the proxy-appended (right-most) entry", got)
	}
	r.RemoteAddr = "203.0.113.4:5000" // not the local proxy: header is ignored
	if got := policyClientIP(r); got != "203.0.113.4" {
		t.Errorf("direct peer: got %q", got)
	}
}
