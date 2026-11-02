package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTermsGate(t *testing.T) {
	cases := []struct {
		accepted, path string
		allowed        bool
	}{
		{portalTermsVersion, "/api/portal/requests", true},
		{"", "/api/portal/requests", false},
		{"2020-01-old", "/api/portal/remediation", false},
		{"", "/api/portal/me", true},
		{"", "/api/portal/terms/accept", true},
		{"", "/api/portal/logout", true},
		{"", "/api/portal/mfa/setup", false},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, c.path, nil)
		if got := termsGate(w, r, c.accepted); got != c.allowed {
			t.Errorf("termsGate(%q, %s) = %v, want %v", c.accepted, c.path, got, c.allowed)
		}
		if !c.allowed && w.Code != 403 {
			t.Errorf("%s: refused with %d, want 403", c.path, w.Code)
		}
	}
	// Accepting must stay possible for an auditor and during 2FA enrollment.
	if !isAuditorSelfService("/api/portal/terms/accept") || !mfaEnrollmentPaths["/api/portal/terms/accept"] {
		t.Error("terms acceptance must be allowed to auditors and during 2FA enrollment")
	}
}

func TestTermsConsent(t *testing.T) {
	cases := []struct {
		accept  bool
		version string
		code    int // 0 = accepted
	}{
		{true, "", 0},
		{true, portalTermsVersion, 0},
		{false, portalTermsVersion, 400},
		{false, "", 400},
		{true, "2020-01-old", 409},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		ok := termsConsent(w, c.accept, c.version)
		if ok != (c.code == 0) || (c.code != 0 && w.Code != c.code) {
			t.Errorf("termsConsent(%v, %q) = %v/%d, want code %d", c.accept, c.version, ok, w.Code, c.code)
		}
	}
}
