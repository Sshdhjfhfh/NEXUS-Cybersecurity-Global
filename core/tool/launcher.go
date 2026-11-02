package tool

import (
	"context"
	"os/exec"
	"sync/atomic"
)

// Launcher builds the process for a shell command an agent runs: Bash in the
// foreground and background and interactive shell sessions all start their
// command through it. dir is the agent's working directory and env the explicit
// extra "KEY=VALUE" entries; the returned command must not have been started.
//
// The callers keep driving the *exec.Cmd (pipes or a PTY, process-group kill,
// Wait), so a launcher that runs the command elsewhere returns a local client
// process that relays stdio, signals and the exit status, as `docker exec` does.
type Launcher func(ctx context.Context, command, dir string, env []string) *exec.Cmd

var launcher atomic.Pointer[Launcher]

// SetLauncher replaces the launcher for every later command; nil restores
// LocalLauncher. Hosts call it once at startup.
func SetLauncher(l Launcher) {
	if l == nil {
		launcher.Store(nil)
		return
	}
	launcher.Store(&l)
}

// LocalLauncher runs the command in this process's own environment, with
// orchestrator secrets removed (ChildEnv). It is not an isolation boundary.
func LocalLauncher(ctx context.Context, command, dir string, env []string) *exec.Cmd {
	shell, flags := shellCmd()
	cmd := exec.CommandContext(ctx, shell, append(flags, command)...)
	cmd.Dir = dir
	cmd.Env = withEnv(env)
	return cmd
}

func launch(ctx context.Context, command, dir string, env []string) *exec.Cmd {
	if l := launcher.Load(); l != nil {
		return (*l)(ctx, command, dir, env)
	}
	return LocalLauncher(ctx, command, dir, env)
}
