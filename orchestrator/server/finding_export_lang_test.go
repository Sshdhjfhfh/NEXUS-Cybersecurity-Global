package server

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nexus-ia/nexus/governance/report"
)

func TestExportLang(t *testing.T) {
	cases := []struct {
		query, header string
		want          report.Lang
	}{
		{"", "", report.ES},
		{"", "en-US,en;q=0.9", report.EN},
		{"", "zh-CN,zh;q=0.9", report.ZH},
		{"lang=es", "en-US", report.ES}, // the panel's language wins over the browser's
		{"lang=en", "", report.EN},
		{"lang=fr", "en-US", report.ES}, // an explicit unsupported choice is not overridden
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/api/report?"+c.query, nil)
		if c.header != "" {
			r.Header.Set("Accept-Language", c.header)
		}
		if got := exportLang(r); got != c.want {
			t.Errorf("query %q, Accept-Language %q: got %q, want %q", c.query, c.header, got, c.want)
		}
	}
}

func TestFindingsExportFollowsLang(t *testing.T) {
	_, f, request := trafficEvidenceServer(t)
	for lang, header := range map[string]string{"": "ID,Nombre,", "&lang=en": "ID,Name,", "&lang=zh": "ID,名称,"} {
		w := request("GET", fmt.Sprintf("/api/exploration/findings/export?format=csv&scope=selected&ids=%d%s", f.FindingID, lang), "")
		if w.Code != 200 {
			t.Fatal(lang, w.Code, w.Body)
		}
		if body := strings.TrimPrefix(w.Body.String(), "\xEF\xBB\xBF"); !strings.HasPrefix(body, header) {
			t.Errorf("lang %q: CSV starts %q, want %q", lang, body[:min(len(body), 40)], header)
		}
	}
	w := request("GET", fmt.Sprintf("/api/exploration/findings/export?format=md-single&scope=selected&ids=%d", f.FindingID), "")
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "# Informe consolidado de hallazgos") {
		t.Fatalf("default Markdown export is not Spanish: %d %.80s", w.Code, w.Body)
	}
}
