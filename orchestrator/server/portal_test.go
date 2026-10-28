package server

import "testing"

func TestPortalJWTRoundTripAndTypeGuard(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	tok, err := signPortalJWT(key, 7, 42)
	if err != nil {
		t.Fatal(err)
	}
	org, uid, ok := parsePortalJWT(tok, key)
	if !ok || org != 42 || uid != 7 {
		t.Fatalf("roundtrip failed: org=%d uid=%d ok=%v", org, uid, ok)
	}
	if _, _, ok := parsePortalJWT(tok, []byte("ffffffffffffffffffffffffffffffff")); ok {
		t.Fatal("a token signed with another key must not validate")
	}
	// A console token (no typ=portal) must NOT be accepted as a portal token.
	consoleTok, err := signJWT(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := parsePortalJWT(consoleTok, key); ok {
		t.Fatal("a console token must not be accepted as a portal token")
	}
}
