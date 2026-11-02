//go:build !windows

package runner

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// Environment of the per-command client process.
const (
	envSocket  = "NEXUS_RUNNER_SOCKET"
	envRequest = "NEXUS_RUNNER_REQUEST"
)

// ClientSubcommand is the argument that makes the orchestrator binary act as
// the per-command client (see ClientMain).
const ClientSubcommand = "runner-exec"

// Launcher returns a tool.Launcher that runs every agent command in the runner
// listening on socket. The returned process is this binary acting as the client,
// so core/tool keeps its pipes, PTY and process-group handling unchanged.
func Launcher(socket string) func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	return func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
		raw, _ := json.Marshal(Request{V: Version, Command: command, Dir: dir, Env: env})
		cmd := exec.CommandContext(ctx, self, ClientSubcommand)
		cmd.Env = []string{envSocket + "=" + socket, envRequest + "=" + string(raw)}
		return cmd
	}
}

// Exec runs req on the runner at socket, relaying stdin, stdout and stderr,
// forwarding signals and terminal resizes, and returns the command's exit
// status (-1 when it was killed by a signal). Cancelling ctx drops the
// connection, which makes the runner kill the command.
func Exec(ctx context.Context, socket string, req Request, stdin io.Reader, stdout, stderr io.Writer,
	signals <-chan os.Signal, resizes <-chan [2]uint16) (int, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return 0, fmt.Errorf("runner: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	req.V = Version
	fw := newFrameWriter(conn)
	if err := writeRequest(conn, req); err != nil {
		return 0, fmt.Errorf("runner: %w", err)
	}
	if stdin != nil {
		go func() {
			_, _ = io.Copy(streamWriter{fw, frameStdin}, stdin)
			if !req.TTY {
				_ = fw.send(frameCloseStdin, nil)
			}
		}()
	} else if !req.TTY {
		_ = fw.send(frameCloseStdin, nil)
	}
	go func() {
		for {
			select {
			case sig, ok := <-signals:
				if !ok {
					signals = nil
					continue
				}
				if s, ok := sig.(syscall.Signal); ok {
					_ = fw.send(frameSignal, []byte{byte(s)})
				}
			case rc, ok := <-resizes:
				if !ok {
					resizes = nil
					continue
				}
				var b [4]byte
				binary.BigEndian.PutUint16(b[:], rc[0])
				binary.BigEndian.PutUint16(b[2:], rc[1])
				_ = fw.send(frameResize, b[:])
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		typ, payload, err := readFrame(conn)
		if err != nil {
			if ctx.Err() != nil {
				return 0, ctx.Err()
			}
			return 0, fmt.Errorf("runner: connection lost: %w", err)
		}
		switch typ {
		case frameStdout:
			_, _ = stdout.Write(payload)
		case frameStderr:
			_, _ = stderr.Write(payload)
		case frameExit:
			if len(payload) != 4 {
				return 0, errors.New("runner: bad exit frame")
			}
			return int(int32(binary.BigEndian.Uint32(payload))), nil
		case frameFailure:
			return 0, fmt.Errorf("runner: %s", payload)
		}
	}
}

// ClientMain is the per-command client: it reads the request from its
// environment, relays its own stdio (switching a terminal to raw mode, so the
// remote PTY does the line editing), forwards signals and window size, and
// returns the exit code to use. A command killed by a signal exits 137, and a
// command the runner refused to start exits 126 with the reason on stderr.
func ClientMain() int {
	var req Request
	if err := json.Unmarshal([]byte(os.Getenv(envRequest)), &req); err != nil {
		fmt.Fprintf(os.Stderr, "runner: bad request: %v\n", err)
		return 126
	}
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	var resizes chan [2]uint16
	fd := int(os.Stdin.Fd())
	if old, err := unix.IoctlGetTermios(fd, unix.TCGETS); err == nil {
		req.TTY = true
		if ws, err := pty.GetsizeFull(os.Stdin); err == nil {
			req.Rows, req.Cols = ws.Rows, ws.Cols
		}
		raw := *old
		raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
		raw.Oflag &^= unix.OPOST
		raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
		raw.Cflag &^= unix.CSIZE | unix.PARENB
		raw.Cflag |= unix.CS8
		raw.Cc[unix.VMIN], raw.Cc[unix.VTIME] = 1, 0
		if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err == nil {
			defer func() { _ = unix.IoctlSetTermios(fd, unix.TCSETS, old) }()
		}
		resizes = make(chan [2]uint16, 1)
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, syscall.SIGWINCH)
		go func() {
			for range winch {
				if ws, err := pty.GetsizeFull(os.Stdin); err == nil {
					select {
					case resizes <- [2]uint16{ws.Rows, ws.Cols}:
					default:
					}
				}
			}
		}()
	}
	code, err := Exec(context.Background(), os.Getenv(envSocket), req, os.Stdin, os.Stdout, os.Stderr, sigs, resizes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 126
	}
	if code < 0 {
		return 137
	}
	return code
}
