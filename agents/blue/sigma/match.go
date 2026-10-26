package sigma

import (
	"fmt"
	"regexp"
	"strings"
)

// Event is a normalized telemetry record: field name -> value. Values are
// compared as their string form, case-insensitively (pragmatic default).
type Event map[string]any

// Match reports whether the event satisfies the rule's detection and condition.
func (r Rule) Match(ev Event) (bool, error) {
	cond, err := r.Condition()
	if err != nil {
		return false, err
	}
	sel := map[string]bool{}
	names := make([]string, 0, len(r.Detection))
	for name, spec := range r.Detection {
		if name == "condition" {
			continue
		}
		sel[name] = evalSelection(spec, ev)
		names = append(names, name)
	}
	return evalCondition(cond, sel, names)
}

// evalSelection evaluates one selection block against the event.
//   - map   -> every field condition must hold (AND)
//   - list  -> any sub-selection may hold (OR)
//   - string-> keyword: any field value contains it
func evalSelection(spec any, ev Event) bool {
	switch v := spec.(type) {
	case map[string]any:
		if len(v) == 0 {
			return false
		}
		for key, want := range v {
			if !matchField(key, want, ev) {
				return false
			}
		}
		return true
	case []any:
		for _, item := range v {
			if evalSelection(item, ev) {
				return true
			}
		}
		return false
	case string:
		return anyFieldContains(ev, v)
	default:
		return false
	}
}

// matchField applies a "field|modifiers" condition to the event.
func matchField(key string, want any, ev Event) bool {
	parts := strings.Split(key, "|")
	field := parts[0]
	raw, ok := ev[field]
	if !ok {
		return false
	}
	got := strings.ToLower(toStr(raw))

	cmp := "equals"
	requireAll := false
	for _, m := range parts[1:] {
		switch strings.ToLower(m) {
		case "contains", "startswith", "endswith", "re":
			cmp = strings.ToLower(m)
		case "all":
			requireAll = true
		}
	}

	test := func(w string) bool {
		w = strings.ToLower(w)
		switch cmp {
		case "contains":
			return strings.Contains(got, w)
		case "startswith":
			return strings.HasPrefix(got, w)
		case "endswith":
			return strings.HasSuffix(got, w)
		case "re":
			re, err := regexp.Compile(w)
			return err == nil && re.MatchString(got)
		default:
			return got == w
		}
	}

	wants := toStrings(want)
	if requireAll {
		for _, w := range wants {
			if !test(w) {
				return false
			}
		}
		return true
	}
	for _, w := range wants {
		if test(w) {
			return true
		}
	}
	return false
}

func anyFieldContains(ev Event, sub string) bool {
	sub = strings.ToLower(sub)
	for _, v := range ev {
		if strings.Contains(strings.ToLower(toStr(v)), sub) {
			return true
		}
	}
	return false
}

func toStr(v any) string { return fmt.Sprintf("%v", v) }

func toStrings(v any) []string {
	if list, ok := v.([]any); ok {
		out := make([]string, 0, len(list))
		for _, e := range list {
			out = append(out, toStr(e))
		}
		return out
	}
	return []string{toStr(v)}
}

// ---- condition expression ----

var ofExpr = regexp.MustCompile(`(?i)(all|any|\d+)\s+of\s+([a-zA-Z0-9_*]+|them)`)

// expandOf rewrites "all of them", "N of them", "all of pre*", "1 of pre*" into
// parenthesized and/or expressions over the concrete selection names. Numeric
// "N of" is approximated as "at least one" (OR) in this first increment.
func expandOf(cond string, names []string) string {
	return ofExpr.ReplaceAllStringFunc(cond, func(m string) string {
		g := ofExpr.FindStringSubmatch(m)
		quant := strings.ToLower(g[1])
		target := g[2]
		var matched []string
		switch {
		case strings.EqualFold(target, "them"):
			matched = names
		case strings.HasSuffix(target, "*"):
			pre := strings.TrimSuffix(target, "*")
			for _, n := range names {
				if strings.HasPrefix(n, pre) {
					matched = append(matched, n)
				}
			}
		default:
			matched = []string{target}
		}
		if len(matched) == 0 {
			if quant == "all" {
				return "true"
			}
			return "false"
		}
		op := " or "
		if quant == "all" {
			op = " and "
		}
		return "( " + strings.Join(matched, op) + " )"
	})
}

func evalCondition(cond string, sel map[string]bool, names []string) (bool, error) {
	toks := tokenize(expandOf(strings.TrimSpace(cond), names))
	p := &condParser{toks: toks, sel: sel}
	v := p.parseOr()
	if p.err != nil {
		return false, p.err
	}
	if p.pos != len(p.toks) {
		return false, fmt.Errorf("unexpected token %q in condition", p.peek())
	}
	return v, nil
}

func tokenize(s string) []string {
	s = strings.ReplaceAll(s, "(", " ( ")
	s = strings.ReplaceAll(s, ")", " ) ")
	return strings.Fields(s)
}

type condParser struct {
	toks []string
	pos  int
	sel  map[string]bool
	err  error
}

func (p *condParser) peek() string {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return ""
}
func (p *condParser) next() string {
	t := p.peek()
	p.pos++
	return t
}

func (p *condParser) parseOr() bool {
	v := p.parseAnd()
	for strings.EqualFold(p.peek(), "or") {
		p.next()
		v = p.parseAnd() || v
	}
	return v
}
func (p *condParser) parseAnd() bool {
	v := p.parseNot()
	for strings.EqualFold(p.peek(), "and") {
		p.next()
		v = p.parseNot() && v
	}
	return v
}
func (p *condParser) parseNot() bool {
	if strings.EqualFold(p.peek(), "not") {
		p.next()
		return !p.parseNot()
	}
	return p.parseAtom()
}
func (p *condParser) parseAtom() bool {
	t := p.next()
	switch {
	case t == "(":
		v := p.parseOr()
		if p.peek() == ")" {
			p.next()
		} else if p.err == nil {
			p.err = fmt.Errorf("missing closing parenthesis")
		}
		return v
	case t == "true":
		return true
	case t == "false":
		return false
	case t == "":
		p.err = fmt.Errorf("unexpected end of condition")
		return false
	default:
		return p.sel[t]
	}
}
