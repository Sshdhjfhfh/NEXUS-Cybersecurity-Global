package sigma

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// Original fixtures derived from the public Sigma semantic contract, not from
// third-party rule content. Expected verdicts are authored independently.
func TestCountConditionTruthTable(t *testing.T) {
	r := mustParse(t, "title: count\ndetection:\n  one:\n    a: yes\n  two:\n    b: yes\n  three:\n    c: yes\n  condition: 2 of them\n")
	for mask := 0; mask < 8; mask++ {
		ev := Event{}
		count := 0
		for bit, name := range []string{"a", "b", "c"} {
			if mask&(1<<bit) != 0 {
				ev[name] = "yes"
				count++
			}
		}
		if got := mustMatch(t, r, ev); got != (count >= 2) {
			t.Errorf("mask %03b: got %t, want %t", mask, got, count >= 2)
		}
	}
}

func TestUnsupportedRulesAreRejected(t *testing.T) {
	for name, detection := range map[string]string{
		"modifier":              "  sel:\n    path|base64: data\n  condition: sel\n",
		"unknown selection":     "  sel:\n    path: data\n  condition: missing\n",
		"empty wildcard":        "  sel:\n    path: data\n  condition: all of missing*\n",
		"invalid regex":         "  sel:\n    path|re: '[broken'\n  condition: sel\n",
		"timeframe":             "  sel:\n    path: data\n  timeframe: 1m\n  condition: sel\n",
		"empty values":          "  sel:\n    path|all: []\n  condition: sel\n",
		"all scalar":            "  sel:\n    path|all: data\n  condition: sel\n",
		"all singleton":         "  sel:\n    path|all: [data]\n  condition: sel\n",
		"conflicting modifiers": "  sel:\n    path|contains|endswith: data\n  condition: sel\n",
		"trailing document":     "  sel:\n    path: data\n  condition: sel\n---\ntitle: ignored\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte("title: invalid\ndetection:\n" + detection)); err == nil {
				t.Fatal("unsupported semantics accepted without an error")
			}
		})
	}
}

func TestAllModifierRequiresEveryValue(t *testing.T) {
	r := mustParse(t, "title: all values\ndetection:\n  sel:\n    path|contains|all: [admin, report]\n  condition: sel\n")
	for path, want := range map[string]bool{"/admin/report": true, "/ADMIN/REPORT": true, "/admin/index": false, "/public/report": false, "/public/index": false} {
		if got := mustMatch(t, r, Event{"path": path}); got != want {
			t.Errorf("path %q: got %v, want %v", path, got, want)
		}
	}
}

func TestRegexPreservesCaseAndEscapes(t *testing.T) {
	r := mustParse(t, "title: regex\ndetection:\n  sel:\n    path|re: '^/ADMIN/\\D+$'\n  condition: sel\n")
	for _, tc := range []struct {
		path string
		want bool
	}{{"/ADMIN/abc", true}, {"/admin/abc", false}, {"/ADMIN/123", false}} {
		if got := mustMatch(t, r, Event{"path": tc.path}); got != tc.want {
			t.Errorf("path %q: got %t, want %t", tc.path, got, tc.want)
		}
	}
}

func TestWildcardAndMissingFieldSemantics(t *testing.T) {
	for _, tc := range []struct {
		field string
		value string
		event Event
		want  bool
	}{
		{"path", "'/ADMIN/*.php'", Event{"path": "/admin/report.php"}, true},
		{"path", "'/ADMIN/?.php'", Event{"path": "/ADMIN/ab.php"}, false},
		{"path", "'/literal/\\*.txt'", Event{"path": "/literal/*.txt"}, true},
		{"path", "null", Event{}, true},
		{"path", "null", Event{"path": "null"}, false},
		{"path|exists", "false", Event{}, true},
		{"path|exists", "false", Event{"path": ""}, false},
	} {
		r := mustParse(t, fmt.Sprintf("title: field\ndetection:\n  sel:\n    %s: %s\n  condition: sel\n", tc.field, tc.value))
		if got := mustMatch(t, r, tc.event); got != tc.want {
			t.Errorf("%s %s, event=%v: got %t, want %t", tc.field, tc.value, tc.event, got, tc.want)
		}
	}
}

func BenchmarkSigmaMatch(b *testing.B) {
	r, err := Parse([]byte(ruleAndContains))
	if err != nil {
		b.Fatal(err)
	}
	ev := Event{"uri": "/webdav/update.exe"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if matched, err := r.Match(ev); err != nil || !matched {
			b.Fatal(matched, err)
		}
	}
}

func BenchmarkSigmaCompiledMatch(b *testing.B) {
	r, err := Parse([]byte(ruleAndContains))
	if err != nil {
		b.Fatal(err)
	}
	c, err := r.Compile()
	if err != nil {
		b.Fatal(err)
	}
	ev := Event{"uri": "/webdav/update.exe"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if !c.Match(ev) {
			b.Fatal("missed positive event")
		}
	}
}

func TestCompiledRuleOwnsPredicatesAndIsConcurrent(t *testing.T) {
	r := mustParse(t, ruleAndContains)
	c, err := r.Compile()
	if err != nil {
		t.Fatal(err)
	}
	r.Detection["sel_webdav"].(map[string]any)["uri|contains"] = "changed"
	r.Detection["condition"] = "invalid"
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if !c.Match(Event{"uri": "/webdav/update.exe"}) || c.Match(Event{"uri": "/public/readme.txt"}) {
					t.Error("compiled rule changed or misclassified")
				}
			}
		}()
	}
	wg.Wait()
}

func TestConditionPatternsPrecedenceAndHiddenSelections(t *testing.T) {
	r := mustParse(t, "title: conditions\ndetection:\n  _helper:\n    x: hidden\n  select_a_end:\n    x: a\n  select_b_end:\n    x: b\n  filter:\n    ignored: true\n  condition: (1 of select*_end and not filter) or all of them\n")
	for _, tc := range []struct {
		e    Event
		want bool
	}{
		{Event{"x": "a"}, true}, {Event{"x": "b", "ignored": true}, false}, {Event{"x": "hidden"}, false},
	} {
		if got := mustMatch(t, r, tc.e); got != tc.want {
			t.Errorf("%v: %t != %t", tc.e, got, tc.want)
		}
	}
	_, err := Parse([]byte("title: bound\ndetection:\n  s:\n    a: b\n  condition: " + strings.Repeat("not ", maxDepth+2) + "s\n"))
	if err == nil {
		t.Fatal("unbounded condition nesting accepted")
	}
}

func FuzzParseAndCompile(f *testing.F) {
	for _, s := range []string{ruleAndContains, ruleOneOfThem, ruleNot, "title: broken\ndetection: ["} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > MaxRuleBytes {
			return
		}
		r, err := Parse([]byte(s))
		if err != nil {
			return
		}
		c, err := r.Compile()
		if err != nil {
			t.Fatalf("parsed rule cannot compile: %v", err)
		}
		_ = c.Match(Event{"uri": "/benign", "status": 200})
	})
}
