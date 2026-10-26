// Package sigma implements a pragmatic subset of the Sigma detection-rule
// standard (https://github.com/SigmaHQ/sigma) for NEXUS blue-team detection.
//
// The Sigma SPECIFICATION and format are public domain. Community rule CONTENT
// in the SigmaHQ repository is licensed under the Detection Rule License (DRL)
// 1.1 — if you vendor community rules into this repository, keep that license
// file alongside them. This package contains only original NEXUS code; it loads
// rules from a directory you supply, so no third-party rule content is embedded.
//
// Supported: single-document rules, field maps/lists, wildcards, null, exists,
// contains/startswith/endswith/re/all/cased, regex flags i/m/s, and boolean/count
// conditions. Unsupported features (including timeframe and correlation) fail
// validation explicitly rather than approximating their meaning.
package sigma

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LogSource identifies the telemetry a rule applies to.
type LogSource struct {
	Product  string `yaml:"product" json:"product,omitempty"`
	Service  string `yaml:"service" json:"service,omitempty"`
	Category string `yaml:"category" json:"category,omitempty"`
}

// Rule is a parsed Sigma detection rule.
type Rule struct {
	Title       string         `yaml:"title"`
	ID          string         `yaml:"id"`
	Status      string         `yaml:"status"`
	Description string         `yaml:"description"`
	Author      string         `yaml:"author"`
	License     string         `yaml:"license"`
	References  []string       `yaml:"references"`
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
	if len(data) > MaxRuleBytes {
		return Rule{}, fmt.Errorf("sigma rule size limit exceeded")
	}
	var r Rule
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&r); err != nil {
		return Rule{}, err
	}
	if r.Title == "" || r.Detection == nil {
		return Rule{}, fmt.Errorf("invalid sigma rule: missing title or detection")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Rule{}, fmt.Errorf("sigma requires exactly one YAML document")
	}
	if _, err := r.Compile(); err != nil {
		return Rule{}, err
	}
	return r, nil
}

// LoadDir loads every *.yml / *.yaml rule under dir (recursive). Rules that fail
// to parse are skipped; their errors are returned alongside the valid rules so a
// handful of malformed community rules never abort the whole load. Traversal
// errors are reported too, distinguishing an inaccessible tree from an empty one.
func LoadDir(dir string) ([]Rule, []error) {
	var rules []Rule
	var errs []error
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			return nil
		}
		if d.IsDir() {
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
