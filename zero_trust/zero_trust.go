// Nexus IA v6.0 — Zero Trust Framework (MAESTRO)
//
// Framework Zero Trust completo basado en:
//   - MAESTRO (BSI alemán — Bundesamt für Sicherheit in der Informationstechnik)
//   - Agentic AI Survey 2026
//
// Los 5 principios fundamentales:
//
//   1. Mínimo privilegio — cada agente solo hace lo que su rol define
//   2. Never trust, always verify — cada llamada entre módulos se autentica
//   3. Micro-segmentación — cada agente en su propia red Docker
//   4. Assume breach — el sistema asume que cualquier agente puede comprometerse
//   5. Audit everything — hash + firma Dilithium + timestamp en cada acción
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package zero_trust

import (
	"time"
)

// Principle enumera los 5 principios MAESTRO.
type Principle string

const (
	PrincipleMinimalPrivilege Principle = "MINIMAL_PRIVILEGE"
	PrincipleNeverTrust        Principle = "NEVER_TRUST_ALWAYS_VERIFY"
	PrincipleMicroSegmentation Principle = "MICRO_SEGMENTATION"
	PrincipleAssumeBreach      Principle = "ASSUME_BREACH"
	PrincipleAuditEverything   Principle = "AUDIT_EVERYTHING"
)

// AllPrinciples devuelve los 5 principios MAESTRO.
func AllPrinciples() []Principle {
	return []Principle{
		PrincipleMinimalPrivilege,
		PrincipleNeverTrust,
		PrincipleMicroSegmentation,
		PrincipleAssumeBreach,
		PrincipleAuditEverything,
	}
}

// Decision es el resultado de una verificación Zero-Trust.
type Decision int

// Orden de severidad (worst wins al consolidar):
//   Allow (0) < Ask (1) < Deny (2) < Quarantine (3)
const (
	// Allow: la acción es permitida.
	Allow Decision = iota
	// Ask: requiere aprobación humana.
	Ask
	// Deny: la acción es rechazada.
	Deny
	// Quarantine: el agente es aislado (asume breach).
	Quarantine
)

// String devuelve una representación legible de la decisión.
func (d Decision) String() string {
	switch d {
	case Deny:
		return "DENY"
	case Allow:
		return "ALLOW"
	case Ask:
		return "ASK"
	case Quarantine:
		return "QUARANTINE"
	default:
		return "UNKNOWN"
	}
}

// RiskLevel clasifica el riesgo de una acción.
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskHigh     RiskLevel = "HIGH"
	RiskCritical RiskLevel = "CRITICAL"
)

// Action es una acción que un agente quiere ejecutar.
type Action struct {
	// Identificación
	ActorID  string // AgentID del actor
	ActorDID string // DID del actor
	AgentType string // Tipo de agente

	// Acción
	Type   string // Tipo de acción (ActionType del nhi_manager)
	Target string // Módulo o asset objetivo

	// Contexto
	SessionID string
	TraceID   string
	Timestamp time.Time

	// Metadatos
	Metadata map[string]string
}

// Request es una solicitud de verificación.
type Request struct {
	Action Action

	// JWT del actor (verificado por nhi_manager)
	JWT string

	// Certificado X.509 (opcional)
	CertPEM []byte

	// Riesgo estimado por el caller
	Risk RiskLevel

	// ¿Requiere aprobación humana?
	RequiresHumanApproval bool
}

// Result es el resultado de una verificación.
type Result struct {
	Decision  Decision
	Principle Principle // Qué principio aplicó
	Reason    string    // Por qué
	TTL       time.Duration // Cuánto tiempo vale esta decisión
	RequiresHumanApproval bool
	TraceID   string
	Timestamp time.Time
}

// Config configura el framework Zero-Trust.
type Config struct {
	// ¿Requerir mTLS en todas las conexiones?
	RequireMTLS bool

	// ¿Aislar automáticamente en breach detectado?
	AutoQuarantine bool

	// TTL por defecto de las decisiones
	DefaultDecisionTTL time.Duration

	// ¿Auditar CADA acción?
	AuditEverything bool

	// Riesgo máximo permitido sin aprobación humana
	MaxRiskWithoutApproval RiskLevel
}

// DefaultConfig devuelve una configuración segura por defecto.
func DefaultConfig() Config {
	return Config{
		RequireMTLS:            true,
		AutoQuarantine:         true,
		DefaultDecisionTTL:     5 * time.Minute,
		AuditEverything:        true,
		MaxRiskWithoutApproval: RiskMedium,
	}
}

// Rule es una regla de Zero-Trust.
// Cada regla implementa un principio MAESTRO.
type Rule interface {
	// Name devuelve el nombre de la regla.
	Name() string

	// Principle devuelve el principio MAESTRO que implementa.
	Principle() Principle

	// Evaluate evalúa la acción y devuelve una decisión.
	// Si no aplica, devuelve Allow con Reason "not applicable".
	Evaluate(req *Request) Result
}

// String devuelve la representación legible del principio.
func (p Principle) String() string {
	return string(p)
}
