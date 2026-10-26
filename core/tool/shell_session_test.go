// Nexus IA v6.0 — Absorbido de norma/tool
// Upstream: github.com/Autumn-27/norma/tool
// Licencia: AGPL-3.0

package tool

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestShellToolsJSONSessionID drives the tools via their JSON input (snake_case
// session_id) to guard against missing json tags — without tags session_id never
// binds and every read/send/close reports "会话不存在".
func TestShellToolsJSONSessionID(t *testing.T) {
	if !ptySupported() {
		t.Skip("no PTY on this platform")
	}
	m, err := NewManager("test-tools")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cleanup()
	s, err := m.openSession("cat", "", nil, 24, 80, "hands-free", nil)
	if err != nil {
		t.Fatal(err)
	}
	tc := &ToolContext{Tasks: m}
	ctx := context.Background()
	res, _ := NewShellRead().Call(ctx, []byte(`{"session_id":"`+s.id+`","view":"stream"}`), tc)
	if res.IsError {
		t.Fatalf("shell_read by session_id failed: %s", res.Flatten())
	}
	res, _ = NewShellClose().Call(ctx, []byte(`{"session_id":"`+s.id+`"}`), tc)
	if res.IsError {
		t.Fatalf("shell_close by session_id failed: %s", res.Flatten())
	}
}

// TestShellMonitor: a session in monitor mode fires a notification when a watched
// pattern shows up in the output — without ending the session.
func TestShellMonitor(t *testing.T) {
	if !ptySupported() {
		t.Skip("no PTY on this platform")
	}
	m, err := NewManager("test-mon")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cleanup()
	s, err := m.openSession("printf 'START\\n'; sleep 0.3; printf 'TRIGGERWORD\\n'; sleep 5", "", nil, 24, 80, "monitor", nil)
	if err != nil {
		t.Fatal(err)
	}
	go m.monitorWaiter(s, []*regexp.Regexp{regexp.MustCompile("TRIGGERWORD")})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, n := range m.DrainNotifications() {
			if strings.Contains(n.Summary, "TRIGGERWORD") {
				return // success
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("expected monitor notification for TRIGGERWORD")
}

// TestShellSessionEcho drives a real PTY: `cat` echoes stdin, so sending a line
// should show up in the incremental read — exercising open/send/read end to end.
func TestShellSessionEcho(t *testing.T) {
	if !ptySupported() {
		t.Skip("no PTY on this platform")
	}
	m, err := NewManager("test-shell")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cleanup()

	s, err := m.openSession("cat", "", nil, 24, 80, "hands-free", nil)
	if err != nil {
		t.Fatalf("openSession: %v", err)
	}
	if _, err := s.ptmx.Write([]byte("hello-pty\r")); err != nil {
		t.Fatalf("write: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var out string
	for time.Now().Before(deadline) {
		out, _, _ = s.readSince(0, 0, 8192, true)
		if strings.Contains(out, "hello-pty") {
			return // success
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected echoed input in output, got %q", out)
}

func TestKeyToBytes(t *testing.T) {
	cases := map[string]string{
		"ctrl+c": "\x03", "ctrl+d": "\x04", "enter": "\r", "tab": "\t",
		"esc": "\x1b", "up": "\x1b[A", "backspace": "\x7f",
	}
	for k, want := range cases {
		got, ok := keyToBytes(k)
		if !ok || got != want {
			t.Errorf("keyToBytes(%q)=%q,%v want %q", k, got, ok, want)
		}
	}
	if _, ok := keyToBytes("bogus"); ok {
		t.Errorf("expected bogus key to be rejected")
	}
}

func TestHexToBytes(t *testing.T) {
	got := hexToBytes([]string{"0x1b", "5b", "0x41"})
	if string(got) != "\x1b[A" {
		t.Errorf("hexToBytes=%q want ESC[A", got)
	}
}

func TestReadSinceIncremental(t *testing.T) {
	s := &ptySession{notifyCh: make(chan struct{}, 1)}
	s.append([]byte("line1\nline2\n"))
	out, cur, _ := s.readSince(0, 0, 8192, false)
	if out != "line1\nline2\n" {
		t.Fatalf("got %q", out)
	}
	s.append([]byte("line3\n"))
	out2, _, _ := s.readSince(cur, 0, 8192, false)
	if out2 != "line3\n" {
		t.Fatalf("incremental got %q", out2)
	}
}

func TestWaitCompleteIgnoresPreviousPrompt(t *testing.T) {
	s := &ptySession{notifyCh: make(chan struct{}, 1), done: make(chan struct{})}
	s.append([]byte("old output\n>>> ")) // leave its notification queued
	before := s.outputCursor()
	got := s.waitComplete(context.Background(), time.Second, 0, 30*time.Millisecond,
		regexp.MustCompile(`>>> $`), "", before)
	if got != "timeout" {
		t.Fatalf("old prompt completed a new command: %s", got)
	}
}

func TestWaitCompleteUsesOnlyNewOutput(t *testing.T) {
	s := &ptySession{notifyCh: make(chan struct{}, 1), done: make(chan struct{})}
	s.append([]byte("old output\n>>> "))
	before := s.outputCursor()
	// Output may arrive before waitComplete starts and coalesce with the old ping.
	s.append([]byte("new output\n>>> "))
	got := s.waitComplete(context.Background(), time.Second, 0, time.Second,
		regexp.MustCompile(`>>> $`), "", before)
	if got != "prompt" {
		t.Fatalf("missed new prompt: %s", got)
	}
	out, _, _ := s.readSince(before, 0, 8192, false)
	if out != "new output\n>>> " {
		t.Fatalf("unexpected command output: %q", out)
	}
}

func TestTailAfterHandlesRingEviction(t *testing.T) {
	s := &ptySession{notifyCh: make(chan struct{}, 1)}
	s.append([]byte(strings.Repeat("x", sessRingCap+20)))
	before := s.outputCursor()
	s.append([]byte("fresh"))
	if got := s.tailAfter(before, 4096); got != "fresh" {
		t.Fatalf("fresh output: %q", got)
	}
	if got := s.tailAfter(0, 3); got != "esh" {
		t.Fatalf("bounded tail: %q", got)
	}
}

func TestShellSendWaitsForCurrentCommand(t *testing.T) {
	if !ptySupported() {
		t.Skip("no PTY on this platform")
	}
	m, err := NewManager("test-current-command")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cleanup()
	s, err := m.openSession(`stty -echo; printf 'READY> '; while IFS= read -r line; do sleep 0.05; printf 'ACK:%s\nREADY> ' "$line"; done`,
		t.TempDir(), nil, 24, 80, "interactive", regexp.MustCompile(`READY> $`))
	if err != nil {
		t.Fatal(err)
	}
	if by := s.waitComplete(context.Background(), time.Second, 0, 3*time.Second, s.prompt, "", 0); by != "prompt" {
		t.Fatalf("startup: %s", by)
	}
	// Reproduce a queued old-output ping before a slower command can respond.
	select {
	case s.notifyCh <- struct{}{}:
	default:
	}
	in, _ := json.Marshal(map[string]any{"session_id": s.id, "text": "hello", "submit": true, "quiet_ms": 1000, "timeout_ms": 3000})
	res, err := NewShellSend().Call(context.Background(), in, &ToolContext{Tasks: m})
	if err != nil || res.IsError {
		t.Fatalf("send: %s, %v", res.Flatten(), err)
	}
	var got struct {
		Output string `json:"output"`
		DoneBy string `json:"done_by"`
	}
	if err := json.Unmarshal([]byte(res.Flatten()), &got); err != nil {
		t.Fatal(err)
	}
	if got.DoneBy != "prompt" || !strings.Contains(got.Output, "ACK:hello") {
		t.Fatalf("returned before current command: %+v", got)
	}
}
