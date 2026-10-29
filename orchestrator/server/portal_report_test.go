package server

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/db"
)

func sampleReportInput(kind, lang string) *reportInput {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	owner := int64(7)
	return &reportInput{
		Kind: kind, Lang: lang, OrgName: "Ministerio de Ejemplo", Title: "Evaluación web Q4 — portal ciudadano",
		Scope: "portal.example.gob\napi.example.gob", RequestID: 42, RequestedAt: now.AddDate(0, 0, -20),
		GeneratedAt: now,
		Items: []*db.RemediationItem{
			{FindingID: 1, Name: "Inyección SQL en /buscar", VulnClass: "sqli", Severity: "critical", State: db.RemOpen,
				Bucket: "open", DetectedAt: now.AddDate(0, 0, -10), DueDate: "2026-10-25", DueSource: "sla", Overdue: true,
				Summary: "El parámetro q se concatena en la consulta."},
			{FindingID: 2, Name: "XSS reflejado", VulnClass: "xss", Severity: "high", State: db.RemInProgress,
				Bucket: "in_progress", DetectedAt: now.AddDate(0, 0, -10), DueDate: "2026-11-17", DueSource: "sla",
				AssigneeID: &owner, AssigneeEmail: "equipo@example.org"},
			{FindingID: 3, Name: "Cabeceras de seguridad ausentes", VulnClass: "config", Severity: "low",
				State: db.RemRiskAccepted, Bucket: "risk_accepted", DetectedAt: now.AddDate(0, 0, -10),
				RiskReason: "Sistema heredado aislado; sustitución en Q2.", RiskExpiresAt: "2027-01-31", RiskAcceptedBy: "ciso@example.org"},
			{FindingID: 4, Name: "", VulnClass: "info-leak", Severity: "", State: db.RemVerifiedFixed, Bucket: "remediated",
				DetectedAt: now.AddDate(0, 0, -10)},
		},
		Reports: map[int64]string{
			1: "# Detalle\n\nLa petición **vulnerable**:\n\n```http\nGET /buscar?q=1'+OR+'1'='1 HTTP/1.1\nHost: portal.example.gob\n```\n\n- Paso uno con `código`\n- Paso [dos](https://example.org)\n  - anidado\n1. numerado\n> cita\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n---\nTexto con emoji 🚀 y chino 漏洞 y\ttab.",
			2: strings.Repeat("Línea larga sin espacios: "+strings.Repeat("A", 300)+"\n", 3),
		},
	}
}

func TestRenderReports(t *testing.T) {
	for _, kind := range []string{reportExecutive, reportTechnical} {
		for _, lang := range []string{"es", "en"} {
			in := sampleReportInput(kind, lang)
			pdf, err := renderReport(in)
			if err != nil {
				t.Fatalf("%s/%s: %v", kind, lang, err)
			}
			if !bytes.HasPrefix(pdf, []byte("%PDF-")) || len(pdf) < 20000 {
				t.Fatalf("%s/%s: not a PDF (%d bytes)", kind, lang, len(pdf))
			}
			if dir := os.Getenv("NEXUS_REPORT_SAMPLES"); dir != "" {
				_ = os.WriteFile(dir+"/sample-"+kind+"-"+lang+".pdf", pdf, 0o644)
			}
		}
	}
	// Empty evaluation and draft watermark must render too.
	in := sampleReportInput(reportExecutive, "es")
	in.Items, in.Reports, in.Draft = nil, nil, true
	if _, err := renderReport(in); err != nil {
		t.Fatalf("empty draft: %v", err)
	}
}

func TestCleanStripsUnsupportedRunes(t *testing.T) {
	got := clean("a\x00b\tc🚀d\r")
	if got != "ab    cd" {
		t.Fatalf("clean = %q", got)
	}
}

func TestGlyphCoverageFromCmap(t *testing.T) {
	sets := reportGlyphSets()
	for _, style := range []string{"R", "SB", "B", "M"} {
		cov := sets[style]
		for _, r := range "AzáéíóúñÑü—·€" {
			if !cov[r] {
				t.Errorf("%s: expected glyph for %q", style, r)
			}
		}
		if cov['漏'] {
			t.Errorf("%s: unexpected CJK glyph", style)
		}
	}
	p := &rpt{curFont: "R"}
	if got := p.s("Año 漏洞 ok"); got != "Año ?? ok" {
		t.Fatalf("s = %q", got)
	}
}

func TestReportSlug(t *testing.T) {
	if s := reportSlug("Evaluación Web Q4 — Ñandú!"); s != "evaluacion-web-q4-nandu" {
		t.Fatalf("slug = %q", s)
	}
	if s := reportSlug("🚀"); s != "evaluacion" {
		t.Fatalf("empty slug = %q", s)
	}
}

func TestRecommendationsFollowData(t *testing.T) {
	p := &rpt{t: reportTexts["es"]}
	st := computeStats(sampleReportInput(reportExecutive, "es").Items)
	if st.exposure != "critical" || st.overdue != 1 || st.unassignedOpen != 1 || st.total.risk != 1 {
		t.Fatalf("stats = %+v", st)
	}
	recs := strings.Join(p.recommendations(st), "\n")
	for _, want := range []string{"7 días", "30 días", "fecha objetivo", "responsable", "verificación de NEXUS", "riesgo aceptado"} {
		if !strings.Contains(recs, want) {
			t.Errorf("recommendations missing %q:\n%s", want, recs)
		}
	}
	empty := computeStats(nil)
	if empty.exposure != "none" || len(p.recommendations(empty)) != 1 {
		t.Fatalf("empty org should only get the periodic-retest advice")
	}
}
