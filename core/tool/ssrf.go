package tool

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

// ssrfSafeDialContext returns a DialContext that refuses to connect to private,
// loopback, link-local or cloud-metadata addresses. The check runs in the
// dialer's Control hook — against the *resolved* IP right before connect — so it
// also stops DNS rebinding and HTTP redirects that point at internal targets.
// WebFetch uses it for direct (non-proxied) egress; when a capture/egress proxy
// is configured, that proxy is the egress boundary instead. The guard is on by
// default; set NEXUS_WEBFETCH_SSRF_GUARD=0 to disable it (emergencies only).
//
// Why this matters: without it, a prompt-injected agent could make WebFetch hit
// http://169.254.169.254/ (cloud IAM credentials) or reach Postgres/Redis on the
// internal network — a textbook SSRF-to-cloud-takeover primitive.
func ssrfSafeDialContext() func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			if !ssrfGuardEnabled() {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("ssrf guard: could not parse address %q", address)
			}
			if blockedEgressIP(ip) {
				return fmt.Errorf("ssrf guard: blocked internal/metadata address %s", ip)
			}
			return nil
		},
	}
	return d.DialContext
}

// blockedEgressIP reports whether ip is an address a fetched URL must never
// reach: link-local (incl. 169.254.169.254 cloud metadata, fe80::/10), the AWS
// IMDSv6 endpoint, multicast and the unspecified address. These are never valid
// assessment targets and are the SSRF-to-cloud-credential vector.
//
// Loopback and RFC1918/ULA private addresses are deliberately NOT blocked here:
// NEXUS legitimately assesses authorized internal and local-lab targets (e.g. a
// Juice Shop on 127.0.0.1). WHICH internal hosts are in scope is enforced by the
// task scope gate and, in production, the deny-by-default egress gateway — not by
// a blanket private-range block that would break authorized internal engagements.
func blockedEgressIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip.Equal(net.ParseIP("fd00:ec2::254")) {
		return true // AWS IMDSv6 (ULA, so not caught above)
	}
	return false
}

func ssrfGuardEnabled() bool {
	v := strings.TrimSpace(os.Getenv("NEXUS_WEBFETCH_SSRF_GUARD"))
	return v != "0" && !strings.EqualFold(v, "false")
}
