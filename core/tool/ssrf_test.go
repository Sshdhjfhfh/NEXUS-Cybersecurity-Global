package tool

import (
	"context"
	"net"
	"testing"
)

func TestBlockedEgressIP(t *testing.T) {
	// Always blocked: metadata/link-local, multicast, unspecified.
	blocked := []string{
		"169.254.169.254", "169.254.1.1", "fe80::1", "fd00:ec2::254",
		"0.0.0.0", "::", "224.0.0.1", "239.1.1.1",
	}
	for _, s := range blocked {
		if !blockedEgressIP(net.ParseIP(s)) {
			t.Errorf("should block %s", s)
		}
	}
	// Allowed here (scope gate / egress gateway decide internal reachability):
	// loopback, RFC1918/ULA private, CGNAT, and public addresses.
	allowed := []string{
		"127.0.0.1", "::1", "10.0.0.1", "172.16.5.5", "192.168.1.1", "100.64.0.1",
		"fd00::1", "8.8.8.8", "1.1.1.1", "93.184.216.34",
	}
	for _, s := range allowed {
		if blockedEgressIP(net.ParseIP(s)) {
			t.Errorf("should allow %s", s)
		}
	}
}

func TestSSRFDialerRejectsInternal(t *testing.T) {
	t.Setenv("NEXUS_WEBFETCH_SSRF_GUARD", "")
	dial := ssrfSafeDialContext()
	for _, addr := range []string{"169.254.169.254:80", "169.254.1.1:443", "[fe80::1]:80"} {
		if _, err := dial(context.Background(), "tcp", addr); err == nil {
			t.Errorf("dialer should reject metadata/link-local address %s", addr)
		}
	}
}

func TestSSRFGuardEscapeHatch(t *testing.T) {
	t.Setenv("NEXUS_WEBFETCH_SSRF_GUARD", "0")
	if ssrfGuardEnabled() {
		t.Fatal("guard should be disabled when set to 0")
	}
	t.Setenv("NEXUS_WEBFETCH_SSRF_GUARD", "")
	if !ssrfGuardEnabled() {
		t.Fatal("guard should be ON by default")
	}
}
