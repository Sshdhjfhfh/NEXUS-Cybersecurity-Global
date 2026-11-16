package tool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfinePathAllowsInsideRejectsEscape(t *testing.T) {
	root := t.TempDir()
	tc := &ToolContext{WorkingDir: root}
	// inside the task dir → allowed (including not-yet-existing files)
	for _, p := range []string{root, filepath.Join(root, "sub", "new.txt")} {
		if err := confinePath(tc, p); err != nil {
			t.Fatalf("inside path rejected %q: %v", p, err)
		}
	}
	// escapes → rejected
	for _, p := range []string{
		"/app/keys/jwt.key", "/etc/passwd",
		filepath.Join(root, "..", "other"),
		filepath.Join(root, "..", filepath.Base(root)+"-sibling", "x"),
	} {
		if err := confinePath(tc, p); err == nil {
			t.Fatalf("escape not rejected: %q", p)
		}
	}
	// no sandbox (empty WorkingDir / nil tc) → not confined
	if err := confinePath(&ToolContext{WorkingDir: ""}, "/etc/passwd"); err != nil {
		t.Fatalf("empty WorkingDir must not confine: %v", err)
	}
	if err := confinePath(nil, "/etc/passwd"); err != nil {
		t.Fatalf("nil tc must not confine: %v", err)
	}
}

func TestConfinePathBlocksSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := confinePath(&ToolContext{WorkingDir: root}, filepath.Join(link, "secret.txt")); err == nil {
		t.Fatal("symlink escape not rejected")
	}
}

func TestConfineFileToolsEscapeHatch(t *testing.T) {
	tc := &ToolContext{WorkingDir: t.TempDir()}
	t.Setenv("NEXUS_CONFINE_FILE_TOOLS", "0")
	if err := confinePath(tc, "/etc/passwd"); err != nil {
		t.Fatalf("escape hatch (=0) should disable confinement: %v", err)
	}
	t.Setenv("NEXUS_CONFINE_FILE_TOOLS", "")
	if err := confinePath(tc, "/etc/passwd"); err == nil {
		t.Fatal("confinement must be ON by default")
	}
}
