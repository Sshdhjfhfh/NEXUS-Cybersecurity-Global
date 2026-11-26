// Package scan wraps external security scanners (httpx, nuclei) as structured,
// authorization-gated operations for the NEXUS red agent.
//
// SAFETY: every target is checked against the laboratory gate (lab.Authorizer)
// BEFORE any process is started, so a target outside the authorized scope is
// never contacted. These wrappers DETECT and REPORT only — nuclei runs
// detection templates (CVEs, misconfigurations, exposed web shells/backdoors,
// default credentials); it does not exploit. Findings are meant to be reported
// to the audited organization.
package scan

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nexus-ia/nexus/lab"
)

// Scanner runs authorized external scanners. Bin fields default to the tool name
// resolved on PATH; override them to pin a lab toolchain location.
type Scanner struct {
	HTTPXBin    string
	NucleiBin   string
	CrawlBin    string
	GitleaksBin string
	OSVBin      string
	gate        lab.Authorizer
	agentID     string
}

// New returns a Scanner bound to a laboratory authorizer and agent identity.
func New(gate lab.Authorizer, agentID string) (*Scanner, error) {
	if gate == nil || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("scanner requires a laboratory authorizer and agent id")
	}
	return &Scanner{
		HTTPXBin: "httpx", NucleiBin: "nuclei", CrawlBin: "katana",
		GitleaksBin: "gitleaks", OSVBin: "osv-scanner",
		gate: gate, agentID: agentID,
	}, nil
}

// Probe is a live web endpoint discovered by httpx.
type Probe struct {
	URL         string   `json:"url"`
	Scheme      string   `json:"scheme"`
	Title       string   `json:"title"`
	Webserver   string   `json:"webserver"`
	ContentType string   `json:"content_type"`
	Host        string   `json:"host"`
	Status      int      `json:"status"`
	Tech        []string `json:"tech,omitempty"`
}

// Finding is a vulnerability/exposure reported by nuclei, ready to hand to the
// audited organization.
type Finding struct {
	TemplateID  string   `json:"template_id"`
	Name        string   `json:"name"`
	Severity    string   `json:"severity"`
	MatchedAt   string   `json:"matched_at"`
	Host        string   `json:"host"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Reference   []string `json:"reference,omitempty"`
	CVE         string   `json:"cve,omitempty"`
	CWE         string   `json:"cwe,omitempty"`
}

func (s *Scanner) authorize(ctx context.Context, taskID, name, target string) error {
	if s == nil || s.gate == nil {
		return fmt.Errorf("scanner not configured")
	}
	return s.gate.Authorize(ctx, lab.Action{AgentID: s.agentID, TaskID: taskID, Name: name, Target: target})
}

// HTTPX probes each target (authorized first) and returns the live endpoints.
func (s *Scanner) HTTPX(ctx context.Context, taskID string, targets []string) ([]Probe, error) {
	if len(targets) == 0 {
		return nil, nil
	}
	for _, t := range targets {
		if err := s.authorize(ctx, taskID, "red.httpx", t); err != nil {
			return nil, fmt.Errorf("authorization denied for %q: %w", t, err)
		}
	}
	bin, err := exec.LookPath(s.HTTPXBin)
	if err != nil {
		return nil, fmt.Errorf("httpx not found on PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, bin, "-silent", "-json", "-status-code", "-title", "-tech-detect", "-no-color", "-disable-update-check", "-no-fallback-scheme", "-rate-limit", "2", "-threads", "1", "-timeout", "5")
	cmd.Stdin = strings.NewReader(strings.Join(targets, "\n"))
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("httpx run: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return parseHTTPX(out.Bytes())
}

func parseHTTPX(data []byte) ([]Probe, error) {
	var probes []Probe
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var raw struct {
			URL         string   `json:"url"`
			Scheme      string   `json:"scheme"`
			Title       string   `json:"title"`
			Webserver   string   `json:"webserver"`
			ContentType string   `json:"content_type"`
			Host        string   `json:"host"`
			StatusCode  int      `json:"status_code"`
			Tech        []string `json:"tech"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		probes = append(probes, Probe{
			URL: raw.URL, Scheme: raw.Scheme, Title: raw.Title, Webserver: raw.Webserver,
			ContentType: raw.ContentType, Host: raw.Host, Status: raw.StatusCode, Tech: raw.Tech,
		})
	}
	return probes, sc.Err()
}

// NucleiOptions scopes a nuclei run. All fields are optional; empty means the
// default template set. Scoping by tag or template id makes a focused audit
// much faster than the full default set.
type NucleiOptions struct {
	Severities   []string // e.g. critical, high
	Tags         []string // e.g. exposure, misconfig, cve
	TemplateIDs  []string // specific template ids
	TemplatesDir string   // path to a nuclei-templates checkout (overrides auto-update)
}

// Nuclei runs detection templates against one authorized target and returns the
// vulnerabilities/exposures found, ready to report to the audited organization.
func (s *Scanner) Nuclei(ctx context.Context, taskID, target string, opt NucleiOptions) ([]Finding, error) {
	return s.NucleiTargets(ctx, taskID, []string{target}, opt)
}

// NucleiTargets runs nuclei against several authorized targets at once. Every
// target is authorized before the scan starts; one out-of-scope target aborts
// the whole run.
func (s *Scanner) NucleiTargets(ctx context.Context, taskID string, targets []string, opt NucleiOptions) ([]Finding, error) {
	if len(targets) == 0 {
		return nil, nil
	}
	for _, t := range targets {
		if err := s.authorize(ctx, taskID, "red.nuclei", t); err != nil {
			return nil, fmt.Errorf("authorization denied for %q: %w", t, err)
		}
	}
	bin, err := exec.LookPath(s.NucleiBin)
	if err != nil {
		return nil, fmt.Errorf("nuclei not found on PATH: %w", err)
	}
	listFile, err := os.CreateTemp("", "nexus-targets-*.txt")
	if err != nil {
		return nil, err
	}
	defer os.Remove(listFile.Name())
	if _, err := listFile.WriteString(strings.Join(targets, "\n") + "\n"); err != nil {
		listFile.Close()
		return nil, err
	}
	listFile.Close()
	args := []string{"-jsonl", "-silent", "-no-color", "-disable-update-check", "-no-interactsh", "-rate-limit", "2", "-concurrency", "1", "-timeout", "5", "-l", listFile.Name()}
	if len(opt.Severities) > 0 {
		args = append(args, "-severity", strings.Join(opt.Severities, ","))
	}
	if len(opt.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opt.Tags, ","))
	}
	if len(opt.TemplateIDs) > 0 {
		args = append(args, "-id", strings.Join(opt.TemplateIDs, ","))
	}
	if opt.TemplatesDir != "" {
		args = append(args, "-t", opt.TemplatesDir)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("nuclei run: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return parseNuclei(out.Bytes())
}

func parseNuclei(data []byte) ([]Finding, error) {
	var findings []Finding
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var raw struct {
			TemplateID string `json:"template-id"`
			MatchedAt  string `json:"matched-at"`
			Host       string `json:"host"`
			Info       struct {
				Name           string   `json:"name"`
				Severity       string   `json:"severity"`
				Tags           []string `json:"tags"`
				Reference      []string `json:"reference"`
				Description    string   `json:"description"`
				Classification struct {
					CVE []string `json:"cve-id"`
					CWE []string `json:"cwe-id"`
				} `json:"classification"`
			} `json:"info"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		f := Finding{
			TemplateID: raw.TemplateID, Name: raw.Info.Name, Severity: raw.Info.Severity,
			MatchedAt: raw.MatchedAt, Host: raw.Host, Description: raw.Info.Description,
			Tags: raw.Info.Tags, Reference: raw.Info.Reference,
		}
		if len(raw.Info.Classification.CVE) > 0 {
			f.CVE = raw.Info.Classification.CVE[0]
		}
		if len(raw.Info.Classification.CWE) > 0 {
			f.CWE = raw.Info.Classification.CWE[0]
		}
		findings = append(findings, f)
	}
	return findings, sc.Err()
}

// Crawl uses katana to discover URLs under an authorized target, expanding the
// attack surface before scanning. depth <= 0 defaults to 2. Returned URLs are
// deduplicated and ready to feed into NucleiTargets.
func (s *Scanner) Crawl(ctx context.Context, taskID, target string, depth int) ([]string, error) {
	if err := s.authorize(ctx, taskID, "red.katana", target); err != nil {
		return nil, fmt.Errorf("authorization denied for %q: %w", target, err)
	}
	if depth <= 0 {
		depth = 2
	}
	bin, err := exec.LookPath(s.CrawlBin)
	if err != nil {
		return nil, fmt.Errorf("katana not found on PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, bin, "-jsonl", "-silent", "-no-color", "-d", fmt.Sprintf("%d", depth), "-u", target)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("katana run: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	return parseKatana(out.Bytes()), nil
}

func parseKatana(data []byte) []string {
	seen := map[string]bool{}
	var urls []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var raw struct {
			Endpoint string `json:"endpoint"`
			URL      string `json:"url"`
			Request  struct {
				Endpoint string `json:"endpoint"`
				URL      string `json:"url"`
			} `json:"request"`
		}
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		u := firstNonEmpty(raw.Request.Endpoint, raw.Request.URL, raw.Endpoint, raw.URL)
		if u != "" && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	return urls
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Secret is a credential leaked in source code, found by gitleaks. The raw
// secret value is NEVER stored — only a short redacted preview — so the report
// can be handed to the audited organization without re-leaking it.
type Secret struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	File        string `json:"file"`
	StartLine   int    `json:"start_line"`
	Preview     string `json:"preview"`
}

// Secrets scans an authorized local source directory for leaked credentials
// (API keys, tokens, private keys) with gitleaks. The directory must be inside
// the engagement's authorized sources, or the scan is refused before running.
func (s *Scanner) Secrets(ctx context.Context, taskID, dir string) ([]Secret, error) {
	if err := s.authorize(ctx, taskID, "red.gitleaks", dir); err != nil {
		return nil, fmt.Errorf("authorization denied for %q: %w", dir, err)
	}
	bin, err := exec.LookPath(s.GitleaksBin)
	if err != nil {
		return nil, fmt.Errorf("gitleaks not found on PATH: %w", err)
	}
	report, err := os.CreateTemp("", "nexus-gitleaks-*.json")
	if err != nil {
		return nil, err
	}
	reportPath := report.Name()
	report.Close()
	defer os.Remove(reportPath)

	// --exit-code 0 makes gitleaks return 0 even when leaks are found, so a
	// clean exit means "ran ok" and findings come from the JSON report.
	cmd := exec.CommandContext(ctx, bin, "dir", dir,
		"--report-format", "json", "--report-path", reportPath,
		"--exit-code", "0", "--no-banner")
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gitleaks run: %w: %s", err, strings.TrimSpace(errb.String()))
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("read gitleaks report: %w", err)
	}
	return parseGitleaks(data)
}

func parseGitleaks(data []byte) ([]Secret, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	var raw []struct {
		RuleID      string `json:"RuleID"`
		Description string `json:"Description"`
		File        string `json:"File"`
		StartLine   int    `json:"StartLine"`
		Secret      string `json:"Secret"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gitleaks report: %w", err)
	}
	out := make([]Secret, 0, len(raw))
	for _, r := range raw {
		out = append(out, Secret{
			RuleID: r.RuleID, Description: r.Description,
			File: r.File, StartLine: r.StartLine, Preview: redact(r.Secret),
		})
	}
	return out, nil
}

// redact keeps only a short fingerprint of a secret so a finding can point to
// it without disclosing the value.
func redact(s string) string {
	s = strings.TrimSpace(s)
	n := len(s)
	if n == 0 {
		return ""
	}
	if n <= 4 {
		return fmt.Sprintf("%s (%d chars)", strings.Repeat("*", n), n)
	}
	return fmt.Sprintf("%s… (%d chars)", s[:4], n)
}

// DepVuln is a vulnerable third-party dependency found by osv-scanner (SCA),
// ready to report to the audited organization for upgrade/patching.
type DepVuln struct {
	Package   string   `json:"package"`
	Ecosystem string   `json:"ecosystem"`
	Version   string   `json:"version"`
	Source    string   `json:"source"`
	ID        string   `json:"id"`
	Aliases   []string `json:"aliases,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Severity  string   `json:"severity,omitempty"`
}

// SCA scans an authorized local source directory for vulnerable dependencies
// (known CVEs in lockfiles / manifests) with osv-scanner. The directory must be
// inside the engagement's authorized sources, or the scan is refused.
func (s *Scanner) SCA(ctx context.Context, taskID, dir string) ([]DepVuln, error) {
	if err := s.authorize(ctx, taskID, "red.osv", dir); err != nil {
		return nil, fmt.Errorf("authorization denied for %q: %w", dir, err)
	}
	bin, err := exec.LookPath(s.OSVBin)
	if err != nil {
		return nil, fmt.Errorf("osv-scanner not found on PATH: %w", err)
	}
	// osv-scanner exits non-zero when vulnerabilities are found, so a run error
	// with JSON on stdout is expected; only treat it as fatal if stdout is empty.
	cmd := exec.CommandContext(ctx, bin, "scan", "source", "--format", "json", "-r", dir)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	runErr := cmd.Run()
	if runErr != nil && len(bytes.TrimSpace(out.Bytes())) == 0 {
		return nil, fmt.Errorf("osv-scanner run: %w: %s", runErr, strings.TrimSpace(errb.String()))
	}
	return parseOSV(out.Bytes())
}

func parseOSV(data []byte) ([]DepVuln, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	var raw struct {
		Results []struct {
			Source struct {
				Path string `json:"path"`
			} `json:"source"`
			Packages []struct {
				Package struct {
					Name      string `json:"name"`
					Version   string `json:"version"`
					Ecosystem string `json:"ecosystem"`
				} `json:"package"`
				Vulnerabilities []struct {
					ID      string   `json:"id"`
					Aliases []string `json:"aliases"`
					Summary string   `json:"summary"`
				} `json:"vulnerabilities"`
				Groups []struct {
					IDs         []string `json:"ids"`
					Aliases     []string `json:"aliases"`
					MaxSeverity string   `json:"max_severity"`
				} `json:"groups"`
			} `json:"packages"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse osv-scanner report: %w", err)
	}
	var out []DepVuln
	emitted := map[string]bool{} // package|version|id → already reported
	for _, res := range raw.Results {
		for _, p := range res.Packages {
			// osv-scanner lists each advisory once per id alias (e.g. the same
			// flaw under its PYSEC and its GHSA id). Its "groups" already collapse
			// those aliases into one real vulnerability, so we emit one DepVuln
			// per group — not per alias — to avoid duplicate findings in reports.
			summaries := map[string]string{}
			for _, v := range p.Vulnerabilities {
				if v.Summary != "" {
					summaries[v.ID] = v.Summary
				}
			}
			for _, g := range p.Groups {
				if len(g.IDs) == 0 {
					continue
				}
				id := g.IDs[0]
				key := p.Package.Name + "|" + p.Package.Version + "|" + id
				if emitted[key] {
					continue
				}
				emitted[key] = true
				// Aliases = all ids/aliases in the group except the representative.
				var aliases []string
				seen := map[string]bool{id: true}
				add := func(xs []string) {
					for _, x := range xs {
						if x != "" && !seen[x] {
							seen[x] = true
							aliases = append(aliases, x)
						}
					}
				}
				add(g.IDs)
				add(g.Aliases)
				// Prefer a human summary from any id in the group.
				summary := ""
				for _, gid := range g.IDs {
					if sm, ok := summaries[gid]; ok {
						summary = sm
						break
					}
				}
				out = append(out, DepVuln{
					Package: p.Package.Name, Ecosystem: p.Package.Ecosystem,
					Version: p.Package.Version, Source: res.Source.Path,
					ID: id, Aliases: aliases, Summary: summary, Severity: g.MaxSeverity,
				})
			}
		}
	}
	return out, nil
}
