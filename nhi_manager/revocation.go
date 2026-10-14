// Nexus IA v6.0 — Sistema de revocación de identidades
//
// Cuando un agente es comprometido, se revoca su identidad:
//   - Su NHI queda marcada como revocada
//   - Su JWT deja de valer (aunque no haya expirado)
//   - Su certificado X.509 se añade a la CRL interna
//   - Se registra el motivo en el audit trail
//
// La revocación es INMEDIATA: el siguiente Verify del JWT falla.
package nhi_manager

import (
	"fmt"
	"sync"
	"time"
)

// RevocationReason clasifica por qué se revoca una identidad.
type RevocationReason string

const (
	ReasonCompromised   RevocationReason = "COMPROMISED"
	ReasonManual        RevocationReason = "MANUAL"
	ReasonExpired       RevocationReason = "EXPIRED"
	ReasonPolicy        RevocationReason = "POLICY_VIOLATION"
	ReasonRotation      RevocationReason = "KEY_ROTATION"
)

// RevocationRegistry mantiene la lista de identidades revocadas.
// Es thread-safe.
type RevocationRegistry struct {
	mu         sync.RWMutex
	revoked    map[string]*RevocationRecord // key: AgentID
	byDID      map[string]string            // key: DID → AgentID
}

// RevocationRecord es un registro de revocación.
type RevocationRecord struct {
	AgentID     string           `json:"agent_id"`
	DID         string           `json:"did"`
	Reason      RevocationReason `json:"reason"`
	Message     string           `json:"message,omitempty"`
	RevokedAt   time.Time        `json:"revoked_at"`
	RevokedBy   string           `json:"revoked_by"` // Quién revocó (admin, watchdog, etc.)
	AuditID     string           `json:"audit_id,omitempty"`
}

// NewRevocationRegistry crea un registro de revocaciones vacío.
func NewRevocationRegistry() *RevocationRegistry {
	return &RevocationRegistry{
		revoked: make(map[string]*RevocationRecord),
		byDID:   make(map[string]string),
	}
}

// Revoke añade una identidad a la lista de revocadas.
// Si ya estaba revocada, actualiza el registro.
func (r *RevocationRegistry) Revoke(nhi *NHI, reason RevocationReason, message, revokedBy string) *RevocationRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	record := &RevocationRecord{
		AgentID:   nhi.AgentID,
		DID:       nhi.DID.String(),
		Reason:    reason,
		Message:   message,
		RevokedAt: time.Now(),
		RevokedBy: revokedBy,
	}

	r.revoked[nhi.AgentID] = record
	r.byDID[nhi.DID.String()] = nhi.AgentID

	// Marcar el NHI como revocado también
	nhi.Revoke(string(reason))

	return record
}

// IsRevoked devuelve true si el AgentID está en la lista de revocados.
func (r *RevocationRegistry) IsRevoked(agentID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.revoked[agentID]
	return ok
}

// IsDIDRevoked devuelve true si el DID está en la lista de revocados.
func (r *RevocationRegistry) IsDIDRevoked(did string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byDID[did]
	return ok
}

// Get devuelve el registro de revocación de un AgentID.
// Devuelve nil si no está revocado.
func (r *RevocationRegistry) Get(agentID string) *RevocationRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.revoked[agentID]
}

// List devuelve todos los registros de revocación.
func (r *RevocationRegistry) List() []*RevocationRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*RevocationRecord, 0, len(r.revoked))
	for _, rec := range r.revoked {
		out = append(out, rec)
	}
	return out
}

// Count devuelve cuántas identidades están revocadas.
func (r *RevocationRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.revoked)
}

// VerifyAndRevoked verifica que un JWT no esté revocado.
// Devuelve error si el agente fue revocado.
func (r *RevocationRegistry) VerifyNotRevoked(claims *ClaimsNexus) error {
	if r.IsRevoked(claims.AgentID) {
		rec := r.Get(claims.AgentID)
		return fmt.Errorf("agent %s revoked: %s (%s)",
			claims.AgentID, rec.Reason, rec.RevokedAt.Format(time.RFC3339))
	}
	if r.IsDIDRevoked(claims.DID) {
		return fmt.Errorf("did %s revoked", claims.DID)
	}
	return nil
}
