// Package engagement models an authorized testing engagement: the explicit,
// signed-off scope of targets a NEXUS operator may scan. It is the TECHNICAL
// enforcement of the rule "always require verifiable scope and authorization":
// the scanner refuses any target not listed in the engagement scope.
package engagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/nexus-ia/nexus/lab"
)

// sourceScanActions are the action names that operate on a local source
// directory (secrets / SCA) rather than a network target. For these, the
// authorizer checks the path against AuthorizedSources instead of the host
// scope. Any other action falls back to the network host check (default-deny
// for paths), so a filesystem path can never slip through as if it were a host.
var sourceScanActions = map[string]bool{
	"red.gitleaks": true,
	"red.osv":      true,
}

// Scope is the authorized perimeter of one engagement, loaded from a file the
// client/operator signs off. Targets are hostnames or IPs; AuthorizedSources
// are local directories the operator may scan for secrets / vulnerable
// dependencies.
type Scope struct {
	Engagement        string   `json:"engagement"`
	Client            string   `json:"client"`
	AuthorizedTargets []string `json:"authorized_targets"`
	AuthorizedSources []string `json:"authorized_sources,omitempty"`
}

// LoadScope reads and validates an engagement scope file.
func LoadScope(path string) (Scope, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Scope{}, err
	}
	var s Scope
	if err := json.Unmarshal(b, &s); err != nil {
		return Scope{}, fmt.Errorf("invalid scope file: %w", err)
	}
	if strings.TrimSpace(s.Engagement) == "" {
		return Scope{}, fmt.Errorf("scope must name an engagement")
	}
	if len(s.AuthorizedTargets) == 0 && len(s.AuthorizedSources) == 0 {
		return Scope{}, fmt.Errorf("scope must list at least one authorized target or source")
	}
	return s, nil
}

// Authorizer implements lab.Authorizer, allowing only network targets whose
// host is in scope and only source directories under AuthorizedSources.
type Authorizer struct {
	allowed    map[string]bool
	allowedSrc []string // cleaned absolute path prefixes
}

// NewAuthorizer builds an authorizer from a scope.
func NewAuthorizer(s Scope) (*Authorizer, error) {
	if len(s.AuthorizedTargets) == 0 && len(s.AuthorizedSources) == 0 {
		return nil, fmt.Errorf("engagement scope has no authorized targets or sources")
	}
	a := &Authorizer{allowed: make(map[string]bool, len(s.AuthorizedTargets))}
	for _, t := range s.AuthorizedTargets {
		h := host(t)
		if h == "" {
			return nil, fmt.Errorf("invalid authorized target %q", t)
		}
		a.allowed[strings.ToLower(h)] = true
	}
	for _, p := range s.AuthorizedSources {
		abs, err := filepath.Abs(filepath.Clean(strings.TrimSpace(p)))
		if err != nil || abs == "" {
			return nil, fmt.Errorf("invalid authorized source %q", p)
		}
		a.allowedSrc = append(a.allowedSrc, abs)
	}
	return a, nil
}

// Authorize permits network actions only if the target host is in scope, and
// source-scan actions only if the path is under an authorized source.
func (a *Authorizer) Authorize(_ context.Context, act lab.Action) error {
	if sourceScanActions[act.Name] {
		return a.authorizeSource(act.Target)
	}
	h := host(act.Target)
	if h == "" {
		return fmt.Errorf("action target %q has no host", act.Target)
	}
	if !a.allowed[strings.ToLower(h)] {
		return fmt.Errorf("target %q is OUTSIDE the authorized engagement scope", act.Target)
	}
	return nil
}

// authorizeSource permits a source-scan action only if its path resolves to a
// directory at or under one of the authorized source roots.
func (a *Authorizer) authorizeSource(target string) error {
	p, err := filepath.Abs(filepath.Clean(strings.TrimSpace(target)))
	if err != nil || strings.TrimSpace(target) == "" {
		return fmt.Errorf("invalid source path %q", target)
	}
	for _, root := range a.allowedSrc {
		if p == root || strings.HasPrefix(p, root+string(filepath.Separator)) {
			return nil
		}
	}
	return fmt.Errorf("source path %q is OUTSIDE the authorized engagement scope", target)
}

// host extracts a comparable hostname from a URL or a bare host[:port].
func host(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if strings.Contains(target, "://") {
		if u, err := url.Parse(target); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	if h, _, err := net.SplitHostPort(target); err == nil {
		return h
	}
	return target
}
