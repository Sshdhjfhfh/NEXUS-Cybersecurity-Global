// Nexus IA v6.0 — Non-Human Identity Manager (NHI)
//
// Cada agente tiene identidad criptográfica propia:
//   - Certificado X.509 único (renovación automática cada 24h)
//   - DID (Decentralized Identifier) único e inmutable
//   - JWT con claims exactos de permisos por acción
//   - Audit trail criptográfico firmado con Dilithium de CADA acción
//   - Revocación inmediata si el agente es comprometido
//
// Permite saber exactamente qué agente hizo qué, cuándo y con qué autorización.
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package nhi_manager

import (
	"time"

	"github.com/google/uuid"
)

// AgentType clasifica el tipo de agente (RED, BLUE, PREDICTION, etc.).
type AgentType string

const (
	AgentTypeRed               AgentType = "RED"
	AgentTypeBlue              AgentType = "BLUE"
	AgentTypePrediction        AgentType = "PREDICTION"
	AgentTypeSwarmCoordinator  AgentType = "SWARM_COORDINATOR"
	AgentTypeScout             AgentType = "SCOUT"
	AgentTypeEnum              AgentType = "ENUM"
	AgentTypeExploitNet        AgentType = "EXPLOIT_NET"
	AgentTypeExploitWeb        AgentType = "EXPLOIT_WEB"
	AgentTypePostExploit       AgentType = "POST_EXPLOIT"
	AgentTypeStealth           AgentType = "STEALTH"
)

// ActionType es un tipo de acción que un agente puede ejecutar.
// Cada acción requiere permisos específicos en el JWT.
type ActionType string

const (
	ActionScan             ActionType = "SCAN"
	ActionExploit          ActionType = "EXPLOIT"
	ActionDumpCredentials  ActionType = "DUMP_CREDENTIALS"
	ActionLateralMove      ActionType = "LATERAL_MOVE"
	ActionGeneratePayload  ActionType = "GENERATE_PAYLOAD"
	ActionEvolveModule     ActionType = "EVOLVE_MODULE"
	ActionQueryKnowledge   ActionType = "QUERY_KNOWLEDGE"
	ActionIsolateHost      ActionType = "ISOLATE_HOST"
	ActionDeployDecoy      ActionType = "DECOY"
)

// DID es un Decentralized Identifier único e inmutable.
// Formato: did:nexus:<agent-uuid>
type DID string

// NewDID genera un DID único a partir de un UUID.
func NewDID() DID {
	return DID("did:nexus:" + uuid.New().String())
}

// String devuelve el DID como string.
func (d DID) String() string {
	return string(d)
}

// NHI es una identidad completa de agente no-humano.
// Incluye:
//   - Identidad lógica (agentID, did)
//   - Credenciales criptográficas (certX509, jwtToken)
//   - Permisos (allowedActions)
//   - Ciclo de vida (issuedAt, expiresAt, revoked)
type NHI struct {
	// Identidad
	AgentID   string    // ID lógico del agente (ej: "red-agent-01")
	AgentType AgentType // Tipo de agente
	DID       DID       // Decentralized Identifier único

	// Credenciales
	CertX509PEM []byte // Certificado X.509 en formato PEM
	PublicKey   []byte // Clave pública (Dilithium3)
	JWTToken    string // JWT con claims de permisos

	// Permisos
	AllowedActions []ActionType // Acciones permitidas a este agente
	AllowedAssets  []string     // Assets que puede tocar (glob patterns)

	// Ciclo de vida
	IssuedAt  time.Time  // Cuándo se emitió
	ExpiresAt time.Time  // Cuándo expira (24h por defecto)
	RevokedAt *time.Time // Cuándo se revocó (nil si activa)
	RevokeReason string  // Motivo de revocación
}

// IsExpired devuelve true si la identidad expiró.
func (n *NHI) IsExpired() bool {
	return time.Now().After(n.ExpiresAt)
}

// IsRevoked devuelve true si la identidad fue revocada.
func (n *NHI) IsRevoked() bool {
	return n.RevokedAt != nil
}

// IsValid devuelve true si la identidad es válida para uso.
// Una identidad es válida si:
//   - No está revocada
//   - No ha expirado
func (n *NHI) IsValid() bool {
	return !n.IsRevoked() && !n.IsExpired()
}

// TTL devuelve el tiempo restante de vida de la identidad.
// Devuelve 0 si ya expiró.
func (n *NHI) TTL() time.Duration {
	if n.IsExpired() {
		return 0
	}
	return time.Until(n.ExpiresAt)
}

// HasPermission devuelve true si el agente puede ejecutar la acción.
func (n *NHI) HasPermission(action ActionType) bool {
	for _, a := range n.AllowedActions {
		if a == action {
			return true
		}
	}
	return false
}

// Revoke marca la identidad como revocada.
// Una vez revocada, no puede volver a usarse.
func (n *NHI) Revoke(reason string) {
	now := time.Now()
	n.RevokedAt = &now
	n.RevokeReason = reason
}

// String devuelve una representación legible de la identidad.
func (n *NHI) String() string {
	return "NHI{" + n.AgentID + " (" + string(n.AgentType) + ") " + n.DID.String() + "}"
}
