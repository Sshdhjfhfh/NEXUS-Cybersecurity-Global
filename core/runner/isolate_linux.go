//go:build linux

package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// InitSubcommand is argv[1] when the runner binary re-execs itself as the
// in-namespace init (see InitMain). The server inserts it in front of the
// agent command when an isolator is active.
const InitSubcommand = "runner-init"

// MountIsolate returns an isolator that runs each command in a fresh PID, mount
// and user namespace. Inside, /proc is remounted so it reflects only the new PID
// namespace — the command cannot read /proc/<pid>/environ of the orchestrator or
// of other tasks, and cannot see their processes at all. Each path in mask is
// hidden behind an empty, unreadable mount (a tmpfs over a directory, /dev/null
// over a file), so secrets like /app/keys stay invisible even when the runner
// shares a filesystem with the orchestrator in development. no_new_privs is set.
//
// self must be the path of the runner binary; the isolator re-execs it as
// InitSubcommand to perform the mounts, because Go cannot run code between clone
// and exec. In production the strong boundary is the runner's own container (no
// secret volumes, no database route); this namespace is defence in depth and the
// only boundary in a shared-filesystem dev run.
func MountIsolate(self string, mask []string) func(cmd *exec.Cmd, root string) error {
	return func(cmd *exec.Cmd, root string) error {
		argv := append([]string{self, InitSubcommand, root, strconv.Itoa(len(mask))}, mask...)
		argv = append(argv, "--")
		argv = append(argv, cmd.Args...) // cmd.Args[0] is /bin/sh
		cmd.Path = self
		cmd.Args = argv
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER
		cmd.SysProcAttr.Unshareflags |= syscall.CLONE_NEWNS
		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}}
		cmd.SysProcAttr.GidMappingsEnableSetgroups = false
		return nil
	}
}

// InitMain is the re-exec'd init that runs inside the new namespaces: it makes
// mount propagation private, mounts a fresh /proc for the new PID namespace,
// masks the requested paths, sets no_new_privs, and execs the agent command. It
// returns a process exit code only on failure; on success it never returns.
//
// args is os.Args[2:] — the layout the isolator built:
//
//	<root> <nmask> <mask...> -- <argv...>
func InitMain(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "runner-init: too few arguments")
		return 126
	}
	n, err := strconv.Atoi(args[1])
	if err != nil || n < 0 || 2+n >= len(args) {
		fmt.Fprintln(os.Stderr, "runner-init: bad mask count")
		return 126
	}
	mask := args[2 : 2+n]
	rest := args[2+n:]
	if len(rest) < 2 || rest[0] != "--" {
		fmt.Fprintln(os.Stderr, "runner-init: missing command")
		return 126
	}
	argv := rest[1:]

	// Stop our mounts from propagating back to the host mount namespace.
	if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
		fmt.Fprintf(os.Stderr, "runner-init: make private: %v\n", err)
		return 126
	}
	// A /proc that reflects the new PID namespace, replacing the host's.
	if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Fprintf(os.Stderr, "runner-init: mount proc: %v\n", err)
		return 126
	}
	for _, p := range mask {
		fi, err := os.Stat(p)
		if err != nil {
			continue // nothing to hide
		}
		if fi.IsDir() {
			_ = unix.Mount("tmpfs", p, "tmpfs", unix.MS_RDONLY, "mode=0000")
		} else {
			_ = unix.Mount("/dev/null", p, "", unix.MS_BIND, "")
		}
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "runner-init: no_new_privs: %v\n", err)
		return 126
	}
	if err := unix.Exec(argv[0], argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "runner-init: exec %s: %v\n", argv[0], err)
		return 127
	}
	return 0 // unreachable
}
