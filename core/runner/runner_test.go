//go:build !windows

package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"

	"github.com/nexus-ia/nexus/core/tool"
)

// The test binary doubles as the per-command client (the way the orchestrator
// binary does with ClientSubcommand) and as the in-namespace init (InitSubcommand).
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == InitSubcommand {
		os.Exit(InitMain(os.Args[2:]))
	}
	if os.Getenv(envRequest) != "" {
		os.Exit(ClientMain())
	}
	os.Exit(m.Run())
}

type fixture struct {
	base, socket string
	srv          *Server
}

func newFixture(t *testing.T, peerUID int) *fixture {
	return newFixtureOpt(t, peerUID, nil)
}

// newFixtureOpt is newFixture with an optional isolator, set before Serve starts
// so -race sees no write to srv.Isolate racing the accept loop.
func newFixtureOpt(t *testing.T, peerUID int, isolate func(*exec.Cmd, string) error) *fixture {
	t.Helper()
	base := t.TempDir()
	for _, d := range []string{"tasks/7/i3", "tasks/8", "sessions/abc"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Unix socket paths are limited to ~100 bytes; t.TempDir() can be longer.
	sockDir, err := os.MkdirTemp("", "rn")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(sockDir) })
	socket := filepath.Join(sockDir, "s")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{WorkBase: base, PeerUID: peerUID, BaseEnv: []string{"PATH=/usr/bin:/bin"}, Isolate: isolate}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Serve(ctx, l) }()
	return &fixture{base: base, socket: socket, srv: srv}
}

func (f *fixture) dir(rel string) string { return filepath.Join(f.base, rel) }

func (f *fixture) exec(t *testing.T, req Request, stdin string) (int, string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	var in *strings.Reader
	if stdin != "" {
		in = strings.NewReader(stdin)
	}
	var r interface{ Read([]byte) (int, error) }
	if in != nil {
		r = in
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	code, err := Exec(ctx, f.socket, req, r, &out, &errb, nil, nil)
	return code, out.String(), errb.String(), err
}

// TestMountIsolateSeparatesPIDAndProc checks the concrete property the mount
// namespace gives: an agent command runs in a fresh PID namespace with its own
// /proc, so it cannot read the orchestrator's /proc/<pid>/environ, and masked
// paths (e.g. a key directory) are invisible. Skips where unprivileged user
// namespaces are disabled.
func TestMountIsolateSeparatesPIDAndProc(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	secretDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(secretDir, "jwt.key"), []byte("TOPSECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := newFixtureOpt(t, os.Getuid(), MountIsolate(self, []string{secretDir}))

	code, out, errOut, err := f.exec(t, Request{Command: "echo $$", Dir: f.dir("tasks/7")}, "")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "permission denied") {
			t.Skipf("unprivileged user namespaces unavailable: %v", err)
		}
		t.Fatalf("exec: %v (stderr=%q)", err, errOut)
	}
	// In a fresh PID namespace the shell is PID 1.
	if strings.TrimSpace(out) != "1" {
		t.Fatalf("shell pid in namespace = %q, want 1 (stderr=%q)", out, errOut)
	}

	// The orchestrator (this test process) has no /proc entry in the namespace.
	pid := strconv.Itoa(os.Getpid())
	code, _, _, err = f.exec(t, Request{Command: "cat /proc/" + pid + "/environ", Dir: f.dir("tasks/7")}, "")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if code == 0 {
		t.Fatalf("reading /proc/%s/environ succeeded inside the namespace; it must not be visible there", pid)
	}

	// The masked key directory is hidden.
	_, out, _, err = f.exec(t, Request{Command: "cat " + filepath.Join(secretDir, "jwt.key"), Dir: f.dir("tasks/7")}, "")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if strings.Contains(out, "TOPSECRET") {
		t.Fatalf("masked secret leaked through isolation: %q", out)
	}
}

func TestExecStreamsAndExitCode(t *testing.T) {
	f := newFixture(t, os.Getuid())
	code, out, errOut, err := f.exec(t, Request{Command: "echo out; echo err >&2; exit 3", Dir: f.dir("tasks/7")}, "")
	if err != nil || code != 3 || out != "out\n" || errOut != "err\n" {
		t.Fatalf("code=%d out=%q err=%q %v", code, out, errOut, err)
	}
	code, out, _, err = f.exec(t, Request{Command: "cat", Dir: f.dir("tasks/7")}, "hello runner")
	if err != nil || code != 0 || out != "hello runner" {
		t.Fatalf("stdin: code=%d out=%q %v", code, out, err)
	}
}

func TestExecEnvironmentIsExplicit(t *testing.T) {
	t.Setenv("RUNNER_TEST_SECRET", "s3cret")
	f := newFixture(t, os.Getuid())
	_, out, _, err := f.exec(t, Request{
		Command: `echo "${RUNNER_TEST_SECRET:-none} $HTTP_PROXY $HOME"; pwd`,
		Dir:     f.dir("tasks/7/i3"),
		Env:     []string{"HTTP_PROXY=http://127.0.0.1:8788"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	root, _ := filepath.EvalSymlinks(f.dir("tasks/7"))
	cwd, _ := filepath.EvalSymlinks(f.dir("tasks/7/i3"))
	if want := "none http://127.0.0.1:8788 " + root + "\n" + cwd + "\n"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestExecConfinesDirectory(t *testing.T) {
	f := newFixture(t, os.Getuid())
	outside := t.TempDir()
	if err := os.Symlink(outside, f.dir("tasks/7/escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(f.dir("tasks/8"), f.dir("tasks/7/other-task")); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{
		outside,
		f.base,
		f.dir("tasks"),
		f.dir("tasks/7/../../.."),
		f.dir("tasks/7/escape"),
		f.dir("other/1"),
		f.dir("tasks/404"),
		"tasks/7",
		"",
	} {
		_, _, _, err := f.exec(t, Request{Command: "pwd", Dir: dir}, "")
		if err == nil {
			t.Errorf("dir %q was accepted", dir)
		}
	}
	// A symlink into another task resolves to that task's root, never task 7's.
	_, out, _, err := f.exec(t, Request{Command: `echo "$HOME"`, Dir: f.dir("tasks/7/other-task")}, "")
	if root8, _ := filepath.EvalSymlinks(f.dir("tasks/8")); err != nil || out != root8+"\n" {
		t.Fatalf("symlinked dir: %q %v", out, err)
	}
	if _, out, _, err := f.exec(t, Request{Command: "pwd", Dir: f.dir("sessions/abc")}, ""); err != nil || !strings.HasSuffix(out, "sessions/abc\n") {
		t.Fatalf("session root: %q %v", out, err)
	}
}

func TestExecRefusesOtherPeers(t *testing.T) {
	f := newFixture(t, os.Getuid()+1)
	if _, _, _, err := f.exec(t, Request{Command: "true", Dir: f.dir("tasks/7")}, ""); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("got %v, want a refusal", err)
	}
}

func TestExecRefusesOtherVersions(t *testing.T) {
	f := newFixture(t, os.Getuid())
	conn, err := net.Dial("unix", f.socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	raw, _ := json.Marshal(Request{V: Version + 1, Command: "true", Dir: f.dir("tasks/7")})
	if err := writeFrame(conn, 0, raw); err != nil {
		t.Fatal(err)
	}
	typ, payload, err := readFrame(conn)
	if err != nil || typ != frameFailure || !strings.Contains(string(payload), "version") {
		t.Fatalf("typ=%d payload=%q err=%v", typ, payload, err)
	}
}

func pidAlive(pid int) bool { return syscall.Kill(pid, 0) == nil }

func waitPID(t *testing.T, path string) int {
	t.Helper()
	for i := 0; i < 200; i++ {
		if raw, err := os.ReadFile(path); err == nil && strings.HasSuffix(string(raw), "\n") {
			pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
			if err == nil {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("command did not write its pid")
	return 0
}

// When the orchestrator kills the client (task timeout, KillAll), the
// connection drops and the runner must kill the command and its children.
func TestDisconnectKillsCommandGroup(t *testing.T) {
	f := newFixture(t, os.Getuid())
	pidFile := f.dir("tasks/7/child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := Exec(ctx, f.socket, Request{Command: "sleep 30 & echo $! > child.pid; wait", Dir: f.dir("tasks/7")},
			nil, &bytes.Buffer{}, &bytes.Buffer{}, nil, nil)
		done <- err
	}()
	child := waitPID(t, pidFile)
	cancel()
	if err := <-done; err == nil {
		t.Fatal("Exec returned no error after cancel")
	}
	for i := 0; i < 200 && pidAlive(child); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if pidAlive(child) {
		_ = syscall.Kill(child, syscall.SIGKILL)
		t.Fatal("background child survived the disconnect")
	}
}

func TestSignalIsForwarded(t *testing.T) {
	f := newFixture(t, os.Getuid())
	sigs := make(chan os.Signal, 1)
	var out bytes.Buffer
	done := make(chan int, 1)
	go func() {
		code, _ := Exec(context.Background(), f.socket, Request{
			Command: `trap 'echo got-term; exit 5' TERM; echo $$ > sh.pid; while :; do sleep 0.05; done`,
			Dir:     f.dir("tasks/7"),
		}, nil, &out, &bytes.Buffer{}, sigs, nil)
		done <- code
	}()
	waitPID(t, f.dir("tasks/7/sh.pid"))
	sigs <- syscall.SIGTERM
	select {
	case code := <-done:
		if code != 5 || !strings.Contains(out.String(), "got-term") {
			t.Fatalf("code=%d out=%q", code, out.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("SIGTERM was not forwarded")
	}
}

func TestTTYMode(t *testing.T) {
	f := newFixture(t, os.Getuid())
	code, out, _, err := f.exec(t, Request{Command: "test -t 0 && echo is-tty; stty size", Dir: f.dir("tasks/7"), TTY: true, Rows: 30, Cols: 100}, "")
	if err != nil || code != 0 || !strings.Contains(out, "is-tty") || !strings.Contains(out, "30 100") {
		t.Fatalf("code=%d out=%q err=%v", code, out, err)
	}
}

// Interactive sessions start the client under a local PTY: it must switch that
// terminal to raw mode and relay keystrokes to the remote PTY, which does the
// line editing.
func TestClientUnderPTY(t *testing.T) {
	f := newFixture(t, os.Getuid())
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(Request{V: Version, Command: `read x; echo "got:$x"; stty size`, Dir: f.dir("tasks/7")})
	cmd := exec.Command(self)
	cmd.Env = []string{envSocket + "=" + f.socket, envRequest + "=" + string(raw), "GOCOVERDIR=" + t.TempDir()}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 33, Cols: 101})
	if err != nil {
		t.Fatal(err)
	}
	defer ptmx.Close()
	if _, err := ptmx.Write([]byte("abc\r")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	done := make(chan struct{})
	go func() { _, _ = out.ReadFrom(ptmx); close(done) }()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("client exit: %v (output %q)", err, out.String())
	}
	<-done
	if !strings.Contains(out.String(), "got:abc") || !strings.Contains(out.String(), "33 101") {
		t.Fatalf("output %q", out.String())
	}
}

func TestFrameLimit(t *testing.T) {
	var b bytes.Buffer
	b.Write([]byte{frameStdin, 0xff, 0xff, 0xff, 0xff})
	if _, _, err := readFrame(&b); err != errFrameTooLarge {
		t.Fatalf("got %v", err)
	}
	if err := writeFrame(&b, frameStdin, make([]byte, maxFrame+1)); err != errFrameTooLarge {
		t.Fatalf("write: %v", err)
	}
}

// End to end through core/tool: with the runner launcher installed, Bash,
// background tasks and interactive sessions all execute in the runner.
func TestToolCommandsRunInRunner(t *testing.T) {
	f := newFixture(t, os.Getuid())
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launch := Launcher(f.socket)
	coverDir := t.TempDir()
	tool.SetLauncher(func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
		cmd := launch(ctx, command, dir, env)
		cmd.Path = self // the test binary acts as the client (see TestMain)
		// Under -cover the client would otherwise print a GOCOVERDIR warning.
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverDir)
		return cmd
	})
	t.Cleanup(func() { tool.SetLauncher(nil) })
	t.Setenv("RUNNER_TEST_SECRET", "s3cret")
	dir := f.dir("tasks/7/i3")

	raw, _ := json.Marshal(map[string]any{"command": `echo "in:$HOME:${RUNNER_TEST_SECRET:-none}"; exit 4`})
	res, err := tool.NewBash().Call(context.Background(), raw, &tool.ToolContext{WorkingDir: dir})
	root, _ := filepath.EvalSymlinks(f.dir("tasks/7"))
	if err != nil || !strings.Contains(res.Flatten(), "in:"+root+":none") || !strings.Contains(res.Flatten(), "exit code 4") {
		t.Fatalf("Bash via runner: %v %q", err, res.Flatten())
	}

	m, err := tool.NewManager("runner-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.KillAll() })
	_, finished, out, err := m.RunForeground(context.Background(), tool.SpawnSpec{Command: "pwd", WorkingDir: dir}, 20*time.Second)
	if cwd, _ := filepath.EvalSymlinks(dir); err != nil || !finished || strings.TrimSpace(out) != cwd {
		t.Fatalf("managed: finished=%v err=%v out=%q", finished, err, out)
	}

	// A directory outside any task root is refused before anything runs.
	raw, _ = json.Marshal(map[string]any{"command": "touch should-not-exist"})
	res, _ = tool.NewBash().Call(context.Background(), raw, &tool.ToolContext{WorkingDir: t.TempDir()})
	if !strings.Contains(res.Flatten(), "outside the work base") {
		t.Fatalf("outside dir: %q", res.Flatten())
	}
}
