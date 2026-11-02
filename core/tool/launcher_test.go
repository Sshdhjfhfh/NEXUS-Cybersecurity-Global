package tool

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// Every way an agent runs a shell command must go through the launcher, or a
// remote runner would leave one path executing inside the orchestrator.
func TestCommandsGoThroughLauncher(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	SetLauncher(func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
		mu.Lock()
		seen = append(seen, command+"|"+dir+"|"+strings.Join(env, ","))
		mu.Unlock()
		return LocalLauncher(ctx, "echo routed", dir, env)
	})
	t.Cleanup(func() { SetLauncher(nil) })
	dir := t.TempDir()
	env := []string{"HTTP_PROXY=http://127.0.0.1:8788"}

	raw, _ := json.Marshal(map[string]any{"command": "echo sync"})
	res, err := NewBash().Call(context.Background(), raw, &ToolContext{WorkingDir: dir, Env: env})
	if err != nil || !strings.Contains(res.Flatten(), "routed") {
		t.Fatalf("sync Bash: %v %q", err, res.Flatten())
	}

	m, err := NewManager("launcher-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.KillAll() })
	_, finished, out, err := m.RunForeground(context.Background(), SpawnSpec{Command: "echo managed", WorkingDir: dir, Env: env}, 10*time.Second)
	if err != nil || !finished || !strings.Contains(out, "routed") {
		t.Fatalf("managed Bash: finished=%v err=%v out=%q", finished, err, out)
	}

	want := []string{"echo sync|" + dir + "|" + env[0], "echo managed|" + dir + "|" + env[0]}
	if ptySupported() {
		if _, err := m.openSession("echo session", dir, env, 0, 0, "", nil); err != nil {
			t.Fatal(err)
		}
		want = append(want, "echo session|"+dir+"|"+env[0])
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("launcher calls:\n%s\nwant:\n%s", strings.Join(seen, "\n"), strings.Join(want, "\n"))
	}
}

func TestSetLauncherNilRestoresLocal(t *testing.T) {
	SetLauncher(func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
		return LocalLauncher(ctx, "echo replaced", dir, env)
	})
	SetLauncher(nil)
	raw, _ := json.Marshal(map[string]any{"command": "echo local"})
	res, err := NewBash().Call(context.Background(), raw, nil)
	if err != nil || !strings.Contains(res.Flatten(), "local") {
		t.Fatalf("after SetLauncher(nil): %v %q", err, res.Flatten())
	}
}
