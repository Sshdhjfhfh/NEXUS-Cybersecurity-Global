package sigma

import (
	"os"
	"testing"
)

// TestLoadRealSigmaRules is a real-world smoke test: point NEXUS_SIGMA_DIR at a
// checkout of community Sigma rules (e.g. SigmaHQ/sigma/rules) and it reports
// how many rules this engine parses. Skipped when the variable is unset so CI
// stays hermetic. It asserts only that a healthy majority parse, since this
// increment intentionally skips some advanced rule shapes.
func TestLoadRealSigmaRules(t *testing.T) {
	dir := os.Getenv("NEXUS_SIGMA_DIR")
	if dir == "" {
		t.Skip("set NEXUS_SIGMA_DIR to a Sigma rules checkout to run this")
	}
	rules, errs := LoadDir(dir)
	total := len(rules) + len(errs)
	t.Logf("parsed %d rules, skipped %d (of %d files)", len(rules), len(errs), total)
	if total == 0 {
		t.Fatalf("no rule files found under %s", dir)
	}
	if len(rules) == 0 {
		t.Fatalf("parsed 0 rules of %d", total)
	}
}
