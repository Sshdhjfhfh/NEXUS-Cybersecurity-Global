package tool

import (
	"os"
	"strings"
)

// ChildEnv returns the environment for a command run on an agent's behalf: the
// process environment with orchestrator secrets removed (database DSNs and
// passwords, SMTP credentials, NEXUS_* configuration, API keys and tokens),
// followed by the explicit extra "KEY=VALUE" entries, which are always kept.
//
// This keeps secrets out of `env`, `printenv` and command output. It is NOT an
// isolation boundary: a command running in the orchestrator's container can
// still read /proc/<ppid>/environ, mounted key files and reach the database
// network. Real isolation needs a separate tool-runner (see docs/roadmap.md).
func ChildEnv(extra []string) []string {
	parent := os.Environ()
	env := make([]string, 0, len(parent)+len(extra))
	for _, kv := range parent {
		k, _, _ := strings.Cut(kv, "=")
		if !SensitiveEnvKey(k) {
			env = append(env, kv)
		}
	}
	return append(env, extra...)
}

var (
	sensitiveEnvPrefixes = []string{
		"NEXUS_", "ARTEX_", "POSTGRES", "PG", "DATABASE_", "SMTP", "AWS_", "AZURE_",
	}
	sensitiveEnvSuffixes = []string{"_KEY", "_PASS", "_DSN", "_KEY_ID"}
	sensitiveEnvWords    = []string{"SECRET", "PASSWORD", "PASSWD", "TOKEN", "CREDENTIAL", "API_KEY", "PRIVATE"}
)

// SensitiveEnvKey reports whether an inherited variable must not reach a
// command spawned for an agent. Matching is case-insensitive (Windows keys).
func SensitiveEnvKey(key string) bool {
	k := strings.ToUpper(key)
	for _, p := range sensitiveEnvPrefixes {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	for _, s := range sensitiveEnvSuffixes {
		if strings.HasSuffix(k, s) {
			return true
		}
	}
	for _, w := range sensitiveEnvWords {
		if strings.Contains(k, w) {
			return true
		}
	}
	return false
}
