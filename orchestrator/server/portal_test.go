package server

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestPortalJWTRoundTripAndTypeGuard(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	tok, err := signPortalJWT(key, 7, 42, "sso", "sess-1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := parsePortalJWT(tok, key)
	if !ok || c.OrgID != 42 || c.UserID != 7 || c.SID != "sess-1" || c.AMR != "sso" {
		t.Fatalf("roundtrip failed: %+v ok=%v", c, ok)
	}
	if c.IAT <= 0 {
		t.Fatal("iat must be present for session-revocation checks")
	}
	if _, ok := parsePortalJWT(tok, []byte("ffffffffffffffffffffffffffffffff")); ok {
		t.Fatal("a token signed with another key must not validate")
	}
	// A token without a server-side session id cannot be signed out: refused.
	legacy, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "7", "org": 42, "typ": "portal", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parsePortalJWT(legacy, key); ok {
		t.Fatal("a portal token without sid must be refused")
	}
	// A console token (no typ=portal) must NOT be accepted as a portal token.
	consoleTok, err := signJWT(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := parsePortalJWT(consoleTok, key); ok {
		t.Fatal("a console token must not be accepted as a portal token")
	}
	// And the reverse: a portal token (same signing key) must NOT authenticate
	// the console, or a portal client could escalate to console/admin APIs.
	if verifyJWT(tok, key) {
		t.Fatal("a portal token must not be accepted as a console token")
	}
	if !verifyJWT(consoleTok, key) {
		t.Fatal("a genuine console token must still validate")
	}
}

func TestRoleHierarchy(t *testing.T) {
	// owner outranks admin outranks member; unknown roles have no privilege.
	if !(roleRank("owner") > roleRank("admin") && roleRank("admin") > roleRank("member")) {
		t.Fatal("role ranks must be strictly ordered owner > admin > member")
	}
	if roleRank("") != 0 || roleRank("superuser") != 0 {
		t.Fatal("unknown roles must rank 0")
	}
	// A member must not satisfy an admin requirement; an owner must.
	if roleRank("member") >= roleRank("admin") {
		t.Fatal("member must not satisfy an admin gate")
	}
	if roleRank("owner") < roleRank("admin") {
		t.Fatal("owner must satisfy an admin gate")
	}
}

func TestInviteRoleValidation(t *testing.T) {
	// Invitations may only grant admin or member, never owner or garbage.
	if !validInviteRole("admin") || !validInviteRole("member") {
		t.Fatal("admin/member must be invitable")
	}
	if validInviteRole("owner") || validInviteRole("") || validInviteRole("root") {
		t.Fatal("owner and unknown roles must not be invitable")
	}
	// validRole accepts the three real roles and rejects the rest.
	if !validRole("owner") || !validRole("admin") || !validRole("member") {
		t.Fatal("the three real roles must be valid")
	}
	if validRole("") || validRole("guest") {
		t.Fatal("unknown roles must be invalid")
	}
}

func TestAuditorRole(t *testing.T) {
	// The auditor is a real, invitable role that sits outside the rank ladder:
	// it never satisfies a rank gate, but may view admin-level org data.
	if !validRole("auditor") || !validInviteRole("auditor") {
		t.Fatal("auditor must be a valid, invitable role")
	}
	if roleRank("auditor") != 0 {
		t.Fatal("auditor must not satisfy any rank (write) gate")
	}
	if !canViewOrgAdmin("auditor") || !canViewOrgAdmin("admin") || !canViewOrgAdmin("owner") {
		t.Fatal("auditor, admin and owner see admin-level org data")
	}
	if canViewOrgAdmin("member") || canViewOrgAdmin("") {
		t.Fatal("members and unknown roles must not see admin-level org data")
	}
	if !isReadMethod("GET") || !isReadMethod("HEAD") || isReadMethod("POST") || isReadMethod("DELETE") {
		t.Fatal("only GET/HEAD are reads")
	}
	for path := range auditorSelfService {
		if !strings.HasPrefix(path, "/api/portal/me/") && !strings.HasPrefix(path, "/api/portal/mfa/") &&
			path != "/api/portal/password/change" && path != "/api/portal/email/resend" &&
			path != "/api/portal/terms/accept" { // consent is personal: each person accepts for themselves
			t.Fatalf("auditor write exception %q is not the auditor's own account", path)
		}
	}
	if !isAuditorSelfService("/api/portal/sessions/abc") || !isAuditorSelfService("/api/portal/sessions/revoke-others") {
		t.Fatal("an auditor may sign out their own sessions")
	}
	for _, p := range []string{"/api/portal/sessions", "/api/portal/requests", "/api/portal/members/3/revoke-sessions"} {
		if isAuditorSelfService(p) {
			t.Fatalf("%q must not be an auditor write exception", p)
		}
	}
}

func TestDescribeUserAgent(t *testing.T) {
	cases := []struct{ ua, browser, os string }{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0", "Edge", "Windows"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36", "Chrome", "Windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15", "Safari", "macOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", "Safari", "iOS"},
		{"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36", "Chrome", "Android"},
		{"Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0", "Firefox", "Linux"},
		{"curl/8.5.0", "curl", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		b, o := describeUserAgent(c.ua)
		if b != c.browser || o != c.os {
			t.Errorf("%q -> %q/%q, want %q/%q", c.ua, b, o, c.browser, c.os)
		}
	}
	if got := deviceText("Chrome", "Windows", "en", "?"); got != "Chrome en Windows" {
		t.Errorf("deviceText = %q", got)
	}
	if got := deviceText("", "", "on", "unknown browser"); got != "unknown browser" {
		t.Errorf("deviceText unknown = %q", got)
	}
}

func TestPortalPreferenceValidation(t *testing.T) {
	// The view profile is presentational only; empty means "default by role".
	for _, v := range []string{"", "executive", "technical"} {
		if !validViewProfile(v) {
			t.Fatalf("view profile %q must be valid", v)
		}
	}
	for _, v := range []string{"admin", "owner", "EXECUTIVE", "<script>"} {
		if validViewProfile(v) {
			t.Fatalf("view profile %q must be rejected", v)
		}
	}
	// Only languages with a complete translation are accepted.
	for _, v := range []string{"", "es", "en"} {
		if !validPortalLang(v) {
			t.Fatalf("lang %q must be valid", v)
		}
	}
	for _, v := range []string{"fr", "es-ES", "zz"} {
		if validPortalLang(v) {
			t.Fatalf("lang %q must be rejected", v)
		}
	}
}

func TestInviteTokenUniqueAndURLSafe(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 256; i++ {
		tok, err := newInviteToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(tok) < 24 {
			t.Fatalf("token too short: %q", tok)
		}
		if seen[tok] {
			t.Fatalf("duplicate token generated: %q", tok)
		}
		seen[tok] = true
		// base64url (RawURLEncoding) must not contain +, / or =.
		if strings.ContainsAny(tok, "+/=") {
			t.Fatalf("token not URL-safe: %q", tok)
		}
	}
}
