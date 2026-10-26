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
	secretsDir := flag.String("secrets", "", "scan this local source dir for leaked secrets (gitleaks); must be an authorized source")
	scaDir := flag.String("sca", "", "scan this local source dir for vulnerable dependencies (osv-scanner); must be an authorized source")
	flag.Parse()

	// A web target is optional when only source scanning (secrets/SCA) is asked.
	sourceOnly := *target == "" && (*secretsDir != "" || *scaDir != "")
	if *scopePath == "" || (*target == "" && !sourceOnly) {
		fmt.Fprintln(os.Stderr, "usage: nexus-scan -scope scope.json [-target https://host] [-secrets DIR] [-sca DIR] [-severity critical,high] [-format json|md] [-skip-nuclei]")
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

	// Web scanning path (only when a target is given).
	if *target != "" {
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
	}

	// Source scanning path: leaked secrets (gitleaks) and vulnerable
	// dependencies / SCA (osv-scanner). Each directory must be an authorized
	// source in the engagement scope or the scan is refused before running.
	if *secretsDir != "" {
		secrets, err := scanner.Secrets(ctx, scope.Engagement, *secretsDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "secrets:", err)
			os.Exit(1)
		}
		rep.Secrets = secrets
	}
	if *scaDir != "" {
		deps, err := scanner.SCA(ctx, scope.Engagement, *scaDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sca:", err)
			os.Exit(1)
		}
		rep.DepVulns = deps
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
	Target      string         `json:"target,omitempty"`
	GeneratedAt string         `json:"generated_at"`
	Probes      []scan.Probe   `json:"probes,omitempty"`
	Discovered  []string       `json:"discovered,omitempty"`
	Findings    []scan.Finding `json:"findings,omitempty"`
	Secrets     []scan.Secret  `json:"secrets,omitempty"`
	DepVulns    []scan.DepVuln `json:"dep_vulns,omitempty"`
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
	if r.Target != "" {
		fmt.Fprintf(&b, "**Objetivo:** %s  \n", r.Target)
	}
	fmt.Fprintf(&b, "**Generado:** %s\n\n", r.GeneratedAt)

	// Web surface only when a web target was scanned.
	if r.Target != "" {
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

	if len(r.Probes) > 0 || len(r.Findings) > 0 || r.Target != "" {
		fmt.Fprintf(&b, "## Hallazgos web (%d)\n\n", len(r.Findings))
		if len(r.Findings) == 0 {
			fmt.Fprintf(&b, "_Sin hallazgos de las plantillas ejecutadas._\n\n")
		} else {
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
			fmt.Fprintln(&b)
		}
	}

	if len(r.Secrets) > 0 {
		fmt.Fprintf(&b, "## Secretos filtrados en el código (%d)\n\n", len(r.Secrets))
		fmt.Fprintf(&b, "| Regla | Archivo | Línea | Vista (redactada) |\n| --- | --- | --- | --- |\n")
		for _, s := range r.Secrets {
			fmt.Fprintf(&b, "| %s | %s | %d | %s |\n", s.RuleID, s.File, s.StartLine, s.Preview)
		}
		fmt.Fprintln(&b)
	}

	if len(r.DepVulns) > 0 {
		fmt.Fprintf(&b, "## Dependencias vulnerables / SCA (%d)\n\n", len(r.DepVulns))
		fmt.Fprintf(&b, "| Severidad | Paquete | Versión | ID | Resumen |\n| --- | --- | --- | --- | --- |\n")
		for _, d := range r.DepVulns {
			sev := d.Severity
			if sev == "" {
				sev = "—"
			}
			id := d.ID
			if len(d.Aliases) > 0 {
				id = fmt.Sprintf("%s (%s)", d.ID, strings.Join(d.Aliases, ", "))
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", sev, d.Package, d.Version, id, d.Summary)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}
