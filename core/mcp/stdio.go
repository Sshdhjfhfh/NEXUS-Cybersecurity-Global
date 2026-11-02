// Nexus IA v6.0 — Absorbido de norma/mcp
// Upstream: github.com/Autumn-27/norma/mcp
// Licencia: AGPL-3.0

package mcp

import (
	"context"
	"io"
	"os/exec"

	"github.com/nexus-ia/nexus/core/tool"
)

// stdioCloser shuts down a spawned MCP server process and its pipes.
type stdioCloser struct {
	cmd   *exec.Cmd
	stdin io.Closer
}

func (s *stdioCloser) Close() error {
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	return nil
}

// NewStdioClient spawns an MCP server as a subprocess, performs the handshake,
// and returns a connected Client (FR-10.1). Call Close to terminate it.
//
// env holds extra environment variables (e.g. API tokens an MCP server needs)
// applied on top of the parent process environment minus orchestrator secrets
// (tool.ChildEnv); pass nil for none. Secrets a server needs must be listed here.
func NewStdioClient(ctx context.Context, server, command string, env map[string]string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...)
	extra := make([]string, 0, len(env))
	for k, v := range env {
		extra = append(extra, k+"="+v)
	}
	cmd.Env = tool.ChildEnv(extra)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &Client{
		server: server,
		conn:   newConn(stdout, stdin),
		closer: &stdioCloser{cmd: cmd, stdin: stdin},
	}
	if err := c.initialize(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// newClientWithConn builds a Client over an arbitrary transport (used by tests).
func newClientWithConn(ctx context.Context, server string, r io.Reader, w io.Writer, closer io.Closer) (*Client, error) {
	c := &Client{server: server, conn: newConn(r, w), closer: closer}
	if err := c.initialize(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}
