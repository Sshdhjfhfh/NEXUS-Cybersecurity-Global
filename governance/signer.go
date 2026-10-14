// Nexus IA v6.0 — Firma de reportes con Dilithium3
//
// Cada reporte de gobernanza se firma con Dilithium3 (post-cuántico).
// La firma permite:
//   - Verificar la autenticidad del reporte
//   - Detectar cualquier modificación posterior
//   - Cumplir con Cloud and AI Development Act (2026)
//
// El hash del contenido se incluye en el reporte para verificación
// independiente de la firma.
package governance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
)

// Signer firma reportes con Dilithium3.
type Signer struct {
	publicKey  *mode3.PublicKey
	privateKey *mode3.PrivateKey
	keyID      string
}

// NewSigner crea un nuevo Signer con un par de claves fresco.
func NewSigner(keyID string) (*Signer, error) {
	pub, priv, err := mode3.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generate dilithium key: %w", err)
	}
	return &Signer{
		publicKey:  pub,
		privateKey: priv,
		keyID:      keyID,
	}, nil
}

// PublicKeyBytes devuelve la clave pública serializada.
func (s *Signer) PublicKeyBytes() []byte {
	return s.publicKey.Bytes()
}

// KeyID devuelve el ID de la clave.
func (s *Signer) KeyID() string {
	return s.keyID
}

// SignReport firma un reporte.
//   - 1. Calcula SHA-256 del contenido canónico
//   - 2. Firma el hash con Dilithium3
//   - 3. Rellena ContentHash, Signature, SignatureAlg, SigningKeyID
func (s *Signer) SignReport(r *Report) error {
	if r == nil {
		return fmt.Errorf("nil report")
	}

	// 1. Serializar contenido canónico (sin firma)
	canonical, err := canonicalBytes(r)
	if err != nil {
		return fmt.Errorf("canonical bytes: %w", err)
	}

	// 2. Hash SHA-256
	hash := sha256.Sum256(canonical)
	r.ContentHash = hex.EncodeToString(hash[:])

	// 3. Firmar hash con Dilithium3
	sig := make([]byte, mode3.SignatureSize)
	mode3.SignTo(s.privateKey, hash[:], sig)

	r.Signature = hex.EncodeToString(sig)
	r.SignatureAlg = "dilithium3"
	r.SigningKeyID = s.keyID

	return nil
}

// VerifyReport verifica la firma de un reporte.
//   - 1. Recalcula el hash canónico
//   - 2. Verifica que coincida con ContentHash
//   - 3. Verifica la firma Dilithium
func (s *Signer) VerifyReport(r *Report) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("nil report")
	}
	if r.Signature == "" || r.ContentHash == "" {
		return false, fmt.Errorf("report missing signature")
	}

	// 1. Recalcular hash
	canonical, err := canonicalBytes(r)
	if err != nil {
		return false, fmt.Errorf("canonical bytes: %w", err)
	}
	hash := sha256.Sum256(canonical)
	recalcHash := hex.EncodeToString(hash[:])

	// 2. Verificar que el hash coincida
	if recalcHash != r.ContentHash {
		return false, fmt.Errorf("content hash mismatch: report modified")
	}

	// 3. Verificar firma
	sig, err := hex.DecodeString(r.Signature)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}

	ok := mode3.Verify(s.publicKey, hash[:], sig)
	return ok, nil
}

// canonicalBytes serializa el reporte para firma, excluyendo la firma
// y el hash (que se calculan).
func canonicalBytes(r *Report) ([]byte, error) {
	// Copia sin firma
	cpy := *r
	cpy.Signature = ""
	cpy.SignatureAlg = ""
	cpy.SigningKeyID = ""
	cpy.ContentHash = ""

	// Serializar con claves ordenadas (json.Marshal ordena por struct)
	return json.Marshal(&cpy)
}
