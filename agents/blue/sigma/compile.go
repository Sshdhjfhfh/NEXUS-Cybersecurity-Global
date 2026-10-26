// Original NEXUS code implementing a bounded subset of the public Sigma format.
// No third-party detection-rule content is bundled.
package sigma

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	MaxRuleBytes  = 1 << 20
	maxSelections = 256
	maxDepth      = 32
	maxValues     = 256
)

type eventTest func(Event) bool
type matchBits [maxSelections / 64]uint64

func (v matchBits) has(index int) bool { return v[index/64]&(uint64(1)<<uint(index%64)) != 0 }

type condition func(matchBits) bool

// CompiledRule owns all predicates. It retains no mutable Rule maps and can be
// reused concurrently. The caller must route telemetry by logsource first.
type CompiledRule struct {
	selections []eventTest
	condition  condition
}

func (r *CompiledRule) Match(ev Event) bool {
	if r == nil || r.condition == nil {
		return false
	}
	var matches matchBits
	for i, selection := range r.selections {
		if selection(ev) {
			matches[i/64] |= uint64(1) << uint(i%64)
		}
	}
	return r.condition(matches)
}

var selectionName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Compile rejects unsupported semantics before scoring any events.
func (r Rule) Compile() (*CompiledRule, error) {
	if r.Title == "" || len(r.Detection) < 2 || len(r.Detection) > maxSelections+1 {
		return nil, fmt.Errorf("sigma: title and 1..%d selections required", maxSelections)
	}
	expression, err := r.Condition()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(r.Detection)-1)
	for name := range r.Detection {
		if name == "condition" {
			continue
		}
		if !selectionName.MatchString(name) || name == "timeframe" || name == "and" || name == "or" || name == "not" || name == "of" {
			return nil, fmt.Errorf("sigma: unsupported selection %q", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	c := &CompiledRule{}
	indices := map[string]int{}
	for _, name := range names {
		predicate, err := compileSelection(r.Detection[name], 0)
		if err != nil {
			return nil, fmt.Errorf("sigma rule %q selection %q: %w", r.Title, name, err)
		}
		indices[name] = len(c.selections)
		c.selections = append(c.selections, predicate)
	}
	c.condition, err = compileCondition(expression, indices, names)
	if err != nil {
		return nil, fmt.Errorf("sigma rule %q: %w", r.Title, err)
	}
	return c, nil
}

func compileSelection(spec any, depth int) (eventTest, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("selection nesting limit exceeded")
	}
	switch v := spec.(type) {
	case map[string]any:
		if len(v) == 0 || len(v) > maxValues {
			return nil, fmt.Errorf("invalid field count")
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		predicates := make([]eventTest, 0, len(keys))
		for _, k := range keys {
			p, err := compileField(k, v[k])
			if err != nil {
				return nil, err
			}
			predicates = append(predicates, p)
		}
		return func(e Event) bool {
			for _, p := range predicates {
				if !p(e) {
					return false
				}
			}
			return true
		}, nil
	case []any:
		if len(v) == 0 || len(v) > maxValues {
			return nil, fmt.Errorf("invalid selection list size")
		}
		predicates := make([]eventTest, 0, len(v))
		for _, item := range v {
			p, err := compileSelection(item, depth+1)
			if err != nil {
				return nil, err
			}
			predicates = append(predicates, p)
		}
		return func(e Event) bool {
			for _, p := range predicates {
				if p(e) {
					return true
				}
			}
			return false
		}, nil
	case string:
		p, err := compilePattern(v, "contains", false, "")
		if err != nil {
			return nil, err
		}
		return func(e Event) bool {
			for _, item := range e {
				if matchEventValue(item, p) {
					return true
				}
			}
			return false
		}, nil
	default:
		return nil, fmt.Errorf("unsupported selection type %T", spec)
	}
}

func scalar(v any) (string, bool) {
	switch n := v.(type) {
	case string:
		return n, true
	case bool, int, int64, uint64:
		return fmt.Sprint(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return strconv.FormatInt(i, 10), true
		}
		if !strings.ContainsAny(string(n), ".eE") {
			u, err := strconv.ParseUint(string(n), 10, 64)
			return strconv.FormatUint(u, 10), err == nil
		}
		f, err := n.Float64()
		return strconv.FormatFloat(f, 'g', -1, 64), err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
	case float64:
		return strconv.FormatFloat(n, 'g', -1, 64), !math.IsNaN(n) && !math.IsInf(n, 0)
	default:
		return "", false
	}
}

func matchEventValue(v any, p func(string) bool) bool {
	switch n := v.(type) {
	case []any:
		for _, item := range n {
			if s, ok := scalar(item); ok && p(s) {
				return true
			}
		}
	case []string:
		for _, s := range n {
			if p(s) {
				return true
			}
		}
	default:
		if s, ok := scalar(v); ok {
			return p(s)
		}
	}
	return false
}

func compileField(key string, v any) (eventTest, error) {
	parts := strings.Split(key, "|")
	field := parts[0]
	if field == "" || len(parts) > 8 {
		return nil, fmt.Errorf("invalid field %q", key)
	}
	mode, flags, all, cased := "equals", "", false, false
	seen := map[string]bool{}
	for _, m := range parts[1:] {
		if seen[m] {
			return nil, fmt.Errorf("duplicate modifier %q", m)
		}
		seen[m] = true
		switch m {
		case "contains", "startswith", "endswith", "re", "exists":
			if mode != "equals" {
				return nil, fmt.Errorf("conflicting modifiers in %q", key)
			}
			mode = m
		case "all":
			all = true
		case "cased":
			cased = true
		case "i", "m", "s":
			if mode != "re" {
				return nil, fmt.Errorf("regex flag without re in %q", key)
			}
			flags += m
		default:
			return nil, fmt.Errorf("unsupported modifier %q", m)
		}
	}
	if mode == "exists" {
		want, ok := v.(bool)
		if !ok || len(parts) != 2 {
			return nil, fmt.Errorf("exists requires a boolean and no other modifiers")
		}
		return func(e Event) bool { _, ok := e[field]; return ok == want }, nil
	}
	if v == nil {
		if len(parts) != 1 {
			return nil, fmt.Errorf("null does not support modifiers")
		}
		return func(e Event) bool { v, ok := e[field]; return !ok || v == nil }, nil
	}
	values, isList := v.([]any)
	if all && (!isList || len(values) < 2) {
		return nil, fmt.Errorf("all requires a list of at least two values")
	}
	if !isList {
		values = []any{v}
	}
	if len(values) == 0 || len(values) > maxValues {
		return nil, fmt.Errorf("invalid value count for %q", field)
	}
	predicates := make([]func(string) bool, 0, len(values))
	for _, item := range values {
		text, ok := scalar(item)
		if !ok {
			return nil, fmt.Errorf("unsupported field value %T", item)
		}
		p, err := compilePattern(text, mode, cased, flags)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", field, err)
		}
		predicates = append(predicates, p)
	}
	return func(e Event) bool {
		v, exists := e[field]
		if !exists || v == nil {
			return false
		}
		for _, p := range predicates {
			matched := matchEventValue(v, p)
			if all && !matched {
				return false
			}
			if !all && matched {
				return true
			}
		}
		return all
	}, nil
}

func compilePattern(pattern, mode string, cased bool, flags string) (func(string) bool, error) {
	if len(pattern) > 8192 {
		return nil, fmt.Errorf("pattern size limit exceeded")
	}
	if mode == "re" {
		if cased {
			return nil, fmt.Errorf("cased is not a regex modifier")
		}
		if flags != "" {
			pattern = "(?" + flags + ")" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("unsupported or invalid regular expression: %w", err)
		}
		return re.MatchString, nil
	}
	if !strings.ContainsAny(pattern, `*?\`) {
		if !cased {
			pattern = strings.ToLower(pattern)
		}
		return func(v string) bool {
			if !cased {
				v = strings.ToLower(v)
			}
			switch mode {
			case "contains":
				return strings.Contains(v, pattern)
			case "startswith":
				return strings.HasPrefix(v, pattern)
			case "endswith":
				return strings.HasSuffix(v, pattern)
			default:
				return v == pattern
			}
		}, nil
	}
	var expression strings.Builder
	if cased {
		expression.WriteString("(?s)")
	} else {
		expression.WriteString("(?is)")
	}
	if mode != "contains" && mode != "endswith" {
		expression.WriteString("^")
	}
	chars := []rune(pattern)
	for i := 0; i < len(chars); i++ {
		switch chars[i] {
		case '\\':
			if i+1 < len(chars) && strings.ContainsRune(`\*?`, chars[i+1]) {
				i++
				expression.WriteString(regexp.QuoteMeta(string(chars[i])))
			} else {
				expression.WriteString(`\\`)
			}
		case '*':
			expression.WriteString(".*")
		case '?':
			expression.WriteString(".")
		default:
			expression.WriteString(regexp.QuoteMeta(string(chars[i])))
		}
	}
	if mode != "contains" && mode != "startswith" {
		expression.WriteString("$")
	}
	re, err := regexp.Compile(expression.String())
	if err != nil {
		return nil, err
	}
	return re.MatchString, nil
}
