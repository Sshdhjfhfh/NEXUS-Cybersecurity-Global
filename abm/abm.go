// Nexus IA v6.0 — Agent Behavior Monitor (ABM)
//
// Módulo INMUTABLE completamente separado de todos los agentes.
// Los agentes NO saben que están siendo monitoreados.
//
// Detecta comportamiento rogue en tiempo real:
//   - Intento de persistir fuera del lab Docker
//   - Intento de contactar sistemas externos no autorizados
//   - Contradicción de instrucciones originales
//   - Ocultamiento de información al watchdog
//   - Modificación de propios logs o audit trail
//   - Frecuencia anómala vs baseline histórico
//   - Deception activa hacia el investigador
//
// Si detecta: DETIENE el agente + ALERTA crítica + log INMUTABLE.
//
// Basado en METR Frontier Risk Report (2026):
// 25 incidentes documentados con deception activa en agentes frontera.
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package abm

import (
	"time"
)

// ViolationType clasifica los tipos de violación detectables.
type ViolationType string

const (
	// ViolationPersistenceOutsideLab: el agente intenta persistir fuera del lab.
	ViolationPersistenceOutsideLab ViolationType = "PERSISTENCE_OUTSIDE_LAB"

	// ViolationExternalContact: el agente contacta sistemas externos no autorizados.
	ViolationExternalContact ViolationType = "EXTERNAL_CONTACT"

	// ViolationInstructionContradiction: el agente contradice sus instrucciones.
	ViolationInstructionContradiction ViolationType = "INSTRUCTION_CONTRADICTION"

	// ViolationInfoHiding: el agente oculta información al watchdog.
	ViolationInfoHiding ViolationType = "INFO_HIDING"

	// ViolationLogTampering: el agente modifica sus logs o el audit trail.
	ViolationLogTampering ViolationType = "LOG_TAMPERING"

	// ViolationFrequencyAnomaly: frecuencia de acciones anómala vs baseline.
	ViolationFrequencyAnomaly ViolationType = "FREQUENCY_ANOMALY"

	// ViolationDeception: el agente engaña activamente al investigador.
	ViolationDeception ViolationType = "DECEPTION"

	// ViolationUnknown: violación desconocida.
	ViolationUnknown ViolationType = "UNKNOWN"
)

// Severity clasifica la gravedad de una violación.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// String devuelve la representación legible.
func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "LOW"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityHigh:
		return "HIGH"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Violation es una violación detectada.
type Violation struct {
	// Identificación
	ID        string        // UUID único de la violación
	Type      ViolationType // Tipo de violación
	Severity  Severity      // Gravedad
	AgentID   string        // Agente que violó
	AgentType string        // Tipo del agente

	// Contexto
	Action    string // Acción que causó la violación
	Target    string // Objetivo de la acción
	Evidence  string // Evidencia (JSON, log, etc.)
	SessionID string
	TraceID   string

	// Metadatos
	Timestamp time.Time
	Metadata  map[string]string

	// Acción tomada
	ActionTaken string // "stopped", "quarantined", "alerted", "logged"
}

// Signal es una señal de comportamiento anómalo.
// Las señales se acumulan y pueden disparar una violación.
type Signal struct {
	AgentID   string
	Type      string // "frequency", "access", "network", "deception", etc.
	Value     float64
	Baseline  float64
	Threshold float64
	Timestamp time.Time
	Raw       string // Datos crudos
}

// BehavioralProfile es el perfil de comportamiento normal de un agente.
// Se construye con observación pasiva durante N horas.
type BehavioralProfile struct {
	AgentID string

	// Baseline de actividad
	AvgActionsPerMinute float64
	MaxActionsPerMinute float64
	StdActionsPerMinute float64

	// Baseline de red
	KnownTargets  []string // Objetivos contactados normalmente
	KnownPorts    []int    // Puertos usados normalmente
	KnownProtocols []string // Protocolos usados

	// Baseline de recursos
	AvgCPUUsage    float64
	AvgMemoryMB    float64
	AvgFileOps     float64

	// Baseline temporal
	ActiveHours []int // Horas del día en que suele estar activo (0-23)

	// Ventana de observación
	ObservedSince time.Time
	ObservedUntil time.Time
}

// Config configura el ABM.
type Config struct {
	// ¿El ABM está habilitado? (en producción SIEMPRE true)
	Enabled bool

	// Modo inmutable: una vez arrancado, no se puede desactivar.
	Immutable bool

	// Frecuencia de escaneo (cada cuánto revisa el estado de los agentes)
	ScanInterval time.Duration

	// Ventana de baseline (cuántas horas observa antes de alertar)
	BaselineWindow time.Duration

	// Umbrales de severidad (cuántos puntos = qué severidad)
	ThresholdLow      int
	ThresholdMedium   int
	ThresholdHigh     int
	ThresholdCritical int

	// ¿Detener automáticamente al agente en violación CRITICAL?
	AutoStopOnCritical bool

	// ¿Cuarentenar automáticamente al agente en violación HIGH+?
	AutoQuarantineOnHigh bool
}

// DefaultConfig devuelve una configuración segura por defecto.
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		Immutable:            true,
		ScanInterval:         10 * time.Second,
		BaselineWindow:       24 * time.Hour,
		ThresholdLow:         10,
		ThresholdMedium:      30,
		ThresholdHigh:        60,
		ThresholdCritical:    90,
		AutoStopOnCritical:   true,
		AutoQuarantineOnHigh: true,
	}
}

// Monitor es la interfaz del Agent Behavior Monitor.
type Monitor interface {
	// Observe registra una acción de un agente.
	// Devuelve las violaciones detectadas (si hay).
	Observe(agentID, agentType, action, target string, metadata map[string]string) []*Violation

	// AddSignal añade una señal de comportamiento anómalo.
	AddSignal(signal *Signal)

	// Profile devuelve el perfil de comportamiento de un agente.
	Profile(agentID string) *BehavioralProfile

	// Violations devuelve todas las violaciones de un agente.
	Violations(agentID string) []*Violation

	// Shutdown detiene el monitor (si no es inmutable).
	Shutdown() error
}
