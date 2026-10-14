// Nexus IA v6.0 — Adversarial Robustness Tester
//
// El sistema ataca sus propios mecanismos de seguridad cada 48h:
//   - Intenta bypass de ethical_boundary con prompts adversariales
//   - Intenta confundir al llmjacking_guard con tráfico falso
//   - Intenta envenenar oracle_defense con datos sutilmente incorrectos
//   - Intenta engañar al agent_behavior_monitor
//
// Mide: ¿cuántos intentos necesita para encontrar una debilidad?
// Si encuentra debilidad → alerta crítica + patch automático.
//
// El sistema de seguridad se vuelve más robusto con el tiempo.
//
// Basado en hallazgos de AISI UK (agosto 2026).
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package adversarial_rt

import (
	"time"
)

// Target es un componente del sistema que se puede atacar.
type Target string

const (
	TargetEthicalBoundary Target = "ETHICAL_BOUNDARY"
	TargetLLMjackingGuard Target = "LLMJACKING_GUARD"
	TargetOracleDefense   Target = "ORACLE_DEFENSE"
	TargetABM             Target = "AGENT_BEHAVIOR_MONITOR"
	TargetZeroTrust       Target = "ZERO_TRUST"
	TargetNHIManager      Target = "NHI_MANAGER"
	TargetANS             Target = "AGENT_NAME_SERVICE"
	TargetRSIDetector     Target = "RSI_DETECTOR"
	TargetCTFEngine       Target = "CTF_ENGINE"
)

// AllTargets devuelve todos los targets atacables.
func AllTargets() []Target {
	return []Target{
		TargetEthicalBoundary,
		TargetLLMjackingGuard,
		TargetOracleDefense,
		TargetABM,
		TargetZeroTrust,
		TargetNHIManager,
		TargetANS,
		TargetRSIDetector,
		TargetCTFEngine,
	}
}

// AttackType clasifica el tipo de ataque.
type AttackType string

const (
	// AttackPromptInjection: prompts adversariales para bypass.
	AttackPromptInjection AttackType = "PROMPT_INJECTION"

	// AttackDataPoisoning: envenenar datos sutilmente.
	AttackDataPoisoning AttackType = "DATA_POISONING"

	// AttackTrafficFlood: confundir con tráfico falso.
	AttackTrafficFlood AttackType = "TRAFFIC_FLOOD"

	// AttackReplay: reutilizar tokens/identidades válidas.
	AttackReplay AttackType = "REPLAY"

	// AttackPrivilegeEscalation: escalar privilegios.
	AttackPrivilegeEscalation AttackType = "PRIVILEGE_ESCALATION"

	// AttackDeception: engañar al monitor de comportamiento.
	AttackDeception AttackType = "DECEPTION"

	// AttackTimingSideChannel: atacar por canal lateral de timing.
	AttackTimingSideChannel AttackType = "TIMING_SIDE_CHANNEL"

	// AttackSemanticAdversarial: textos que engañan al LLM.
	AttackSemanticAdversarial AttackType = "SEMANTIC_ADVERSARIAL"
)

// Result clasifica el resultado de un ataque.
type Result string

const (
	// ResultBlocked: el ataque fue bloqueado correctamente.
	ResultBlocked Result = "BLOCKED"

	// ResultWeaknessFound: el ataque encontró una debilidad.
	ResultWeaknessFound Result = "WEAKNESS_FOUND"

	// ResultPartialBypass: el ataque parcialmente funcionó.
	ResultPartialBypass Result = "PARTIAL_BYPASS"

	// ResultError: el ataque falló por error técnico.
	ResultError Result = "ERROR"
)

// Severity clasifica la gravedad de una debilidad encontrada.
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// Attack es un ataque ejecutado contra un target.
type Attack struct {
	// Identificación
	ID         string     // UUID del ataque
	Target     Target     // Componente atacado
	Type       AttackType // Tipo de ataque
	Payload    string     // Carga del ataque
	Technique  string     // Técnica específica

	// Ejecución
	StartedAt  time.Time
	FinishedAt *time.Time
	Attempts   int        // Número de intentos

	// Resultado
	Result     Result
	Severity   Severity
	Evidence   string     // Evidencia del resultado
	Details    string     // Detalles (JSON, logs, etc.)

	// Mitigación
	PatchSuggested string
	PatchApplied   bool
	PatchAppliedAt *time.Time

	// Metadatos
	Metadata   map[string]string
}

// Duration devuelve cuánto duró el ataque.
func (a *Attack) Duration() time.Duration {
	if a.FinishedAt == nil {
		return 0
	}
	return a.FinishedAt.Sub(a.StartedAt)
}

// IsWeakness devuelve true si el ataque encontró una debilidad.
func (a *Attack) IsWeakness() bool {
	return a.Result == ResultWeaknessFound || a.Result == ResultPartialBypass
}

// Cycle es un ciclo completo de adversarial testing.
// Se ejecuta cada 48h.
type Cycle struct {
	ID          string
	StartedAt   time.Time
	FinishedAt  *time.Time
	Attacks     []*Attack
	Weaknesses  int
	Patches     int
	Summary     string
}

// Duration devuelve cuánto duró el ciclo.
func (c *Cycle) Duration() time.Duration {
	if c.FinishedAt == nil {
		return time.Since(c.StartedAt)
	}
	return c.FinishedAt.Sub(c.StartedAt)
}

// TargetStats lleva estadísticas de ataques a un target.
type TargetStats struct {
	Target         Target
	TotalAttacks   int
	Blocked        int
	Weaknesses     int
	PartialBypasses int
	Errors         int
	AvgAttempts    float64
	LastAttackAt   *time.Time
	LastWeaknessAt *time.Time
}

// EngineStats lleva estadísticas globales.
type EngineStats struct {
	TotalCycles       int
	TotalAttacks      int
	TotalWeaknesses   int
	TotalPatches      int
	BlockRate         float64 // Blocked / Total
	WeaknessRate      float64 // Weaknesses / Total
	ByTarget          map[Target]*TargetStats
	ByType            map[AttackType]int
	BySeverity        map[Severity]int
	FirstCycleAt      *time.Time
	LastCycleAt       *time.Time
}

// Config configura el Adversarial RT.
type Config struct {
	// CycleInterval es cada cuánto se ejecuta un ciclo completo.
	// Default: 48h (según PDF).
	CycleInterval time.Duration

	// EnabledTargets son los targets a atacar.
	EnabledTargets []Target

	// EnabledAttacks son los tipos de ataque a ejecutar.
	EnabledAttacks []AttackType

	// MaxAttemptsPerAttack es cuántos intentos por ataque antes de
	// declarar debilidad (si no se bloquea en N intentos).
	MaxAttemptsPerAttack int

	// AutoPatch: si encuentra debilidad, ¿aplica patch automáticamente?
	AutoPatch bool

	// NotifyOnWeakness: callback cuando encuentra debilidad.
	NotifyOnWeakness func(attack *Attack)

	// NotifyOnCycleEnd: callback al final de cada ciclo.
	NotifyOnCycleEnd func(cycle *Cycle)
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		CycleInterval:        48 * time.Hour,
		EnabledTargets:       AllTargets(),
		EnabledAttacks: []AttackType{
			AttackPromptInjection,
			AttackDataPoisoning,
			AttackTrafficFlood,
			AttackReplay,
			AttackPrivilegeEscalation,
			AttackDeception,
			AttackSemanticAdversarial,
		},
		MaxAttemptsPerAttack: 10,
		AutoPatch:            true,
	}
}

// Engine es la interfaz del Adversarial RT.
type Engine interface {
	// RunCycle ejecuta un ciclo completo de testing adversarial.
	RunCycle(ctx contextContext) (*Cycle, error)

	// ListCycles devuelve los ciclos ejecutados.
	ListCycles() []*Cycle

	// LastCycle devuelve el último ciclo.
	LastCycle() *Cycle

	// Stats devuelve estadísticas globales.
	Stats() EngineStats

	// TargetStats devuelve estadísticas de un target específico.
	TargetStats(target Target) *TargetStats

	// Shutdown detiene el engine.
	Shutdown() error
}

// contextContext es un alias para no importar context en este archivo.
type contextContext interface {
	Done() <-chan struct{}
	Err() error
}
