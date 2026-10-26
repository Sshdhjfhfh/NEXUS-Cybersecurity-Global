// Package sigma implements a pragmatic subset of the Sigma detection-rule
// standard (https://github.com/SigmaHQ/sigma) for NEXUS blue-team detection.
//
// The Sigma SPECIFICATION and format are public domain. Community rule CONTENT
// in the SigmaHQ repository is licensed under the Detection Rule License (DRL)
// 1.1 — if you vendor community rules into this repository, keep that license
// file alongside them. This package contains only original NEXUS code; it loads
// rules from a directory you supply, so no third-party rule content is embedded.
//
// Supported today: single-document rules with named selections (field maps and
// lists), the modifiers contains/startswith/endswith/re/all, list values (OR),
// and conditions using and/or/not/parentheses plus "all of"/"N of" them/prefix*.
// Advanced features (correlation, timeframe, numeric "N of" counting, near) are
// intentionally out of scope for this first increment.
package sigma

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LogSource identifies the telemetry a rule applies to.
type LogSource struct {
	Product  string `yaml:"product"`
	Service  string `yaml:"service"`
	Category string `yaml:"category"`
}

// Rule is a parsed Sigma detection rule.
type Rule struct {
	Title       string         `yaml:"title"`
	ID          string         `yaml:"id"`
	Status      string         `yaml:"status"`
	Description string         `yaml:"description"`
	Author      string         `yaml:"author"`
	Level       string         `yaml:"level"`
	Tags        []string       `yaml:"tags"`
	LogSource   LogSource      `yaml:"logsource"`
	Detection   map[string]any `yaml:"detection"`
}

// Condition returns the rule's detection condition expression.
func (r Rule) Condition() (string, error) {
	c, ok := r.Detection["condition"]
	if !ok {
		return "", fmt.Errorf("rule %q: detection has no condition", r.Title)
	}
	s, ok := c.(string)
	if !ok {
		return "", fmt.Errorf("rule %q: condition is not a single string (unsupported)", r.Title)
	}
	return s, nil
}

// Parse decodes a single Sigma rule from YAML.
func Parse(data []byte) (Rule, error) {
	var r Rule
	if err := yaml.Unmarshal(data, &r); err != nil {
		return Rule{}, err
	}
	if r.Title == "" || r.Detection == nil {
		return Rule{}, fmt.Errorf("invalid sigma rule: missing title or detection")
	}
	if _, err := r.Condition(); err != nil {
		return Rule{}, err
	}
	return r, nil
}

// LoadDir loads every *.yml / *.yaml rule under dir (recursive). Rules that fail
// to parse are skipped; their errors are returned alongside the valid rules so a
// handful of malformed community rules never abort the whole load.
func LoadDir(dir string) ([]Rule, []error) {
	var rules []Rule
	var errs []error
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yml", ".yaml":
		default:
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, readErr))
			return nil
		}
		r, parseErr := Parse(b)
		if parseErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, parseErr))
			return nil
		}
		rules = append(rules, r)
		return nil
	})
	return rules, errs
}
