package server

import (
	"strings"
	"testing"
)

func TestPortalPasswordProblem(t *testing.T) {
	email := "ana.garcia@example-org.test"
	refused := map[string]string{
		"short1":                errPasswordLength,
		strings.Repeat("x", 73): errPasswordLength,
		"password":              errPasswordCommon,
		"Password2026!":         errPasswordCommon,
		"P@ssw0rd123":           errPasswordCommon,
		"password123":           errPasswordCommon,
		"12345678":              errPasswordCommon,
		"aaaaaaaa":              errPasswordCommon,
		"abcdefgh":              errPasswordCommon,
		"98765432":              errPasswordCommon,
		"QWERTYUIOP":            errPasswordCommon,
		"asdfghjk":              errPasswordCommon,
		"iloveyou99":            errPasswordCommon,
		"Contraseña1":           errPasswordCommon,
		"ana.garcia2026":        errPasswordEmail,
		"x" + email + "x":       errPasswordEmail,
	}
	for pw, want := range refused {
		if got := portalPasswordProblem(pw, email); got != want {
			t.Errorf("%q: got %q, want %q", pw, got, want)
		}
	}
	for _, pw := range []string{"Vx9-e2e-Fjord-2026", "correct horse battery staple", "dragonfly-sunset", "Tr0ub4dor&3x", "garcia-river-77"} {
		if got := portalPasswordProblem(pw, email); got != "" {
			t.Errorf("%q must be accepted, got %q", pw, got)
		}
	}
	// A very short local part ("al@") is not a meaningful thing to forbid.
	if got := portalPasswordProblem("albatross-flight", "al@example-org.test"); got != "" {
		t.Errorf("short local part must not block: %q", got)
	}
	// Without a known email only the generic rules apply.
	if got := portalPasswordProblem("ana.garcia2026", ""); got != "" {
		t.Errorf("no email known: %q", got)
	}
}
