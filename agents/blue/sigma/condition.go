package sigma

import (
	"fmt"
	"strconv"
	"strings"
)

type conditionCompiler struct {
	tokens  []string
	pos     int
	indices map[string]int
	names   []string
	err     error
}

func compileCondition(text string, indices map[string]int, names []string) (condition, error) {
	if len(text) > 16384 {
		return nil, fmt.Errorf("condition size limit exceeded")
	}
	tokens := strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(text))
	if len(tokens) == 0 || len(tokens) > 1024 {
		return nil, fmt.Errorf("invalid condition token count")
	}
	p := &conditionCompiler{tokens: tokens, indices: indices, names: names}
	result := p.parseOr(0)
	if p.err != nil {
		return nil, p.err
	}
	if p.pos != len(tokens) {
		return nil, fmt.Errorf("unexpected condition token %q", p.peek())
	}
	return result, nil
}

func (p *conditionCompiler) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}
func (p *conditionCompiler) next() string { token := p.peek(); p.pos++; return token }

func (p *conditionCompiler) parseOr(depth int) condition {
	left := p.parseAnd(depth)
	for strings.EqualFold(p.peek(), "or") {
		p.next()
		right := p.parseAnd(depth)
		previous := left
		left = func(v matchBits) bool { return previous(v) || right(v) }
	}
	return left
}

func (p *conditionCompiler) parseAnd(depth int) condition {
	left := p.parseUnary(depth)
	for strings.EqualFold(p.peek(), "and") {
		p.next()
		right := p.parseUnary(depth)
		previous := left
		left = func(v matchBits) bool { return previous(v) && right(v) }
	}
	return left
}

func (p *conditionCompiler) fail(format string, args ...any) condition {
	if p.err == nil {
		p.err = fmt.Errorf(format, args...)
	}
	return func(matchBits) bool { return false }
}

func (p *conditionCompiler) parseUnary(depth int) condition {
	if depth > maxDepth {
		return p.fail("condition nesting limit exceeded")
	}
	if strings.EqualFold(p.peek(), "not") {
		p.next()
		child := p.parseUnary(depth + 1)
		return func(v matchBits) bool { return !child(v) }
	}
	token := p.next()
	if token == "(" {
		child := p.parseOr(depth + 1)
		if p.peek() != ")" {
			return p.fail("missing closing parenthesis")
		}
		p.next()
		return child
	}
	if strings.EqualFold(p.peek(), "of") {
		p.next()
		pattern := p.next()
		needed, err := strconv.Atoi(token)
		quantifier := strings.ToLower(token)
		if quantifier != "all" && quantifier != "any" && (err != nil || needed < 1) {
			return p.fail("invalid selection count %q", token)
		}
		var matched []int
		if strings.EqualFold(pattern, "them") {
			for _, name := range p.names {
				if !strings.HasPrefix(name, "_") {
					matched = append(matched, p.indices[name])
				}
			}
		} else {
			match, err := compilePattern(pattern, "equals", true, "")
			if err != nil || pattern == "" {
				return p.fail("invalid selection pattern")
			}
			for _, name := range p.names {
				if match(name) {
					matched = append(matched, p.indices[name])
				}
			}
		}
		if len(matched) == 0 {
			return p.fail("selection pattern %q matches no selections", pattern)
		}
		if quantifier == "all" {
			needed = len(matched)
		}
		if quantifier == "any" {
			needed = 1
		}
		return func(v matchBits) bool {
			count := 0
			for _, index := range matched {
				if v.has(index) {
					count++
				}
				if count >= needed {
					return true
				}
			}
			return false
		}
	}
	index, exists := p.indices[token]
	if !exists {
		return p.fail("unknown selection %q", token)
	}
	return func(v matchBits) bool { return v.has(index) }
}
