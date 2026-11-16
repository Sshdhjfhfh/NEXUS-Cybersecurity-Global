package tool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// confinePath rejects a resolved file path that escapes the task's working
// directory. It is the path-level counterpart of the command runner's directory
// confinement: the file tools (Read/Write/Edit/Grep/Glob) run in the orchestrator
// and, without this, resolvePath returns any absolute path unchanged — so an
// agent could read /app/keys/jwt.key, /proc/<pid>/environ or another task's data
// without ever using Bash. When WorkingDir is empty (no task sandbox) nothing is
// confined, so system and non-task uses are unaffected. Set
// NEXUS_CONFINE_FILE_TOOLS=0 to disable (an emergency escape hatch only).
func confinePath(tc *ToolContext, resolved string) error {
	if tc == nil || tc.WorkingDir == "" || !confineFileTools() {
		return nil
	}
	root, err := filepath.EvalSymlinks(tc.WorkingDir)
	if err != nil {
		root = filepath.Clean(tc.WorkingDir)
	}
	real := resolveExisting(filepath.Clean(resolved))
	rel, err := filepath.Rel(root, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("acceso denegado: %q está fuera del directorio de la tarea", resolved)
	}
	return nil
}

// resolveExisting resolves symlinks on the longest existing ancestor of p and
// appends the remaining (not-yet-existing) tail, so a symlink in the existing
// part cannot be used to escape even when the target file does not exist yet
// (e.g. a Write to a new file).
func resolveExisting(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	dir := filepath.Dir(p)
	if dir == p { // reached the filesystem root
		return p
	}
	return filepath.Join(resolveExisting(dir), filepath.Base(p))
}

func confineFileTools() bool {
	v := strings.TrimSpace(os.Getenv("NEXUS_CONFINE_FILE_TOOLS"))
	return v != "0" && !strings.EqualFold(v, "false")
}
