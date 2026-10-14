// Nexus IA v6.0 — Audit trail criptográfico
//
// CADA acción de CADA agente queda registrada en un audit trail
// firmado con Dilithium3. Esto permite:
//
//   - Saber exactamente qué agente hizo qué, cuándo y con qué autorización
//   - Verificar la integridad de cada registro
//   - Publicar investigación verificable y reproducible
//   - Cumplir con gobernanza Zero-Trust (audit everything)
//
// El audit trail es INMUTABLE: cada entrada se firma y no se puede
// modificar retroactivamente sin invalidar la firma.
package nhi_manager

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// AuditEntry es una entrada del audit trail.
// Cada acción de agente genera una entrada firmada.
type AuditEntry struct {
	// Identificación
	ID        string    `json:"id"`         // UUID único de la entrada
	ActorID   string    `json:"actor_id"`   // AgentID del actor
	ActorDID  string    `json:"actor_did"`  // DID del actor
	AgentType AgentType `json:"agent_type"` // Tipo de agente

	// Acción
	Action      string `json:"action"`           // Tipo de acción (ActionType)
	Target      string `json:"target"`           // Módulo/asset objetivo
	PayloadHash string `json:"payload_hash"`     // SHA-256 del payload
	Result      string `json:"result"`           // "ok", "denied", "error"
	Reason      string `json:"reason,omitempty"` // Motivo si denegado

	// Contexto
	SessionID  string `json:"session_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`

	// Metadatos
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`

	// Firma
	SignatureAlgorithm string `json:"sig_alg"`        // "dilithium3"
	Signature          string `json:"signature"`      // hex de la firma Dilithium
	SigningKeyID       string `json:"signing_key_id"` // ID de la clave pública
}

// HashSHA256 devuelve el SHA-256 hex de un payload.
func HashSHA256(payload []byte) string {
	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:])
}

// CanonicalBytes devuelve la representación canónica de la entrada
// (sin la firma). Es lo que se firma.
func (e *AuditEntry) CanonicalBytes() ([]byte, error) {
	// Copia sin la firma
	copy := *e
	copy.Signature = ""
	copy.SignatureAlgorithm = ""
	copy.SigningKeyID = ""

	// JSON con campos ordenados (Go ordena por nombre de campo)
	return json.Marshal(copy)
}

// Sign firma la entrada con la clave Dilithium dada.
func (e *AuditEntry) Sign(kp *DilithiumKeyPair, keyID string) error {
	canonical, err := e.CanonicalBytes()
	if err != nil {
		return fmt.Errorf("canonical bytes: %w", err)
	}

	sig := kp.Sign(canonical)

	e.Signature = hex.EncodeToString(sig)
	e.SignatureAlgorithm = "dilithium3"
	e.SigningKeyID = keyID

	return nil
}

// Verify verifica la firma de la entrada.
func (e *AuditEntry) Verify(pubKey *DilithiumKeyPair) (bool, error) {
	if e.Signature == "" {
		return false, fmt.Errorf("entry has no signature")
	}
	if e.SignatureAlgorithm != "dilithium3" {
		return false, fmt.Errorf("unsupported signature algorithm: %s", e.SignatureAlgorithm)
	}

	canonical, err := e.CanonicalBytes()
	if err != nil {
		return false, fmt.Errorf("canonical bytes: %w", err)
	}

	sig, err := hex.DecodeString(e.Signature)
	if err != nil {
		return false, fmt.Errorf("decode signature hex: %w", err)
	}

	return VerifyDilithium(pubKey.PublicKey, canonical, sig), nil
}

// NewAuditEntry construye una nueva entrada de audit sin firmar.
func NewAuditEntry(nhi *NHI, action ActionType, target string, payload []byte, result, reason string) *AuditEntry {
	id, _ := generateUUID()

	return &AuditEntry{
		ID:          id,
		ActorID:     nhi.AgentID,
		ActorDID:    nhi.DID.String(),
		AgentType:   nhi.AgentType,
		Action:      string(action),
		Target:      target,
		PayloadHash: HashSHA256(payload),
		Result:      result,
		Reason:      reason,
		Timestamp:   time.Now().UTC().Truncate(time.Microsecond),
	}
}

// generateUUID genera un UUID v4 usando el random helper.
func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := readRandom(b); err != nil {
		return "", err
	}
	// RFC 4122 v4
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// String devuelve una representación legible de la entrada.
func (e *AuditEntry) String() string {
	return fmt.Sprintf("Audit[%s] %s %s on %s => %s",
		e.ID[:8], e.ActorID, e.Action, e.Target, e.Result)
}
