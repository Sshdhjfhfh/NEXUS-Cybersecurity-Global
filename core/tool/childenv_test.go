package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSensitiveEnvKey(t *testing.T) {
	for _, k := range []string{
		"NEXUS_PG_DSN", "NEXUS_SMTP_PASS", "nexus_key_dir", "PGPASSWORD", "PGHOST", "POSTGRES_PASSWORD",
		"DATABASE_URL", "SMTP_USER", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GITHUB_TOKEN",
		"AWS_SECRET_ACCESS_KEY", "AWS_ACCESS_KEY_ID", "JWT_SECRET", "MY_DB_PASSWORD", "ARTEX_PG_DSN",
		"SSH_PRIVATE_KEY", "GOOGLE_APPLICATION_CREDENTIALS",
	} {
		if !SensitiveEnvKey(k) {
			t.Errorf("%s must be scrubbed", k)
		}
	}
	for _, k := range []string{
		"PATH", "HOME", "USER", "LANG", "TERM", "PWD", "HTTP_PROXY", "SSL_CERT_FILE",
		"NODE_EXTRA_CA_CERTS", "TMPDIR", "SHELL", "GOPATH", "SSH_AUTH_SOCK",
	} {
		if SensitiveEnvKey(k) {
			t.Errorf("%s must be kept", k)
		}
	}
}

// Agent commands never see orchestrator secrets in their environment, while the
// explicit per-session entries (proxy, CA) still reach them.
func TestBashDoesNotInheritSecrets(t *testing.T) {
	t.Setenv("NEXUS_PG_DSN", "postgres://nexus:hunter2@db/nexus")
	t.Setenv("NEXUS_SMTP_PASS", "smtp-secret")
	t.Setenv("OPENAI_API_KEY", "sk-test-not-real")
	t.Setenv("NEXUS_CHILDENV_PLAIN", "x")
	t.Setenv("CHILDENV_VISIBLE", "visible")
	for _, tc := range []*ToolContext{nil, {}, {Env: []string{"HTTP_PROXY=http://127.0.0.1:8080"}}} {
		raw, _ := json.Marshal(map[string]any{"command": "env"})
		res, err := NewBash().Call(context.Background(), raw, tc)
		if err != nil {
			t.Fatalf("bash: %v", err)
		}
		out := res.Flatten()
		for _, leak := range []string{"hunter2", "smtp-secret", "sk-test-not-real", "NEXUS_"} {
			if strings.Contains(out, leak) {
				t.Errorf("env leaked %q to agent command (tc=%+v)", leak, tc)
			}
		}
		if !strings.Contains(out, "CHILDENV_VISIBLE=visible") {
			t.Errorf("non-sensitive variable dropped (tc=%+v)", tc)
		}
		if tc != nil && len(tc.Env) > 0 && !strings.Contains(out, "HTTP_PROXY=http://127.0.0.1:8080") {
			t.Errorf("explicit entry dropped: %q", out)
		}
	}
}

func TestChildEnvKeepsExplicitSecrets(t *testing.T) {
	t.Setenv("NEXUS_PG_DSN", "inherited")
	env := ChildEnv([]string{"MCP_API_TOKEN=given"})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "NEXUS_PG_DSN") || !strings.Contains(joined, "MCP_API_TOKEN=given") {
		t.Fatalf("explicit entries must be kept and inherited secrets dropped: %v", env)
	}
}
