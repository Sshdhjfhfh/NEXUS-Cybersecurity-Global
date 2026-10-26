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
	"strings"

	"github.com/nexus-ia/nexus/lab"
)

// Scope is the authorized perimeter of one engagement, loaded from a file the
// client/operator signs off. Targets are hostnames or IPs.
type Scope struct {
	Engagement        string   `json:"engagement"`
	Client            string   `json:"client"`
	AuthorizedTargets []string `json:"authorized_targets"`
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
	if strings.TrimSpace(s.Engagement) == "" || len(s.AuthorizedTargets) == 0 {
		return Scope{}, fmt.Errorf("scope must name an engagement and at least one authorized target")
	}
	return s, nil
}

// Authorizer implements lab.Authorizer, allowing only targets whose host is in
// the engagement scope.
type Authorizer struct{ allowed map[string]bool }

// NewAuthorizer builds an authorizer from a scope.
func NewAuthorizer(s Scope) (*Authorizer, error) {
	if len(s.AuthorizedTargets) == 0 {
		return nil, fmt.Errorf("engagement scope has no authorized targets")
	}
	a := &Authorizer{allowed: make(map[string]bool, len(s.AuthorizedTargets))}
	for _, t := range s.AuthorizedTargets {
		h := host(t)
		if h == "" {
			return nil, fmt.Errorf("invalid authorized target %q", t)
		}
		a.allowed[strings.ToLower(h)] = true
	}
	return a, nil
}

// Authorize permits the action only if its target host is in scope.
func (a *Authorizer) Authorize(_ context.Context, act lab.Action) error {
	h := host(act.Target)
	if h == "" {
		return fmt.Errorf("action target %q has no host", act.Target)
	}
	if !a.allowed[strings.ToLower(h)] {
		return fmt.Errorf("target %q is OUTSIDE the authorized engagement scope", act.Target)
	}
	return nil
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
