package sigma

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDirReportsMissingRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	rules, errs := LoadDir(dir)
	if len(rules) != 0 || len(errs) != 1 {
		t.Fatalf("missing root: got %d rules and %d errors, want 0 and 1", len(rules), len(errs))
	}
	if !errors.Is(errs[0], fs.ErrNotExist) || !strings.Contains(errs[0].Error(), dir) {
		t.Fatalf("missing root error must preserve its cause and path: %v", errs[0])
	}
}

func TestLoadDirEmptyDirectoryIsNotAnError(t *testing.T) {
	rules, errs := LoadDir(t.TempDir())
	if len(rules) != 0 || len(errs) != 0 {
		t.Fatalf("empty directory: got %d rules and %d errors", len(rules), len(errs))
	}
}

func TestLoadDirKeepsValidRulesAlongsideErrors(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"valid.YAML":  ruleAndContains,
		"broken.yml":  "title: [unterminated",
		"ignored.txt": "not a rule",
	} {
		if err := os.WriteFile(filepath.Join(nested, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rules, errs := LoadDir(dir)
	if len(rules) != 1 || rules[0].Title != "WebDAV executable download" || len(errs) != 1 {
		t.Fatalf("mixed tree: rules=%v errors=%v", rules, errs)
	}
	if !strings.Contains(errs[0].Error(), "broken.yml") {
		t.Fatalf("parse error must identify the affected file: %v", errs[0])
	}
}

func TestLoadDirReportsUnreadableSubdirectory(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.Mkdir(blocked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "valid.yml"), []byte(ruleAndContains), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o700) })
	if _, err := os.ReadDir(blocked); err == nil {
		t.Skip("current user can read directories without permission bits")
	}
	rules, errs := LoadDir(dir)
	if len(rules) != 1 || len(errs) != 1 {
		t.Fatalf("unreadable subtree: got %d rules and %d errors, want 1 and 1", len(rules), len(errs))
	}
	if !errors.Is(errs[0], fs.ErrPermission) || !strings.Contains(errs[0].Error(), blocked) {
		t.Fatalf("traversal error must preserve its cause and path: %v", errs[0])
	}
}
