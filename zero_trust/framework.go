// Nexus IA v6.0 — Framework MAESTRO (orquestador)
//
// Une las 5 reglas (una por principio MAESTRO) en un único framework
// que evalúa CADA acción de CADA agente antes de permitirla.
//
// Flujo de una evaluación:
//
//   1. Recibir Request (acción + JWT + contexto)
//   2. Pasar por cada regla en orden
//   3. Consolidar decisiones:
//      - Quarantine > Deny > Ask > Allow
//   4. Auditar la evaluación completa
//   5. Devolver Result
//
// Es thread-safe.
package zero_trust

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Framework es el orquestador principal de Zero-Trust.
type Framework struct {
	cfg Config

	mu    sync.RWMutex
	rules []Rule

	// Rule por nombre para enable/disable dinámico
	ruleByName map[string]Rule

	// Regla de audit (se ejecuta al final, siempre)
	auditRule *AuditEverythingRule

	// Contadores
	stats FrameworkStats
}

// FrameworkStats lleva estadísticas de uso.
type FrameworkStats struct {
	mu             sync.Mutex
	TotalEvaluated int64
	TotalAllowed   int64
	TotalDenied    int64
	TotalAsk       int64
	TotalQuarantine int64
}

// NewFramework crea un framework con las 5 reglas por defecto.
//
// Parámetros:
//   - verifyJWT: verificador de JWT (desde nhi_manager)
//   - checkRevocation: verificador de revocación (desde nhi_manager)
//   - auditSink: destino del audit trail (desde nhi_manager)
func NewFramework(
	cfg Config,
	verifyJWT JWTVerifier,
	checkRevocation func(string) bool,
	auditSink AuditSink,
) *Framework {
	f := &Framework{
		cfg:        cfg,
		ruleByName: make(map[string]Rule),
	}

	// Regla 1: Mínimo privilegio
	f.AddRule(NewMinimalPrivilegeRule())

	// Regla 2: Never trust, always verify
	nt := NewNeverTrustRule(verifyJWT)
	nt.CheckRevocation = checkRevocation
	f.AddRule(nt)

	// Regla 3: Micro-segmentación
	f.AddRule(NewMicroSegmentationRule())

	// Regla 4: Assume breach
	f.AddRule(NewAssumeBreachRule())

	// Regla 5: Audit everything (se ejecuta al final)
	if auditSink != nil {
		f.auditRule = NewAuditEverythingRule(auditSink)
		f.AddRule(f.auditRule)
	}

	return f
}

// AddRule añade una regla al framework.
func (f *Framework) AddRule(r Rule) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rules = append(f.rules, r)
	f.ruleByName[r.Name()] = r
}

// Rules devuelve las reglas activas.
func (f *Framework) Rules() []Rule {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]Rule(nil), f.rules...)
}

// Rule devuelve una regla por nombre.
func (f *Framework) Rule(name string) Rule {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.ruleByName[name]
}

// Evaluate evalúa una acción y devuelve el resultado consolidado.
func (f *Framework) Evaluate(ctx context.Context, req *Request) (*Result, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}

	f.mu.RLock()
	rules := append([]Rule(nil), f.rules...)
	f.mu.RUnlock()

	// Timestamp por defecto
	if req.Action.Timestamp.IsZero() {
		req.Action.Timestamp = time.Now()
	}

	// Evaluar todas las reglas
	decisions := make([]DecisionRecord, 0, len(rules))
	finalDecision := Allow
	finalReason := "all rules passed"
	finalPrinciple := Principle("")
	var decisionTTL time.Duration
	requiresApproval := false

	for _, rule := range rules {
		// Audit es especial: nunca deniega, solo observa
		if rule.Principle() == PrincipleAuditEverything {
			continue
		}

		result := rule.Evaluate(req)
		decisions = append(decisions, DecisionRecord{
			Rule:      rule.Name(),
			Principle: rule.Principle(),
			Decision:  result.Decision,
			Reason:    result.Reason,
		})

		// Consolidar (worst-case)
		if result.Decision > finalDecision {
			finalDecision = result.Decision
			finalReason = result.Reason
			finalPrinciple = result.Principle
		}

		if result.TTL > 0 && (decisionTTL == 0 || result.TTL < decisionTTL) {
			decisionTTL = result.TTL
		}
		if result.RequiresHumanApproval {
			requiresApproval = true
		}
	}

	// TTL por defecto si ninguna regla lo puso
	if decisionTTL == 0 {
		decisionTTL = f.cfg.DefaultDecisionTTL
	}

	// Si el riesgo excede el máximo sin aprobación, forzar Ask
	if f.cfg.MaxRiskWithoutApproval != "" && req.Risk != "" {
		if riskExceeds(req.Risk, f.cfg.MaxRiskWithoutApproval) {
			requiresApproval = true
			if finalDecision < Ask {
				finalDecision = Ask
				finalReason = "risk level " + string(req.Risk) + " requires human approval"
			}
		}
	}

	finalResult := &Result{
		Decision:              finalDecision,
		Principle:             finalPrinciple,
		Reason:                finalReason,
		TTL:                   decisionTTL,
		RequiresHumanApproval: requiresApproval,
		TraceID:               req.Action.TraceID,
		Timestamp:             req.Action.Timestamp,
	}

	// Actualizar stats
	f.updateStats(finalDecision)

	// Auditar (siempre, si el audit rule está configurada)
	if f.auditRule != nil {
		_ = f.auditRule.RecordEvaluation(ctx, req, finalResult, decisions)
	}

	return finalResult, nil
}

// updateStats actualiza los contadores.
func (f *Framework) updateStats(d Decision) {
	f.stats.mu.Lock()
	defer f.stats.mu.Unlock()

	f.stats.TotalEvaluated++
	switch d {
	case Allow:
		f.stats.TotalAllowed++
	case Deny:
		f.stats.TotalDenied++
	case Ask:
		f.stats.TotalAsk++
	case Quarantine:
		f.stats.TotalQuarantine++
	}
}

// Stats devuelve las estadísticas de uso.
func (f *Framework) Stats() FrameworkStats {
	f.stats.mu.Lock()
	defer f.stats.mu.Unlock()
	// Devolvemos una copia solo de los contadores: copiar el struct completo
	// duplicaría el sync.Mutex incluido en FrameworkStats (aviso de go vet).
	return FrameworkStats{
		TotalEvaluated:  f.stats.TotalEvaluated,
		TotalAllowed:    f.stats.TotalAllowed,
		TotalDenied:     f.stats.TotalDenied,
		TotalAsk:        f.stats.TotalAsk,
		TotalQuarantine: f.stats.TotalQuarantine,
	}
}

// Report devuelve un reporte legible del framework.
func (f *Framework) Report() string {
	stats := f.Stats()
	f.mu.RLock()
	numRules := len(f.rules)
	f.mu.RUnlock()

	return fmt.Sprintf(
		"Nexus IA Zero-Trust Framework (MAESTRO)\n"+
			"  Reglas activas:      %d\n"+
			"  Total evaluadas:     %d\n"+
			"  Allow:               %d\n"+
			"  Deny:                %d\n"+
			"  Ask:                 %d\n"+
			"  Quarantine:          %d",
		numRules,
		stats.TotalEvaluated,
		stats.TotalAllowed,
		stats.TotalDenied,
		stats.TotalAsk,
		stats.TotalQuarantine,
	)
}

// riskExceeds devuelve true si actual excede max.
func riskExceeds(actual, max RiskLevel) bool {
	order := map[RiskLevel]int{
		RiskLow:      0,
		RiskMedium:   1,
		RiskHigh:     2,
		RiskCritical: 3,
	}
	return order[actual] > order[max]
}
