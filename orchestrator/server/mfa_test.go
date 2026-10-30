package server

import (
	"testing"
	"time"
)

func TestTOTPRoundTrip(t *testing.T) {
	secret := newTOTPSecret()
	code, err := totpAt(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !verifyTOTP(secret, code) {
		t.Fatal("the current code must verify")
	}
	if verifyTOTP(secret, "12345") {
		t.Fatal("a 5-digit code must be rejected")
	}
	// A code from a far-away window must not verify.
	old, _ := totpAt(secret, time.Now().Add(-10*time.Minute))
	if old != code && verifyTOTP(secret, old) {
		t.Fatal("a code from a distant window must not verify")
	}
}

func TestConsoleTOTPIsSingleUse(t *testing.T) {
	consoleTOTP.Lock()
	consoleTOTP.last = 0
	consoleTOTP.Unlock()
	secret := newTOTPSecret()
	now, _ := totpAt(secret, time.Now())
	prev, _ := totpAt(secret, time.Now().Add(-totpPeriod*time.Second))
	if !verifyConsoleTOTPOnce(secret, now) {
		t.Fatal("a fresh code must be accepted")
	}
	if verifyConsoleTOTPOnce(secret, now) {
		t.Fatal("the same code must not be accepted twice")
	}
	if prev != now && verifyConsoleTOTPOnce(secret, prev) {
		t.Fatal("an older code must not be accepted after a newer one")
	}
}

func TestEncryptDecryptSecretRoundTrip(t *testing.T) {
	s := &Server{jwtKey: []byte("0123456789abcdef0123456789abcdef")}
	enc, err := s.encryptSecret("HELLO-SECRET")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.decryptSecret(enc)
	if err != nil || got != "HELLO-SECRET" {
		t.Fatalf("roundtrip failed: %q %v", got, err)
	}
	if _, err := s.decryptSecret(enc + "AAAA"); err == nil {
		t.Fatal("tampered ciphertext must fail authentication")
	}
	s2 := &Server{jwtKey: []byte("ffffffffffffffffffffffffffffffff")}
	if _, err := s2.decryptSecret(enc); err == nil {
		t.Fatal("a different key must not decrypt the secret")
	}
}
