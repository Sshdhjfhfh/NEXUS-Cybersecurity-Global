// Command nexus-scan runs an authorized vulnerability scan for a NEXUS
// engagement and prints a findings report (JSON or Markdown).
//
// SAFETY: it refuses any target outside the signed engagement scope. It only
// detects and reports — it does not exploit. This is an operator tool for the
// services model (deliver audits), independent of the AGPL orchestrator.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/agents/red/scan"
	"github.com/nexus-ia/nexus/lab/engagement"
)

func main() {
	scopePath := flag.String("scope", "", "path to engagement scope JSON (required)")
	target := flag.String("target", "", "target URL to scan, must be in scope (required)")
	severity := flag.String("severity", "", "comma-separated nuclei severities, e.g. critical,high (optional)")
	tags := flag.String("tags", "", "comma-separated nuclei tags to scope the scan, e.g. exposure,misconfig (optional)")
	templates := flag.String("templates", "", "path to a nuclei-templates checkout (optional; overrides auto-update)")
	format := flag.String("format", "json", "report format: json or md")
	recon := flag.Bool("recon", false, "crawl the target first (katana) and scan the discovered URLs too")
	depth := flag.Int("depth", 2, "crawl depth when -recon is set")
	skipNuclei := flag.Bool("skip-nuclei", false, "run only httpx discovery (no vulnerability templates)")
	flag.Parse()

	if *scopePath == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "usage: nexus-scan -scope scope.json -target https://host [-severity critical,high] [-format json|md] [-skip-nuclei]")
		os.Exit(2)
	}

	scope, err := engagement.LoadScope(*scopePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope:", err)
		os.Exit(1)
	}
	auth, err := engagement.NewAuthorizer(scope)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope:", err)
		os.Exit(1)
	}
	scanner, err := scan.New(auth, "nexus-scan")
	if err != nil {
		fmt.Fprintln(os.Stderr, "scanner:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	rep := report{
		Engagement:  scope.Engagement,
		Client:      scope.Client,
		Target:      *target,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	probes, err := scanner.HTTPX(ctx, scope.Engagement, []string{*target})
	if err != nil {
		fmt.Fprintln(os.Stderr, "httpx:", err)
		os.Exit(1)
	}
	rep.Probes = probes

	// Build the target list: the base target, plus any URLs discovered by the
	// crawler when -recon is set. This is the "recon chain" — discover more
	// surface, then scan all of it.
	targets := []string{*target}
	if *recon {
		discovered, err := scanner.Crawl(ctx, scope.Engagement, *target, *depth)
		if err != nil {
			fmt.Fprintln(os.Stderr, "crawl:", err)
			os.Exit(1)
		}
		rep.Discovered = discovered
		targets = dedupe(append(targets, discovered...))
	}

	if !*skipNuclei {
		opt := scan.NucleiOptions{}
		if s := strings.TrimSpace(*severity); s != "" {
			opt.Severities = strings.Split(s, ",")
		}
		if s := strings.TrimSpace(*tags); s != "" {
			opt.Tags = strings.Split(s, ",")
		}
		opt.TemplatesDir = strings.TrimSpace(*templates)
		findings, err := scanner.NucleiTargets(ctx, scope.Engagement, targets, opt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nuclei:", err)
			os.Exit(1)
		}
		rep.Findings = findings
	}

	switch *format {
	case "md":
		fmt.Print(rep.markdown())
	default:
		out, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(out))
	}
}

type report struct {
	Engagement  string         `json:"engagement"`
	Client      string         `json:"client,omitempty"`
	Target      string         `json:"target"`
	GeneratedAt string         `json:"generated_at"`
	Probes      []scan.Probe   `json:"probes"`
	Discovered  []string       `json:"discovered,omitempty"`
	Findings    []scan.Finding `json:"findings"`
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func (r report) markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Informe de seguridad — %s\n\n", r.Engagement)
	if r.Client != "" {
		fmt.Fprintf(&b, "**Cliente:** %s  \n", r.Client)
	}
	fmt.Fprintf(&b, "**Objetivo:** %s  \n**Generado:** %s\n\n", r.Target, r.GeneratedAt)

	fmt.Fprintf(&b, "## Superficie detectada\n\n")
	if len(r.Probes) == 0 {
		fmt.Fprintf(&b, "_Sin endpoints web vivos detectados._\n\n")
	} else {
		fmt.Fprintf(&b, "| URL | Estado | Título | Servidor |\n| --- | --- | --- | --- |\n")
		for _, p := range r.Probes {
			fmt.Fprintf(&b, "| %s | %d | %s | %s |\n", p.URL, p.Status, p.Title, p.Webserver)
		}
		fmt.Fprintln(&b)
	}

	if len(r.Discovered) > 0 {
		fmt.Fprintf(&b, "## Superficie descubierta por crawling (%d URLs)\n\n", len(r.Discovered))
		max := len(r.Discovered)
		if max > 50 {
			max = 50
		}
		for _, u := range r.Discovered[:max] {
			fmt.Fprintf(&b, "- %s\n", u)
		}
		if len(r.Discovered) > 50 {
			fmt.Fprintf(&b, "- _… y %d más_\n", len(r.Discovered)-50)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintf(&b, "## Hallazgos (%d)\n\n", len(r.Findings))
	if len(r.Findings) == 0 {
		fmt.Fprintf(&b, "_Sin hallazgos de las plantillas ejecutadas._\n")
		return b.String()
	}
	fmt.Fprintf(&b, "| Severidad | Hallazgo | Dónde | CVE | CWE |\n| --- | --- | --- | --- | --- |\n")
	for _, f := range r.Findings {
		cve, cwe := f.CVE, f.CWE
		if cve == "" {
			cve = "—"
		}
		if cwe == "" {
			cwe = "—"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", strings.ToUpper(f.Severity), f.Name, f.MatchedAt, cve, cwe)
	}
	return b.String()
}
