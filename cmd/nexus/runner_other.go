//go:build !linux

package main

// The tool runner is Linux-only (it relies on PID/mount/user namespaces). On
// other platforms these are no-ops, so a build still succeeds and agent commands
// run locally.

func maybeRunnerSubcommand() {}

func wireLauncher() {}
