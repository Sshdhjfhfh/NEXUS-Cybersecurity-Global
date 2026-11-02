// Nexus IA v6.0 — Absorbido de ARTEX/report
// Upstream: github.com/mhtsec/ARTEX/report
// Licencia: AGPL-3.0

package report

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nexus-ia/nexus/db"
)

// 发现页「导出」用的渲染:把一批 findings 表行渲染成汇总 Markdown、单条 Markdown、
// 或 CSV。JSON 由 server 层直接用 DTO 序列化,不在此处。

// sortFindingsForExport 按严重等级降序、再按时间倒序排,与汇总报告的分组一致。
func sortFindingsForExport(fs []*db.DBFinding) {
	sort.SliceStable(fs, func(i, j int) bool {
		ri, rj := sevRank[fs[i].Severity], sevRank[fs[j].Severity]
		if ri != rj {
			return ri < rj // sevRank 越小越严重
		}
		return fs[i].CreatedAt.After(fs[j].CreatedAt)
	})
}

// findingTitle 取漏洞可读标题:名称 → 类别 → 「未分类」。
func findingTitle(f *db.DBFinding, t *texts) string {
	return nz(f.Name, nz(f.VulnClass, t.uncategorized))
}

// FindingsMarkdown 把一批 findings 整合成一份汇总报告(摘要 + 按严重等级分组,
// 每条含类别/状态/所属任务/证据/详细报告)。
func FindingsMarkdown(fs []*db.DBFinding, generatedAt time.Time, lang Lang) string {
	t := lang.t()
	items := append([]*db.DBFinding(nil), fs...)
	sortFindingsForExport(items)

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", t.exportTitle)
	b.WriteString(t.field(t.generated, generatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(t.field(t.total, fmt.Sprintf(t.countFmt, len(items))))
	b.WriteString("\n")

	// 摘要:各严重等级计数。
	counts := map[string]int{}
	for _, f := range items {
		counts[f.Severity]++
	}
	fmt.Fprintf(&b, "## %s\n\n", t.summary)
	fmt.Fprintf(&b, "| %s | %s |\n| --- | --- |\n", t.severity, t.count)
	for _, s := range []string{"critical", "high", "medium", "low"} {
		fmt.Fprintf(&b, "| %s | %d |\n", t.sev(s), counts[s])
	}
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString(t.noMatches + "\n")
		return b.String()
	}

	fmt.Fprintf(&b, "## %s\n\n", t.details)
	for i, f := range items {
		fmt.Fprintf(&b, "### %d. %s %s\n\n", i+1, t.tag(f.Severity), findingTitle(f, t))
		if f.VulnClass != "" {
			b.WriteString(t.field(t.class, f.VulnClass))
		}
		b.WriteString(t.field(t.status, t.stat(f.Status)))
		if desc := strings.TrimSpace(f.TaskDescription); desc != "" {
			b.WriteString(t.field(t.task, desc))
		}
		b.WriteString(t.field(t.foundAt, f.CreatedAt.Format("2006-01-02 15:04:05")))
		b.WriteString("\n")
		if s := strings.TrimSpace(f.Summary); s != "" {
			fmt.Fprintf(&b, "%s\n\n", s)
		}
		if e := strings.TrimSpace(f.Evidence); e != "" {
			fmt.Fprintf(&b, "**%s%s**\n\n%s\n\n", t.evidence, strings.TrimSpace(t.colon), codeBlock(e))
		}
		if rep := strings.TrimSpace(f.Report); rep != "" {
			fmt.Fprintf(&b, "**%s%s**\n\n", t.detailReport, strings.TrimSpace(t.colon))
			b.WriteString(rep)
			b.WriteString("\n\n")
		}
		b.WriteString(findingTrafficMarkdown(f, t, false))
		b.WriteString("---\n\n")
	}
	return b.String()
}

// SingleFindingMarkdown 渲染单条漏洞为一份独立 Markdown(用于「一漏洞一文件」打包)。
func SingleFindingMarkdown(f *db.DBFinding, generatedAt time.Time, lang Lang) string {
	t := lang.t()
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n", t.tag(f.Severity), findingTitle(f, t))
	if f.VulnClass != "" {
		b.WriteString(t.field(t.class, f.VulnClass))
	}
	b.WriteString(t.field(t.severity, t.sev(f.Severity)))
	b.WriteString(t.field(t.status, t.stat(f.Status)))
	if desc := strings.TrimSpace(f.TaskDescription); desc != "" {
		b.WriteString(t.field(t.task, desc))
	}
	b.WriteString(t.field(t.foundAt, f.CreatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(t.field(t.generated, generatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")
	if s := strings.TrimSpace(f.Summary); s != "" {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", t.overview, s)
	}
	if e := strings.TrimSpace(f.Evidence); e != "" {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", t.evidence, codeBlock(e))
	}
	if rep := strings.TrimSpace(f.Report); rep != "" {
		fmt.Fprintf(&b, "## %s\n\n", t.detailReport)
		b.WriteString(rep)
		b.WriteString("\n")
	}
	b.WriteString(findingTrafficMarkdown(f, t, true))
	return b.String()
}

var unsafeFilenameChars = regexp.MustCompile(`[^\p{Han}\p{L}\p{N}._-]+`)

// FindingFilename 为「一漏洞一文件」生成安全的 .md 文件名,形如
// `critical_SQL注入__123.md`。去掉路径分隔符与控制字符,避免 zip 内非法路径。
func FindingFilename(f *db.DBFinding, lang Lang) string {
	sev := nz(f.Severity, "info")
	title := findingTitle(f, lang.t())
	name := fmt.Sprintf("%s_%s_#%d", sev, title, f.ID)
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._")
	if name == "" {
		name = fmt.Sprintf("finding_%d", f.ID)
	}
	// 防御性:再剥一层路径,杜绝 zip slip。
	name = path.Base(name)
	if len(name) > 120 {
		// Cut on a rune boundary: zip entry names must stay valid UTF-8.
		cut := 120
		for cut > 0 && !utf8.RuneStart(name[cut]) {
			cut--
		}
		name = name[:cut]
	}
	return name + ".md"
}

// FindingsCSV 把一批 findings 渲染成 CSV(带 UTF-8 BOM,便于 Excel 正确识别中文)。
// 不含大段 report/evidence 全文,只放摘要类字段;需要全文用 Markdown/JSON 导出。
func FindingsCSV(fs []*db.DBFinding, lang Lang) []byte {
	t := lang.t()
	items := append([]*db.DBFinding(nil), fs...)
	sortFindingsForExport(items)

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM
	w := csv.NewWriter(&buf)
	_ = w.Write(t.csvHeader)
	for _, f := range items {
		_ = w.Write([]string{
			fmt.Sprintf("%d", f.ID),
			csvText(findingTitle(f, t)),
			csvText(f.VulnClass),
			csvText(t.sev(f.Severity)),
			csvText(t.stat(f.Status)),
			csvText(f.TaskDescription),
			f.CreatedAt.Format("2006-01-02 15:04:05"),
			csvText(strings.TrimSpace(f.Summary)),
			fmt.Sprint(len(f.TrafficBindings)), findingTrafficIDs(f),
		})
	}
	w.Flush()
	return buf.Bytes()
}

// csvText keeps a spreadsheet from evaluating a cell as a formula: text that
// starts with = + - @ tab or CR gets a leading apostrophe (OWASP CSV injection).
// These fields can carry text from the target under test.
func csvText(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func findingTrafficIDs(f *db.DBFinding) string {
	ids := make([]string, 0, len(f.TrafficBindings))
	for _, b := range f.TrafficBindings {
		ids = append(ids, fmt.Sprint(b.ID))
	}
	return strings.Join(ids, ",")
}

func findingTrafficMarkdown(f *db.DBFinding, t *texts, attachments bool) string {
	stale := f.Report != "" && f.EvidenceVersion != f.ReportEvidenceVersion
	if len(f.TrafficBindings) == 0 && !stale {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, "\n## %s\n\n", t.traffic)
	fmt.Fprintf(&out, t.trafficMetaFmt+"\n\n", f.EvidenceVersion, len(f.TrafficBindings))
	if stale {
		out.WriteString(t.trafficStale + "\n\n")
	}
	for i, b := range f.TrafficBindings {
		fmt.Fprintf(&out, t.trafficItemFmt, i+1, b.ID, b.Role, b.Snapshot.Method, strings.ReplaceAll(b.Snapshot.URL, "`", "%60"), b.Snapshot.Status)
		if b.Note != "" {
			fmt.Fprintf(&out, "   %s\n", strings.ReplaceAll(b.Note, "\n", "\n   "))
		}
		if attachments {
			fmt.Fprintf(&out, "   [%s](evidence/%d/%d/request.http) · [%s](evidence/%d/%d/response.http)\n", t.request, f.ID, b.ID, t.response, f.ID, b.ID)
		}
	}
	out.WriteString("\n")
	return out.String()
}
