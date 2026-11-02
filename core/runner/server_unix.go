//go:build !windows

package runner

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// Server runs commands for one orchestrator. It is meant to live in its own
// container that holds no secrets and cannot reach the database.
type Server struct {
	// WorkBase is the directory holding the agents' working directories,
	// mounted at the same path in the orchestrator and the runner.
	WorkBase string
	// PeerUID is the only uid allowed to connect (SO_PEERCRED); -1 allows any,
	// for tests. Commands themselves run as another uid or in another mount
	// namespace, so they cannot use the socket to escape their root.
	PeerUID int
	// Isolate, if set, prepares each command before it starts (mount namespace
	// that only shows the command's root, privilege drop). root is the absolute
	// task or session directory the command is confined to.
	Isolate func(cmd *exec.Cmd, root string) error
	// BaseEnv is prepended to every command's environment (PATH, LANG...).
	// Nothing is inherited from the runner process.
	BaseEnv []string
}

// rootKinds are the top-level directories under WorkBase that hold one root
// per task or chat session.
var rootKinds = map[string]bool{"tasks": true, "sessions": true}

// resolveRoot checks that dir is an existing directory inside a task or session
// root under WorkBase, after resolving symlinks, and returns the cleaned dir and
// that root.
func (s *Server) resolveRoot(dir string) (string, string, error) {
	if !filepath.IsAbs(dir) {
		return "", "", fmt.Errorf("directory %q is not absolute", dir)
	}
	base, err := filepath.EvalSymlinks(s.WorkBase)
	if err != nil {
		return "", "", fmt.Errorf("work base: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", "", fmt.Errorf("directory %q: %w", dir, err)
	}
	rel, err := filepath.Rel(base, resolved)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", "", fmt.Errorf("directory %q is outside the work base", dir)
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) < 2 || !rootKinds[parts[0]] || parts[1] == "" {
		return "", "", fmt.Errorf("directory %q is not inside a task or session root", dir)
	}
	if fi, err := os.Stat(resolved); err != nil || !fi.IsDir() {
		return "", "", fmt.Errorf("directory %q is not a directory", dir)
	}
	return resolved, filepath.Join(base, parts[0], parts[1]), nil
}

// Serve accepts connections until ctx is done or the listener fails.
func (s *Server) Serve(ctx context.Context, l *net.UnixListener) error {
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()
	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.handle(conn)
	}
}

func peerUID(conn *net.UnixConn) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *unix.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if credErr != nil {
		return 0, credErr
	}
	return int(cred.Uid), nil
}

func (s *Server) handle(conn *net.UnixConn) {
	defer conn.Close()
	fw := newFrameWriter(conn)
	fail := func(format string, args ...any) {
		_ = fw.send(frameFailure, []byte(fmt.Sprintf(format, args...)))
	}
	if s.PeerUID >= 0 {
		uid, err := peerUID(conn)
		if err != nil || uid != s.PeerUID {
			log.Printf("[runner] refused connection from uid %d (%v)", uid, err)
			fail("connection refused")
			return
		}
	}
	req, err := readRequest(conn)
	if err != nil {
		fail("%v", err)
		return
	}
	dir, root, err := s.resolveRoot(req.Dir)
	if err != nil {
		fail("%v", err)
		return
	}
	cmd := exec.Command("/bin/sh", "-c", req.Command)
	cmd.Dir = dir
	cmd.Env = append(append(append([]string(nil), s.BaseEnv...), "HOME="+root), req.Env...)
	if s.Isolate != nil {
		if err := s.Isolate(cmd, root); err != nil {
			fail("isolation: %v", err)
			return
		}
	}
	if req.TTY {
		s.runTTY(conn, fw, cmd, req)
	} else {
		s.runPipes(conn, fw, cmd)
	}
}

// proc is a started command whose process group can be signalled until it has
// been waited for; after that its pgid may belong to another process.
type proc struct {
	cmd  *exec.Cmd
	mu   sync.Mutex
	done bool
}

func (p *proc) killGroup(sig syscall.Signal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.done && p.cmd.Process != nil {
		_ = syscall.Kill(-p.cmd.Process.Pid, sig)
	}
}

func (p *proc) wait() *os.ProcessState {
	_ = p.cmd.Wait()
	p.mu.Lock()
	p.done = true
	p.mu.Unlock()
	return p.cmd.ProcessState
}

// relayClient applies the client's frames until the connection ends. A dropped
// connection (the orchestrator killed the client) kills the command.
func relayClient(conn io.Reader, p *proc, stdin io.WriteCloser, resize func(rows, cols uint16)) {
	for {
		typ, payload, err := readFrame(conn)
		if err != nil {
			p.killGroup(syscall.SIGKILL)
			return
		}
		switch typ {
		case frameStdin:
			if stdin != nil {
				_, _ = stdin.Write(payload)
			}
		case frameCloseStdin:
			if stdin != nil {
				_ = stdin.Close()
			}
		case frameResize:
			if resize != nil && len(payload) == 4 {
				resize(binary.BigEndian.Uint16(payload), binary.BigEndian.Uint16(payload[2:]))
			}
		case frameSignal:
			if len(payload) == 1 && allowedSignal(syscall.Signal(payload[0])) {
				p.killGroup(syscall.Signal(payload[0]))
			}
		}
	}
}

func allowedSignal(sig syscall.Signal) bool {
	switch sig {
	case syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL:
		return true
	}
	return false
}

// keepOpen is a writer whose Close does nothing.
type keepOpen struct{ io.Writer }

func (keepOpen) Close() error { return nil }

func exitFrame(fw *frameWriter, state *os.ProcessState) {
	code := int32(-1)
	if state != nil {
		code = int32(state.ExitCode())
	}
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(code))
	_ = fw.send(frameExit, b[:])
}

func (s *Server) runPipes(conn *net.UnixConn, fw *frameWriter, cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Stdout = streamWriter{fw, frameStdout}
	cmd.Stderr = streamWriter{fw, frameStderr}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = fw.send(frameFailure, []byte(err.Error()))
		return
	}
	if err := cmd.Start(); err != nil {
		_ = fw.send(frameFailure, []byte(err.Error()))
		return
	}
	p := &proc{cmd: cmd}
	go relayClient(conn, p, stdin, nil)
	exitFrame(fw, p.wait())
}

func (s *Server) runTTY(conn *net.UnixConn, fw *frameWriter, cmd *exec.Cmd, req Request) {
	rows, cols := req.Rows, req.Cols
	if rows == 0 || cols == 0 {
		rows, cols = 24, 80
	}
	// pty.Start makes the command a session leader, so its pid is also its
	// process group id and killGroup reaches its children.
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		_ = fw.send(frameFailure, []byte(err.Error()))
		return
	}
	defer ptmx.Close()
	p := &proc{cmd: cmd}
	// Closing the master would hang up the session, so a client's end of input
	// is ignored here; a terminal user ends input with ^D instead.
	go relayClient(conn, p, keepOpen{ptmx}, func(r, c uint16) {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: r, Cols: c})
	})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(streamWriter{fw, frameStdout}, ptmx)
		if err != nil && !errors.Is(err, syscall.EIO) && !errors.Is(err, os.ErrClosed) {
			log.Printf("[runner] pty read: %v", err)
		}
	}()
	state := p.wait()
	// The slave side is closed once the command and its children exit; the
	// copy then ends with EIO after draining what is left.
	wg.Wait()
	exitFrame(fw, state)
}
