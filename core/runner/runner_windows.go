//go:build windows

package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ClientSubcommand is the argument that makes the orchestrator binary act as
// the per-command client. The runner needs Unix sockets and PTYs; on Windows
// the client only reports that.
const ClientSubcommand = "runner-exec"

// Launcher is not available on Windows; commands report the error and exit 126.
func Launcher(socket string) func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
	return func(ctx context.Context, command, dir string, env []string) *exec.Cmd {
		self, _ := os.Executable()
		return exec.CommandContext(ctx, self, ClientSubcommand)
	}
}

// ClientMain reports that the runner is unsupported on Windows.
func ClientMain() int {
	fmt.Fprintln(os.Stderr, "runner: not supported on Windows")
	return 126
}
