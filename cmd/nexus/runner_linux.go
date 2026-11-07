//go:build linux

package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/nexus-ia/nexus/core/runner"
	"github.com/nexus-ia/nexus/core/tool"
)

// maybeRunnerSubcommand handles the re-exec subcommands of the runner before any
// flag parsing, and exits the process when it handles one:
//
//	nexus runner-init …   in-namespace init (mounts, then execs the command)
//	nexus runner-exec      per-command client (speaks to a runner over its socket)
//	nexus runner-serve     the runner server (own container, no secrets, no DB)
func maybeRunnerSubcommand() {
	if len(os.Args) < 2 {
		return
	}
	switch os.Args[1] {
	case runner.InitSubcommand:
		os.Exit(runner.InitMain(os.Args[2:]))
	case runner.ClientSubcommand:
		os.Exit(runner.ClientMain())
	case "runner-serve":
		os.Exit(runnerServe())
	}
}

// wireLauncher routes agent shell commands through the runner when
// NEXUS_RUNNER_SOCKET is set. Without it, commands run locally (development).
func wireLauncher() {
	sock := strings.TrimSpace(os.Getenv("NEXUS_RUNNER_SOCKET"))
	if sock == "" {
		return
	}
	tool.SetLauncher(runner.Launcher(sock))
	log.Printf("[runner] agent commands routed through runner at %s", sock)
}

// runnerServe runs the runner server from its environment. It is meant to be the
// entrypoint of the tool-runner container, which holds no keys and cannot reach
// the database; the namespace isolation here is defence in depth.
func runnerServe() int {
	sock := strings.TrimSpace(os.Getenv("NEXUS_RUNNER_SOCKET"))
	base := strings.TrimSpace(os.Getenv("NEXUS_RUNNER_WORKBASE"))
	if sock == "" || base == "" {
		log.Printf("runner-serve: NEXUS_RUNNER_SOCKET and NEXUS_RUNNER_WORKBASE are required")
		return 2
	}
	peer := -1
	if v := strings.TrimSpace(os.Getenv("NEXUS_RUNNER_PEER_UID")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			peer = n
		}
	}
	var mask []string
	if v := strings.TrimSpace(os.Getenv("NEXUS_RUNNER_MASK")); v != "" {
		mask = strings.Split(v, ":")
	}
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	_ = os.Remove(sock) // a stale socket from a previous run would block Listen
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: sock, Net: "unix"})
	if err != nil {
		log.Printf("runner-serve: listen %s: %v", sock, err)
		return 1
	}
	srv := &runner.Server{
		WorkBase: base,
		PeerUID:  peer,
		BaseEnv:  []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8"},
		Isolate:  runner.MountIsolate(self, mask),
	}
	log.Printf("[runner] serving on %s (workbase=%s, peer_uid=%d, masked=%d)", sock, base, peer, len(mask))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := srv.Serve(ctx, l); err != nil {
		log.Printf("runner-serve: %v", err)
		return 1
	}
	return 0
}
