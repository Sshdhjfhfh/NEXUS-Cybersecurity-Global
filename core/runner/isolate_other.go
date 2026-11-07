//go:build !linux

package runner

import (
	"errors"
	"os/exec"
)

// InitSubcommand exists on every platform so callers can dispatch on it, but the
// in-namespace init only runs on Linux.
const InitSubcommand = "runner-init"

// MountIsolate is Linux-only. On other platforms it returns an isolator that
// refuses to run, so a misconfiguration fails closed instead of running an
// agent command unconfined.
func MountIsolate(self string, mask []string) func(cmd *exec.Cmd, root string) error {
	return func(cmd *exec.Cmd, root string) error {
		return errors.New("runner: namespace isolation is only supported on Linux")
	}
}

// InitMain is never reached off Linux; it reports the error for completeness.
func InitMain(args []string) int {
	return 126
}
