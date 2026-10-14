// Package network defines explicit, private laboratory address boundaries.
// It performs no DNS lookups or network I/O.
package network

import (
	"fmt"
	"net/netip"
)

type Scope struct{ prefixes []netip.Prefix }

func NewScope(cidrs ...string) (Scope, error) {
	if len(cidrs) == 0 {
		return Scope{}, fmt.Errorf("lab scope must not be empty")
	}
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		p, err := netip.ParsePrefix(cidr)
		if err != nil {
			return Scope{}, fmt.Errorf("invalid lab prefix %q: %w", cidr, err)
		}
		p = p.Masked()
		// Requiring the entire prefix to fit an RFC1918 or ULA range prevents
		// private-looking base addresses from authorizing broader public networks.
		allowed := false
		for _, private := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7"} {
			boundary := netip.MustParsePrefix(private)
			if boundary.Addr().BitLen() == p.Addr().BitLen() && p.Bits() >= boundary.Bits() && boundary.Contains(p.Addr()) {
				allowed = true
				break
			}
		}
		if !allowed {
			return Scope{}, fmt.Errorf("lab prefix %q is not entirely private", cidr)
		}
		prefixes = append(prefixes, p)
	}
	return Scope{prefixes: prefixes}, nil
}

// Allows accepts literal IPs only. Hostname resolution must be performed by a
// trusted inventory; accepting arbitrary DNS here would allow rebinding.
func (s Scope) Allows(target string) bool {
	addr, err := netip.ParseAddr(target)
	if err != nil || addr.Zone() != "" || addr.Is4In6() || addr.IsUnspecified() || addr.IsMulticast() {
		return false
	}
	for _, p := range s.prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
func (s Scope) CIDRs() []string {
	result := make([]string, len(s.prefixes))
	for i, p := range s.prefixes {
		result[i] = p.String()
	}
	return result
}
