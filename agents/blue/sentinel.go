// Package blue implements passive SENTINEL analysis of supplied lab evidence.
// It does not probe targets or apply remediation automatically.
package blue

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/lab"
)

type Observation struct {
	ID           string              `json:"id"`
	Target       string              `json:"target"`
	Scheme       string              `json:"scheme"`
	Status       int                 `json:"status"`
	Headers      map[string][]string `json:"headers"`
	FailedLogins int                 `json:"failed_logins"`
	ObservedAt   time.Time           `json:"observed_at"`
}
type Finding struct {
	Rule           string `json:"rule"`
	Severity       string `json:"severity"`
	CWE            string `json:"cwe"`
	EvidenceID     string `json:"evidence_id"`
	Target         string `json:"target"`
	Summary        string `json:"summary"`
	Recommendation string `json:"recommendation"`
}
type Sentinel struct {
	id   string
	gate lab.Authorizer
}

func New(id string, gate lab.Authorizer) (*Sentinel, error) {
	if strings.TrimSpace(id) == "" || gate == nil {
		return nil, fmt.Errorf("agent identity and laboratory authorizer are required")
	}
	return &Sentinel{id: id, gate: gate}, nil
}
func (s *Sentinel) ID() string { return s.id }

// Analyze is deterministic and returns findings linked to the supplied evidence.
// Observations must represent collected responses, not speculative LLM output.
func (s *Sentinel) Analyze(ctx context.Context, taskID string, observations []Observation) ([]Finding, error) {
	if s == nil || s.gate == nil {
		return nil, fmt.Errorf("SENTINEL is not configured")
	}
	if len(observations) > 10000 {
		return nil, fmt.Errorf("observation limit exceeded")
	}
	findings := []Finding{}
	seen := map[string]bool{}
	for _, o := range observations {
		if o.ID == "" || o.ObservedAt.IsZero() || o.Status < 100 || o.Status > 599 || o.FailedLogins < 0 || (o.Scheme != "http" && o.Scheme != "https") {
			return nil, fmt.Errorf("invalid observation %q", o.ID)
		}
		identity := o.Target + "\x00" + o.ID
		if seen[identity] {
			return nil, fmt.Errorf("duplicate observation %q", o.ID)
		}
		seen[identity] = true
		if err := s.gate.Authorize(ctx, lab.Action{AgentID: s.id, TaskID: taskID, Name: "blue.analyze", Target: o.Target}); err != nil {
			return nil, err
		}
		headers := http.Header{}
		for name, values := range o.Headers {
			headers[http.CanonicalHeaderKey(name)] = append(headers[http.CanonicalHeaderKey(name)], values...)
		}
		add := func(rule, severity, cwe, summary, recommendation string) {
			findings = append(findings, Finding{rule, severity, cwe, o.ID, o.Target, summary, recommendation})
		}
		if o.Scheme == "http" {
			add("cleartext", "medium", "CWE-319", "Response used unencrypted HTTP", "Use TLS within the laboratory")
		}
		if o.Scheme == "https" && headers.Get("Strict-Transport-Security") == "" {
			add("missing-hsts", "low", "CWE-319", "HTTPS response lacks HSTS", "Configure an appropriate HSTS policy")
		}
		if strings.Contains(strings.ToLower(headers.Get("Content-Type")), "text/html") && headers.Get("Content-Security-Policy") == "" {
			add("missing-csp", "low", "CWE-693", "HTML response lacks a content security policy", "Define a restrictive content security policy")
		}
		if headers.Get("X-Content-Type-Options") != "nosniff" {
			add("missing-nosniff", "low", "CWE-693", "Response lacks MIME sniffing protection", "Set X-Content-Type-Options to nosniff")
		}
		response := http.Response{Header: headers}
		for _, cookie := range response.Cookies() {
			if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite == http.SameSiteDefaultMode || cookie.SameSite == 0 {
				add("cookie-flags", "medium", "CWE-614", "Cookie "+cookie.Name+" lacks recommended protection", "Review Secure, HttpOnly and SameSite attributes")
			}
		}
		if o.FailedLogins >= 5 {
			add("login-failures", "medium", "CWE-307", "Repeated failed logins were observed", "Review authentication telemetry and rate limits")
		}
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Target != findings[j].Target {
			return findings[i].Target < findings[j].Target
		}
		if findings[i].EvidenceID != findings[j].EvidenceID {
			return findings[i].EvidenceID < findings[j].EvidenceID
		}
		return findings[i].Rule < findings[j].Rule
	})
	return findings, nil
}
