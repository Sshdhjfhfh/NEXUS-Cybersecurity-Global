// Nexus IA v6.0 — Agent Name Service (ANS)
//
// Directorio universal interno de todos los agentes de Nexus.
//
// Cada agente tiene un nombre DNS interno: nexus.local/agents/<name>
// El ANS mapea cada nombre a:
//   - Certificado X.509 (validado)
//   - Permisos (AllowedActions)
//   - Estado (activo, pausado, detenido, cuarentenado)
//   - Última acción conocida
//   - Tipo de agente
//
// REGLA ZERO TRUST:
//   Si un agente NO está en el ANS → NO puede actuar.
//
// Basado en Agent Name Service (ANS) del Agent AI Survey 2026.
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package ans

import (
	"strings"
	"time"
)

// ServiceDomain es el dominio raíz del ANS.
const ServiceDomain = "nexus.local"

// AgentName es el nombre completo de un agente en el ANS.
// Formato: nexus.local/agents/<shortname>
type AgentName string

// NewAgentName construye un AgentName desde un shortname.
func NewAgentName(short string) AgentName {
	short = strings.TrimSpace(short)
	short = strings.TrimPrefix(short, "/")
	return AgentName(ServiceDomain + "/agents/" + short)
}

// Short devuelve el nombre corto (sin dominio).
func (n AgentName) Short() string {
	return strings.TrimPrefix(string(n), ServiceDomain+"/agents/")
}

// String devuelve el nombre completo.
func (n AgentName) String() string {
	return string(n)
}

// IsValid comprueba que el nombre siga el formato esperado.
func (n AgentName) IsValid() bool {
	s := string(n)
	return strings.HasPrefix(s, ServiceDomain+"/agents/") && len(s) > len(ServiceDomain+"/agents/")
}

// AgentStatus es el estado de un agente en el ANS.
type AgentStatus string

const (
	// StatusActive: el agente está activo y puede actuar.
	StatusActive AgentStatus = "ACTIVE"

	// StatusPaused: el agente está pausado temporalmente.
	StatusPaused AgentStatus = "PAUSED"

	// StatusStopped: el agente fue detenido (por ABM o por humano).
	StatusStopped AgentStatus = "STOPPED"

	// StatusQuarantined: el agente está en cuarentena (Assume Breach).
	StatusQuarantined AgentStatus = "QUARANTINED"

	// StatusRevoked: el agente fue revocado (nhi_manager).
	StatusRevoked AgentStatus = "REVOKED"

	// StatusUnknown: el agente no está registrado.
	StatusUnknown AgentStatus = "UNKNOWN"
)

// String devuelve la representación legible.
func (s AgentStatus) String() string {
	return string(s)
}

// Entry es una entrada del ANS.
type Entry struct {
	// Identidad
	Name      AgentName   // nombre DNS interno
	AgentID   string      // ID del agente (coincide con nhi_manager)
	AgentType string      // RED, BLUE, PREDICTION, etc.
	DID       string      // DID del agente

	// Credenciales
	CertX509PEM []byte    // certificado X.509
	PublicKey   []byte    // clave pública Dilithium3

	// Permisos
	AllowedActions []string  // acciones permitidas
	AllowedAssets  []string  // assets autorizados

	// Estado
	Status     AgentStatus
	LastAction string
	LastSeen   time.Time

	// Ciclo de vida
	RegisteredAt time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
}

// IsValid comprueba si la entrada es válida para actuar.
// Es inválida si:
//   - Está revocada o detenida
//   - Está expirada
//   - No tiene certificado X.509
func (e *Entry) IsValid() bool {
	if e.Status == StatusRevoked || e.Status == StatusStopped {
		return false
	}
	if e.Status == StatusQuarantined {
		return false
	}
	if time.Now().After(e.ExpiresAt) {
		return false
	}
	if len(e.CertX509PEM) == 0 {
		return false
	}
	return true
}

// CanAct devuelve true si el agente puede ejecutar una acción.
func (e *Entry) CanAct(action string) bool {
	if !e.IsValid() {
		return false
	}
	for _, a := range e.AllowedActions {
		if a == action {
			return true
		}
	}
	return false
}

// Config configura el ANS.
type Config struct {
	// EntryTTL: cuánto dura una entrada antes de requerir re-registro.
	// Default: 24h (igual que el cert X.509)
	EntryTTL time.Duration

	// RequireCert: si es true, toda entrada debe tener cert X.509.
	RequireCert bool

	// RequireActiveStatus: si es true, solo agentes ACTIVE pueden actuar.
	RequireActiveStatus bool
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		EntryTTL:            24 * time.Hour,
		RequireCert:         true,
		RequireActiveStatus: true,
	}
}

// LookupResult es el resultado de una consulta al ANS.
type LookupResult struct {
	Found   bool
	Entry   *Entry
	Reason  string // Si no found: por qué
}

// RegisterRequest encapsula una solicitud de registro.
type RegisterRequest struct {
	Name           string
	AgentID        string
	AgentType      string
	DID            string
	CertX509PEM    []byte
	PublicKey      []byte
	AllowedActions []string
	AllowedAssets  []string
}

// LookupStats lleva estadísticas de uso del ANS.
type LookupStats struct {
	TotalLookups   int64
	TotalFound     int64
	TotalNotFound  int64
	TotalDenied    int64 // Encontrado pero status inválido
}
