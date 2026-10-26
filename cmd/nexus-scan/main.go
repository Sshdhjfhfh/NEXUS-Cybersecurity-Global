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
	format := flag.String("format", "json", "report format: json or md")
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

	if !*skipNuclei {
		var sevs []string
		if s := strings.TrimSpace(*severity); s != "" {
			sevs = strings.Split(s, ",")
		}
		findings, err := scanner.Nuclei(ctx, scope.Engagement, *target, sevs)
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
	Findings    []scan.Finding `json:"findings"`
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
