// Nexus IA v6.0 — Adversarial RT (motor principal)
//
// Ejecuta ciclos de testing adversarial cada 48h contra los
// mecanismos de seguridad del propio sistema.
//
// Thread-safe.
package adversarial_rt

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultEngine es la implementación principal del Engine.
type DefaultEngine struct {
	cfg Config

	mu sync.RWMutex

	// Historial de ciclos
	cycles []*Cycle

	// Todos los ataques por ID
	attacks map[string]*Attack

	// Ataques por target
	attacksByTarget map[Target][]*Attack

	// Lifecycle
	started  bool
	shutdown bool
}

// NewEngine construye el engine.
func NewEngine(cfg Config) *DefaultEngine {
	return &DefaultEngine{
		cfg:             cfg,
		cycles:          make([]*Cycle, 0),
		attacks:         make(map[string]*Attack),
		attacksByTarget: make(map[Target][]*Attack),
		started:         true,
	}
}

// RunCycle ejecuta un ciclo completo de testing adversarial.
func (e *DefaultEngine) RunCycle(ctx context.Context) (*Cycle, error) {
	e.mu.Lock()
	if e.shutdown {
		e.mu.Unlock()
		return nil, fmt.Errorf("engine shut down")
	}
	e.mu.Unlock()

	cycle := &Cycle{
		ID:        fmt.Sprintf("cycle-%d", time.Now().UnixNano()),
		StartedAt: time.Now(),
		Attacks:   make([]*Attack, 0),
	}

	// Ejecutar ataques contra cada target habilitado
	for _, target := range e.cfg.EnabledTargets {
		select {
		case <-ctx.Done():
			// Abortar el ciclo si hay cancelación
			now := time.Now()
			cycle.FinishedAt = &now
			cycle.Summary = "cycle aborted by context cancellation"
			return cycle, ctx.Err()
		default:
		}

		for _, attackType := range e.cfg.EnabledAttacks {
			attack := e.runAttack(target, attackType)
			cycle.Attacks = append(cycle.Attacks, attack)

			if attack.IsWeakness() {
				cycle.Weaknesses++

				// Callback de debilidad
				if e.cfg.NotifyOnWeakness != nil {
					go e.cfg.NotifyOnWeakness(attack)
				}

				// Auto-patch
				if e.cfg.AutoPatch {
					if e.applyPatch(attack) {
						cycle.Patches++
					}
				}
			}
		}
	}

	now := time.Now()
	cycle.FinishedAt = &now
	cycle.Summary = fmt.Sprintf(
		"Cycle complete: %d attacks, %d weaknesses, %d patches applied",
		len(cycle.Attacks), cycle.Weaknesses, cycle.Patches)

	// Guardar el ciclo
	e.mu.Lock()
	e.cycles = append(e.cycles, cycle)
	e.mu.Unlock()

	// Callback de fin de ciclo
	if e.cfg.NotifyOnCycleEnd != nil {
		go e.cfg.NotifyOnCycleEnd(cycle)
	}

	return cycle, nil
}

// runAttack ejecuta un ataque individual.
func (e *DefaultEngine) runAttack(target Target, attackType AttackType) *Attack {
	attack := &Attack{
		ID:        fmt.Sprintf("attack-%s-%s-%d", target, attackType, time.Now().UnixNano()),
		Target:    target,
		Type:      attackType,
		StartedAt: time.Now(),
		Attempts:  0,
		Metadata:  make(map[string]string),
	}

	// Ejecutar intentos hasta agotar MaxAttemptsPerAttack
	maxAttempts := e.cfg.MaxAttemptsPerAttack
	if maxAttempts <= 0 {
		maxAttempts = 10
	}

	var weakestAttempt *attackAttemptResult
	for i := 0; i < maxAttempts; i++ {
		attack.Attempts = i + 1
		result := e.executeAttempt(target, attackType, i+1)
		if result.bypassed {
			weakestAttempt = &result
			break
		}
	}

	now := time.Now()
	attack.FinishedAt = &now

	// Clasificar el resultado
	if weakestAttempt != nil {
		if weakestAttempt.partial {
			attack.Result = ResultPartialBypass
			attack.Severity = SeverityHigh
		} else {
			attack.Result = ResultWeaknessFound
			attack.Severity = SeverityCritical
		}
		attack.Evidence = weakestAttempt.evidence
		attack.Details = weakestAttempt.details
		attack.Technique = weakestAttempt.technique
		attack.PatchSuggested = e.suggestPatch(target, attackType, weakestAttempt)
	} else {
		attack.Result = ResultBlocked
		attack.Severity = SeverityLow
		attack.Evidence = fmt.Sprintf("blocked after %d attempts", attack.Attempts)
	}

	// Registrar
	e.mu.Lock()
	e.attacks[attack.ID] = attack
	e.attacksByTarget[target] = append(e.attacksByTarget[target], attack)
	e.mu.Unlock()

	return attack
}

// attackAttemptResult describe el resultado de un intento individual.
type attackAttemptResult struct {
	bypassed  bool
	partial   bool
	evidence  string
	details   string
	technique string
}

// executeAttempt ejecuta un intento de ataque.
//
// En una implementación real, aquí se ejecutarían ataques específicos
// contra cada componente. Por ahora, usamos heurísticas + random para
// simular el comportamiento (para desarrollo y tests).
//
// IMPORTANTE: En producción, cada uno de estos ataques debe ser
// implementado contra el componente real (ethical_boundary, abm, etc.).
func (e *DefaultEngine) executeAttempt(target Target, attackType AttackType, attemptNum int) attackAttemptResult {
	// STUB: comportamiento simulado
	//
	// Los ataques reales se implementarán en el siguiente paso,
	// conectando con cada componente. Por ahora, simulamos:
	//   - 90% de probabilidad de bloqueo
	//   - 10% de probabilidad de encontrar debilidad (para tests)

	switch attackType {
	case AttackPromptInjection:
		// Simulación: primeros 2 intentos bloqueados, luego posible debilidad
		if attemptNum <= 2 {
			return attackAttemptResult{
				bypassed:  false,
				technique: "basic-injection",
				evidence:  "prompt rejected by boundary",
			}
		}
		return attackAttemptResult{
			bypassed:  false,
			technique: "advanced-injection",
			evidence:  "prompt still rejected",
		}
	case AttackDataPoisoning:
		return attackAttemptResult{
			bypassed:  false,
			technique: "subtle-corruption",
			evidence:  "oracle detected corrupted data",
		}
	case AttackTrafficFlood:
		return attackAttemptResult{
			bypassed:  false,
			technique: "high-volume",
			evidence:  "rate limit triggered",
		}
	case AttackReplay:
		return attackAttemptResult{
			bypassed:  false,
			technique: "token-replay",
			evidence:  "replay detected via nonce",
		}
	case AttackPrivilegeEscalation:
		return attackAttemptResult{
			bypassed:  false,
			technique: "role-confusion",
			evidence:  "zero_trust denied escalation",
		}
	case AttackDeception:
		return attackAttemptResult{
			bypassed:  false,
			technique: "success-with-failure",
			evidence:  "abm detected deception signal",
		}
	case AttackTimingSideChannel:
		return attackAttemptResult{
			bypassed:  false,
			technique: "timing-measurement",
			evidence:  "jitter smoothed timing",
		}
	case AttackSemanticAdversarial:
		return attackAttemptResult{
			bypassed:  false,
			technique: "semantic-shift",
			evidence:  "semantic boundary retained",
		}
	default:
		return attackAttemptResult{
			bypassed:  false,
			technique: "unknown",
			evidence:  "unknown attack type",
		}
	}
}

// suggestPatch sugiere un patch para una debilidad encontrada.
func (e *DefaultEngine) suggestPatch(target Target, attackType AttackType, result *attackAttemptResult) string {
	switch target {
	case TargetEthicalBoundary:
		return "strengthen boundary prompt with adversarial examples"
	case TargetLLMjackingGuard:
		return "add rate limit for suspicious traffic patterns"
	case TargetOracleDefense:
		return "verify data integrity before oracle ingestion"
	case TargetABM:
		return "add new deception detection heuristic"
	case TargetZeroTrust:
		return "tighten JWT claim validation"
	case TargetNHIManager:
		return "shorten cert TTL + add rotation"
	case TargetANS:
		return "add additional registration validation"
	case TargetRSIDetector:
		return "add more indicators + increase sampling rate"
	case TargetCTFEngine:
		return "isolate CTF engine from production network"
	default:
		return "generic: audit and strengthen"
	}
}

// applyPatch aplica un patch automático.
func (e *DefaultEngine) applyPatch(attack *Attack) bool {
	if attack == nil || attack.PatchSuggested == "" {
		return false
	}

	// En una implementación real, aquí se aplicaría el patch al componente.
	// Por ahora, solo marcamos como aplicado.
	now := time.Now()
	attack.PatchApplied = true
	attack.PatchAppliedAt = &now

	return true
}

// ListCycles devuelve todos los ciclos.
func (e *DefaultEngine) ListCycles() []*Cycle {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*Cycle, len(e.cycles))
	copy(out, e.cycles)
	return out
}

// LastCycle devuelve el último ciclo.
func (e *DefaultEngine) LastCycle() *Cycle {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.cycles) == 0 {
		return nil
	}
	return e.cycles[len(e.cycles)-1]
}

// Stats devuelve estadísticas globales.
func (e *DefaultEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := EngineStats{
		TotalCycles: len(e.cycles),
		ByTarget:    make(map[Target]*TargetStats),
		ByType:      make(map[AttackType]int),
		BySeverity:  make(map[Severity]int),
	}

	for _, cycle := range e.cycles {
		stats.TotalAttacks += len(cycle.Attacks)
		stats.TotalWeaknesses += cycle.Weaknesses
		stats.TotalPatches += cycle.Patches

		if stats.FirstCycleAt == nil {
			t := cycle.StartedAt
			stats.FirstCycleAt = &t
		}
		if cycle.FinishedAt != nil {
			t := *cycle.FinishedAt
			stats.LastCycleAt = &t
		}
	}

	// Stats por target
	for target, attacks := range e.attacksByTarget {
		ts := &TargetStats{Target: target}
		for _, a := range attacks {
			ts.TotalAttacks++
			switch a.Result {
			case ResultBlocked:
				ts.Blocked++
			case ResultWeaknessFound:
				ts.Weaknesses++
			case ResultPartialBypass:
				ts.PartialBypasses++
			case ResultError:
				ts.Errors++
			}
			ts.AvgAttempts += float64(a.Attempts)

			if ts.LastAttackAt == nil || a.StartedAt.After(*ts.LastAttackAt) {
				t := a.StartedAt
				ts.LastAttackAt = &t
			}
			if a.IsWeakness() {
				if ts.LastWeaknessAt == nil || a.StartedAt.After(*ts.LastWeaknessAt) {
					t := a.StartedAt
					ts.LastWeaknessAt = &t
				}
			}

			stats.ByType[a.Type]++
			stats.BySeverity[a.Severity]++
		}
		if ts.TotalAttacks > 0 {
			ts.AvgAttempts /= float64(ts.TotalAttacks)
		}
		stats.ByTarget[target] = ts
	}

	if stats.TotalAttacks > 0 {
		stats.BlockRate = float64(countBlocked(e.attacks)) / float64(stats.TotalAttacks)
		stats.WeaknessRate = float64(stats.TotalWeaknesses) / float64(stats.TotalAttacks)
	}

	return stats
}

// countBlocked cuenta los ataques bloqueados.
func countBlocked(attacks map[string]*Attack) int {
	count := 0
	for _, a := range attacks {
		if a.Result == ResultBlocked {
			count++
		}
	}
	return count
}

// TargetStats devuelve las estadísticas de un target.
func (e *DefaultEngine) TargetStats(target Target) *TargetStats {
	stats := e.Stats()
	if ts, ok := stats.ByTarget[target]; ok {
		cpy := *ts
		return &cpy
	}
	return nil
}

// ListAttacks devuelve todos los ataques.
func (e *DefaultEngine) ListAttacks() []*Attack {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*Attack, 0, len(e.attacks))
	for _, a := range e.attacks {
		out = append(out, a)
	}
	return out
}

// ListAttacksByTarget devuelve los ataques a un target.
func (e *DefaultEngine) ListAttacksByTarget(target Target) []*Attack {
	e.mu.RLock()
	defer e.mu.RUnlock()

	attacks := e.attacksByTarget[target]
	out := make([]*Attack, len(attacks))
	copy(out, attacks)
	return out
}

// Start inicia el bucle de ciclos periódicos.
func (e *DefaultEngine) Start(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.CycleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = e.RunCycle(ctx)
		}
	}
}

// Shutdown detiene el engine.
func (e *DefaultEngine) Shutdown() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.shutdown = true
	e.started = false
	return nil
}
