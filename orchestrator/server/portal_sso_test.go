package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nexus-ia/nexus/db"
)

func TestCheckIdPURL(t *testing.T) {
	t.Setenv("NEXUS_SSO_DEV_INSECURE", "")
	if _, err := checkIdPURL("https://login.example.com/tenant/v2.0"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"http://login.example.com", "ftp://x", "https://user:pw@idp.example.com", "https://", "javascript:alert(1)"} {
		if _, err := checkIdPURL(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	t.Setenv("NEXUS_SSO_DEV_INSECURE", "1")
	if _, err := checkIdPURL("http://127.0.0.1:9999"); err != nil {
		t.Error("dev mode must allow a local http issuer")
	}
}

func TestVerifyIDToken(t *testing.T) {
	t.Setenv("NEXUS_SSO_DEV_INSECURE", "1")
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	b64 := base64.RawURLEncoding
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": "k1", "use": "sig",
			"n": b64.EncodeToString(key.N.Bytes()), "e": b64.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer srv.Close()
	cfg := &db.PortalSSO{Issuer: "https://idp.example.com", ClientID: "nexus", EmailDomains: []string{"acme.example"}}
	disc := &oidcDiscovery{Issuer: cfg.Issuer, JWKSURI: srv.URL + "/jwks"}
	now := time.Now()
	base := func() jwt.MapClaims {
		return jwt.MapClaims{
			"iss": cfg.Issuer, "aud": "nexus", "sub": "abc", "email": "Ana@ACME.example", "email_verified": true,
			"nonce": "n1", "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
		}
	}
	sign := func(c jwt.MapClaims, k *rsa.PrivateKey) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
		tok.Header["kid"] = "k1"
		s, err := tok.SignedString(k)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	ctx := context.Background()

	id, err := verifyIDToken(ctx, cfg, disc, sign(base(), key), "n1")
	if err != nil {
		t.Fatalf("valid token refused: %v", err)
	}
	if id.Email != "ana@acme.example" || id.Subject != "https://idp.example.com|abc" {
		t.Fatalf("identity = %+v", id)
	}

	with := func(k string, v any) jwt.MapClaims { c := base(); c[k] = v; return c }
	without := func(k string) jwt.MapClaims { c := base(); delete(c, k); return c }
	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, base())
	hs.Header["kid"] = "k1"
	hsTok, _ := hs.SignedString(key.N.Bytes()) // algorithm-confusion attempt
	noneTok, _ := jwt.NewWithClaims(jwt.SigningMethodNone, base()).SignedString(jwt.UnsafeAllowNoneSignatureType)
	cases := map[string]string{
		"signed by another key":       sign(base(), other),
		"wrong issuer":                sign(with("iss", "https://evil.example.com"), key),
		"wrong audience":              sign(with("aud", "someone-else"), key),
		"expired":                     sign(with("exp", now.Add(-10*time.Minute).Unix()), key),
		"no expiry":                   sign(without("exp"), key),
		"domain not claimed":          sign(with("email", "ana@other.example"), key),
		"email explicitly unverified": sign(with("email_verified", false), key),
		"no sub":                      sign(without("sub"), key),
		"several audiences, no azp":   sign(with("aud", []string{"nexus", "other"}), key),
		"alg none":                    noneTok,
		"HS256 with the public key":   hsTok,
		"garbage":                     "a.b.c",
	}
	for name, tok := range cases {
		if _, err := verifyIDToken(ctx, cfg, disc, tok, "n1"); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := verifyIDToken(ctx, cfg, disc, sign(base(), key), "other-nonce"); err == nil {
		t.Error("wrong nonce accepted")
	}
	c := with("aud", []string{"nexus", "other"})
	c["azp"] = "nexus"
	if _, err := verifyIDToken(ctx, cfg, disc, sign(c, key), "n1"); err != nil {
		t.Errorf("several audiences with matching azp refused: %v", err)
	}
	if _, err := verifyIDToken(ctx, cfg, disc, sign(without("email_verified"), key), "n1"); err != nil {
		t.Errorf("absent email_verified (Entra ID) refused: %v", err)
	}
}
