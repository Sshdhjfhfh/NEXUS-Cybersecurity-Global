// Nexus IA v6.0 — Absorbido de ARTEX/report
// Upstream: github.com/mhtsec/ARTEX/report
// Licencia: AGPL-3.0

// Package report renders a deliverable Markdown report from the exploration
// graph: confirmed findings (with severity/evidence) (docs §15 P5).
package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

// Input bundles what the report needs.
type Input struct {
	Title       string
	Goal        string
	GeneratedAt time.Time
	AssetCounts map[string]int
	Findings    []*db.Node
	Lang        Lang // zero value renders Spanish
}

type findingView struct {
	VulnClass string
	Name      string
	Severity  string
	Summary   string
	PoC       string
}

func parseFinding(n *db.Node) findingView {
	var p struct {
		VulnClass string `json:"vulnclass"`
		Name      string `json:"name"`
		Severity  string `json:"severity"`
		Summary   string `json:"summary"`
		Evidence  struct {
			PoC string `json:"poc"`
		} `json:"evidence"`
	}
	_ = json.Unmarshal(n.Payload, &p)
	return findingView{p.VulnClass, p.Name, p.Severity, p.Summary, p.Evidence.PoC}
}

var sevRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3, "": 4}

// Markdown renders the report.
func Markdown(in Input) string {
	t := in.Lang.t()
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — %s\n\n", t.reportTitle, nz(in.Title, t.untitledTask))
	b.WriteString(t.field(t.goal, nz(in.Goal, t.unspecified)))
	b.WriteString(t.field(t.generated, in.GeneratedAt.Format("2006-01-02 15:04:05")))
	b.WriteString("\n")

	// summary
	fmt.Fprintf(&b, "## %s\n\n", t.summary)
	fmt.Fprintf(&b, "- "+t.confirmedFmt+"\n", len(in.Findings))
	fmt.Fprintf(&b, "- %s%s", t.assets, t.colon)
	var types []string
	for ty := range in.AssetCounts {
		types = append(types, ty)
	}
	sort.Strings(types)
	for i, ty := range types {
		if i > 0 {
			b.WriteString(t.listSep)
		}
		fmt.Fprintf(&b, "%s %d", ty, in.AssetCounts[ty])
	}
	b.WriteString("\n\n")

	// findings
	fmt.Fprintf(&b, "## %s\n\n", t.findings)
	if len(in.Findings) == 0 {
		b.WriteString(t.noConfirmed + "\n\n")
	} else {
		fs := make([]findingView, 0, len(in.Findings))
		for _, n := range in.Findings {
			fs = append(fs, parseFinding(n))
		}
		sort.SliceStable(fs, func(i, j int) bool { return sevRank[fs[i].Severity] < sevRank[fs[j].Severity] })
		for i, f := range fs {
			fmt.Fprintf(&b, "### %d. %s %s\n\n", i+1, t.tag(f.Severity), nz(f.Name, nz(f.VulnClass, t.uncategorized)))
			fmt.Fprintf(&b, "%s\n\n", nz(f.Summary, ""))
			if f.PoC != "" {
				fmt.Fprintf(&b, "**%s%s**\n\n%s\n\n", t.pocEvidence, strings.TrimSpace(t.colon), codeBlock(f.PoC))
			}
		}
	}

	return b.String()
}

// codeBlock fences s with a backtick run longer than any inside it, so evidence
// taken from a hostile target cannot close the block and write report text.
func codeBlock(s string) string {
	longest, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))
	return fence + "\n" + s + "\n" + fence
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
