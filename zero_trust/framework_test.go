// Nexus IA v6.0 — Tests del framework Zero-Trust
package zero_trust

import (
	"context"
	"testing"
	"time"
)

// fakeSink captura las entradas de audit para verificación.
type fakeSink struct {
	records []*AuditRecord
}

func (s *fakeSink) RecordAudit(_ context.Context, entry *AuditRecord) error {
	s.records = append(s.records, entry)
	return nil
}

// fakeJWTVerifier devuelve claims fijos.
func fakeJWTVerifier(agentID, agentType string) JWTVerifier {
	return func(token string) (*VerifiedClaims, error) {
		if token == "" {
			return nil, errStr("empty token")
		}
		return &VerifiedClaims{
			AgentID:   agentID,
			AgentType: agentType,
			DID:       "did:nexus:test-" + agentID,
			ExpiresAt: time.Now().Add(time.Hour),
		}, nil
	}
}

func errStr(s string) error { return &simpleErr{s} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

// TestMinimalPrivilege verifica el principio 1.
func TestMinimalPrivilege(t *testing.T) {
	rule := NewMinimalPrivilegeRule()

	// RED puede SCAN
	res := rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "red-agent-01",
			AgentType: "RED",
			Type:      "SCAN",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Allow {
		t.Errorf("RED SCAN should be allowed, got %v: %s", res.Decision, res.Reason)
	}

	// RED NO puede ISOLATE_HOST (es de BLUE)
	res = rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "red-agent-01",
			AgentType: "RED",
			Type:      "ISOLATE_HOST",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Deny {
		t.Errorf("RED ISOLATE_HOST should be denied, got %v", res.Decision)
	}

	// BLUE puede ISOLATE_HOST
	res = rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "blue-agent-01",
			AgentType: "BLUE",
			Type:      "ISOLATE_HOST",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Allow {
		t.Errorf("BLUE ISOLATE_HOST should be allowed, got %v", res.Decision)
	}
}

// TestNeverTrust verifica el principio 2.
func TestNeverTrust(t *testing.T) {
	verify := fakeJWTVerifier("red-agent-01", "RED")
	rule := NewNeverTrustRule(verify)

	// Sin JWT → Deny
	res := rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "red-agent-01",
			AgentType: "RED",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Deny {
		t.Errorf("no JWT should be denied, got %v", res.Decision)
	}

	// Con JWT válido pero actor mismatch → Deny
	res = rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "different-agent",
			AgentType: "RED",
			Timestamp: time.Now(),
		},
		JWT: "valid-token",
	})
	if res.Decision != Deny {
		t.Errorf("actor mismatch should be denied, got %v", res.Decision)
	}

	// Con JWT válido y actor correcto → Allow
	res = rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "red-agent-01",
			AgentType: "RED",
			Timestamp: time.Now(),
		},
		JWT: "valid-token",
	})
	if res.Decision != Allow {
		t.Errorf("valid JWT should be allowed, got %v: %s", res.Decision, res.Reason)
	}
}

// TestMicroSegmentation verifica el principio 3.
func TestMicroSegmentation(t *testing.T) {
	rule := NewMicroSegmentationRule()

	// RED puede hablar con core_zone
	res := rule.Evaluate(&Request{
		Action: Action{
			AgentType: "RED",
			Target:    "core_zone:orchestrator",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Allow {
		t.Errorf("RED → core should be allowed, got %v: %s", res.Decision, res.Reason)
	}

	// RED NO puede hablar con blue_zone
	res = rule.Evaluate(&Request{
		Action: Action{
			AgentType: "RED",
			Target:    "blue_zone:blue-agent-01",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Deny {
		t.Errorf("RED → BLUE should be denied, got %v", res.Decision)
	}

	// Target sin zona → no aplica (allow)
	res = rule.Evaluate(&Request{
		Action: Action{
			AgentType: "RED",
			Target:    "external-resource",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Allow {
		t.Errorf("target without zone should be allowed (not applicable), got %v", res.Decision)
	}
}

// TestAssumeBreach verifica el principio 4.
func TestAssumeBreach(t *testing.T) {
	rule := NewAssumeBreachRule()
	rule.Registry.Register("red-agent-01")

	// Agente nuevo → Allow (score 100)
	res := rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "red-agent-01",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Allow {
		t.Errorf("new agent should be allowed, got %v", res.Decision)
	}

	// Reducir score a 15 (quarantine)
	rule.Registry.AdjustScore("red-agent-01", -85, "anomaly", "test anomaly")

	if !rule.Registry.IsQuarantined("red-agent-01") {
		t.Error("agent should be quarantined")
	}

	res = rule.Evaluate(&Request{
		Action: Action{
			ActorID:   "red-agent-01",
			Timestamp: time.Now(),
		},
	})
	if res.Decision != Quarantine {
		t.Errorf("quarantined agent should be quarantined, got %v", res.Decision)
	}
}

// TestFrameworkFullIntegration verifica el framework completo.
func TestFrameworkFullIntegration(t *testing.T) {
	verify := fakeJWTVerifier("red-agent-01", "RED")
	sink := &fakeSink{}

	fw := NewFramework(
		DefaultConfig(),
		verify,
		func(_ string) bool { return false },
		sink,
	)

	// Verificar que las 5 reglas están cargadas
	if len(fw.Rules()) != 5 {
		t.Errorf("expected 5 rules, got %d", len(fw.Rules()))
	}

	// Evaluar una acción válida
	ctx := context.Background()
	req := &Request{
		Action: Action{
			ActorID:   "red-agent-01",
			AgentType: "RED",
			Type:      "SCAN",
			Target:    "core_zone:orchestrator",
			Timestamp: time.Now(),
		},
		JWT: "valid-token",
	}

	res, err := fw.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if res.Decision != Allow {
		t.Errorf("expected Allow, got %v: %s", res.Decision, res.Reason)
	}

	// Verificar que se auditó
	if len(sink.records) != 1 {
		t.Errorf("expected 1 audit record, got %d", len(sink.records))
	}

	// Verificar stats
	stats := fw.Stats()
	if stats.TotalEvaluated != 1 {
		t.Errorf("expected 1 evaluated, got %d", stats.TotalEvaluated)
	}
	if stats.TotalAllowed != 1 {
		t.Errorf("expected 1 allowed, got %d", stats.TotalAllowed)
	}

	// Report
	report := fw.Report()
	if report == "" {
		t.Error("report should not be empty")
	}
}

// TestFrameworkDeniedByRule verifica que una denegación prevalece.
func TestFrameworkDeniedByRule(t *testing.T) {
	verify := fakeJWTVerifier("red-agent-01", "RED")
	sink := &fakeSink{}

	fw := NewFramework(DefaultConfig(), verify, func(_ string) bool { return false }, sink)

	// Acción que viola mínimo privilegio (RED no puede ISOLATE_HOST)
	req := &Request{
		Action: Action{
			ActorID:   "red-agent-01",
			AgentType: "RED",
			Type:      "ISOLATE_HOST",
			Target:    "core_zone:orchestrator",
			Timestamp: time.Now(),
		},
		JWT: "valid-token",
	}

	res, err := fw.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if res.Decision != Deny {
		t.Errorf("expected Deny (privilege), got %v: %s", res.Decision, res.Reason)
	}
}
