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
	"os/exec"
	"strings"

	"github.com/nexus-ia/nexus/lab"
)

// Scanner runs authorized external scanners. Bin fields default to the tool name
// resolved on PATH; override them to pin a lab toolchain location.
type Scanner struct {
	HTTPXBin  string
	NucleiBin string
	gate      lab.Authorizer
	agentID   string
}

// New returns a Scanner bound to a laboratory authorizer and agent identity.
func New(gate lab.Authorizer, agentID string) (*Scanner, error) {
	if gate == nil || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("scanner requires a laboratory authorizer and agent id")
	}
	return &Scanner{HTTPXBin: "httpx", NucleiBin: "nuclei", gate: gate, agentID: agentID}, nil
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
	cmd := exec.CommandContext(ctx, bin, "-silent", "-json", "-status-code", "-title", "-tech-detect", "-no-color")
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

// Nuclei runs detection templates against one authorized target and returns the
// vulnerabilities/exposures found. severities (optional) filters by level,
// e.g. []string{"critical","high"}.
func (s *Scanner) Nuclei(ctx context.Context, taskID, target string, severities []string) ([]Finding, error) {
	if err := s.authorize(ctx, taskID, "red.nuclei", target); err != nil {
		return nil, fmt.Errorf("authorization denied for %q: %w", target, err)
	}
	bin, err := exec.LookPath(s.NucleiBin)
	if err != nil {
		return nil, fmt.Errorf("nuclei not found on PATH: %w", err)
	}
	args := []string{"-jsonl", "-silent", "-no-color", "-disable-update-check", "-u", target}
	if len(severities) > 0 {
		args = append(args, "-severity", strings.Join(severities, ","))
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
