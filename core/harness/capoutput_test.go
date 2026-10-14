// Nexus IA v6.0 — Absorbido de norma/harness
// Upstream: github.com/Autumn-27/norma/harness
// Licencia: AGPL-3.0

package harness

import (
	"strings"
	"testing"

	"github.com/nexus-ia/nexus/core/llm"
	"github.com/nexus-ia/nexus/core/tool"
)

// The global post-tool net caps an oversized text block regardless of the tool.
func TestCapOutputCapsOversizedTextBlock(t *testing.T) {
	tc := &tool.ToolContext{MaxOutputChars: 100} // no OutputDir → head+tail truncation
	long := strings.Repeat("z", 5000)
	res := tool.Result{Content: []llm.ContentBlock{llm.TextBlock(long)}}

	out := capOutput(tc, res)
	got := out.Content[0].Text
	if len(got) >= 5000 || !strings.Contains(got, "characters truncated]") {
		t.Fatalf("capOutput did not cap oversized block: len=%d", len(got))
	}
}

// Output already within budget is passed through untouched.
func TestCapOutputLeavesSmallOutputIntact(t *testing.T) {
	tc := &tool.ToolContext{MaxOutputChars: 100}
	res := tool.Result{Content: []llm.ContentBlock{llm.TextBlock("fine")}}

	if out := capOutput(tc, res); out.Content[0].Text != "fine" {
		t.Fatalf("small output changed: %q", out.Content[0].Text)
	}
}

// Output a tool already captured (carries the spill marker) is not re-processed,
// so the net never double-truncates.
func TestCapOutputIdempotentOnAlreadyCaptured(t *testing.T) {
	tc := &tool.ToolContext{MaxOutputChars: 100}
	pre := strings.Repeat("a", 100) + "\n\n... <persisted-output>[Output too large: full 9999 bytes ... saved to x.txt </persisted-output>"
	res := tool.Result{Content: []llm.ContentBlock{llm.TextBlock(pre)}}

	if out := capOutput(tc, res); out.Content[0].Text != pre {
		t.Fatalf("capOutput re-processed already-captured output")
	}
}
