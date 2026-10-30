package server

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestCSVSafeNeutralisesFormulas(t *testing.T) {
	for _, v := range []string{"=HYPERLINK(\"http://x\")", "+1", "-2+3", "@SUM(A1)", "\tx", "\rx"} {
		if got := csvSafe(v); got != "'"+v {
			t.Errorf("csvSafe(%q) = %q, want it prefixed", v, got)
		}
	}
	for _, v := range []string{"", "ana@example.test", "auth.login", "203.0.113.5", "count=2"} {
		if got := csvSafe(v); got != v {
			t.Errorf("csvSafe(%q) = %q, want unchanged", v, got)
		}
	}
}

func TestAuditFilterFrom(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/portal/audit?category=access&q=ana&from=2026-10-01&to=2026-10-30", nil)
	f, err := auditFilterFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Prefixes) == 0 || f.Prefixes[0] != "auth." || f.Query != "ana" {
		t.Fatalf("filter = %+v", f)
	}
	if !f.From.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) || !f.To.Equal(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("dates: from=%v to=%v (to must include the whole last day)", f.From, f.To)
	}
	for _, bad := range []string{"category=nope", "from=30/10/2026", "from=2026-10-30&to=2026-10-01", "to=2026-13-01"} {
		if _, err := auditFilterFrom(httptest.NewRequest("GET", "/api/portal/audit?"+bad, nil)); err == nil {
			t.Errorf("%s must be rejected", bad)
		}
	}
	if f, err := auditFilterFrom(httptest.NewRequest("GET", "/api/portal/audit?from=2026-10-30&to=2026-10-30", nil)); err != nil || f.To.Sub(f.From) != 24*time.Hour {
		t.Errorf("a single day must be a valid range: %+v %v", f, err)
	}
}
