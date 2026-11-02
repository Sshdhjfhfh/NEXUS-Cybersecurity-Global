// Package runner executes agent shell commands in a separate, isolated
// tool-runner process (docs/security/tool-runner.md). The orchestrator side is a
// tiny client process per command that relays stdio, terminal size, signals and
// the exit status over a Unix socket, so the callers in core/tool keep driving
// an ordinary *exec.Cmd.
package runner

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Version is the protocol version a client sends; the server refuses others.
const Version = 1

// Request is the first message on a connection: one JSON object, then frames.
type Request struct {
	V       int      `json:"v"`
	Command string   `json:"cmd"`            // run as `sh -c Command`
	Dir     string   `json:"dir"`            // absolute; must lie inside a task or session root
	Env     []string `json:"env,omitempty"`  // explicit KEY=VALUE entries (proxy, CA); nothing inherited
	TTY     bool     `json:"tty,omitempty"`  // run under a pseudo-terminal
	Rows    uint16   `json:"rows,omitempty"` // initial terminal size when TTY
	Cols    uint16   `json:"cols,omitempty"`
}

// Frame types. Client → server: stdin, closeStdin, resize, signal.
// Server → client: stdout, stderr, exit, failure.
const (
	frameStdin      byte = 1
	frameCloseStdin byte = 2
	frameResize     byte = 3 // rows uint16, cols uint16
	frameSignal     byte = 4 // signal number uint8
	frameStdout     byte = 10
	frameStderr     byte = 11
	frameExit       byte = 12 // exit status int32 (-1 = killed by a signal)
	frameFailure    byte = 13 // UTF-8 reason the command could not start
)

// maxFrame bounds one frame's payload so a peer cannot make the other side
// allocate without limit.
const maxFrame = 1 << 20

var errFrameTooLarge = errors.New("runner: frame too large")

func writeFrame(w io.Writer, typ byte, payload []byte) error {
	if len(payload) > maxFrame {
		return errFrameTooLarge
	}
	var hdr [5]byte
	hdr[0] = typ
	binary.BigEndian.PutUint32(hdr[1:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readFrame(r io.Reader) (byte, []byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	n := binary.BigEndian.Uint32(hdr[1:])
	if n > maxFrame {
		return 0, nil, errFrameTooLarge
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, nil, err
	}
	return hdr[0], buf, nil
}

// writeRequest sends the request as a length-prefixed frame of type 0.
func writeRequest(w io.Writer, req Request) error {
	raw, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return writeFrame(w, 0, raw)
}

func readRequest(r io.Reader) (Request, error) {
	typ, raw, err := readFrame(r)
	if err != nil {
		return Request{}, err
	}
	if typ != 0 {
		return Request{}, fmt.Errorf("runner: expected request, got frame %d", typ)
	}
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return Request{}, fmt.Errorf("runner: bad request: %w", err)
	}
	if req.V != Version {
		return Request{}, fmt.Errorf("runner: protocol version %d, want %d", req.V, Version)
	}
	return req, nil
}

// frameWriter serializes frames from several goroutines onto one connection.
type frameWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newFrameWriter(w io.Writer) *frameWriter { return &frameWriter{w: w} }

func (fw *frameWriter) send(typ byte, payload []byte) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return writeFrame(fw.w, typ, payload)
}

// streamWriter turns writes into frames of one type, split to maxFrame.
type streamWriter struct {
	fw  *frameWriter
	typ byte
}

func (s streamWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		n := min(len(p), maxFrame)
		if err := s.fw.send(s.typ, p[:n]); err != nil {
			return written, err
		}
		written += n
		p = p[n:]
	}
	return written, nil
}
