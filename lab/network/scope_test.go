package network

import "testing"

func TestScopeCannotAuthorizePublicOrReboundTargets(t *testing.T) {
	scope, err := NewScope("172.30.4.0/24", "fd12:3456::/64")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"172.30.4.4", "fd12:3456::4"} {
		if !scope.Allows(target) {
			t.Errorf("denied lab target %s", target)
		}
	}
	for _, target := range []string{"172.30.5.4", "8.8.8.8", "localhost", "example.com", "127.0.0.1", "169.254.169.254", "::ffff:172.30.4.4", "fd12:3456::4%eth0"} {
		if scope.Allows(target) {
			t.Errorf("authorized unsafe target %s", target)
		}
	}
	copy := scope.CIDRs()
	copy[0] = "0.0.0.0/0"
	if scope.Allows("8.8.8.8") {
		t.Fatal("scope was mutable")
	}
}
func TestScopeRejectsOverbroadAndMissingRanges(t *testing.T) {
	for _, cidr := range []string{"10.0.0.0/7", "172.16.0.0/8", "192.168.0.0/8", "fc00::/6", "0.0.0.0/0", "::/0", "bad"} {
		if _, err := NewScope(cidr); err == nil {
			t.Errorf("accepted %s", cidr)
		}
	}
	if _, err := NewScope(); err == nil {
		t.Fatal("accepted empty scope")
	}
	if (Scope{}).Allows("10.0.0.1") {
		t.Fatal("zero scope allowed traffic")
	}
}
