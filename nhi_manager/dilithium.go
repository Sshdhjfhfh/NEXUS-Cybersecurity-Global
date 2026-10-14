// Nexus IA v6.0 — Firma post-cuántica con Dilithium3
//
// Usa CRYSTALS-Dilithium modo 3 (seguridad equivalente a AES-192).
// Firrma y verifica el audit trail de cada acción de agente.
package nhi_manager

import (
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
)

// DilithiumKeyPair es un par de claves Dilithium3.
type DilithiumKeyPair struct {
	PublicKey  *mode3.PublicKey
	PrivateKey *mode3.PrivateKey
}

// GenerateDilithiumKeyPair genera un par de claves Dilithium3 fresco.
// Usa crypto/rand para el seed.
func GenerateDilithiumKeyPair() (*DilithiumKeyPair, error) {
	pub, priv, err := mode3.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate dilithium key: %w", err)
	}
	return &DilithiumKeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// PublicKeyBytes serializa la clave pública para almacenamiento.
func (kp *DilithiumKeyPair) PublicKeyBytes() []byte {
	return kp.PublicKey.Bytes()
}

// Sign firma un mensaje con Dilithium3.
// Devuelve la firma (2420 bytes para Dilithium3).
func (kp *DilithiumKeyPair) Sign(message []byte) []byte {
	sig := make([]byte, mode3.SignatureSize)
	mode3.SignTo(kp.PrivateKey, message, sig)
	return sig
}

// DilithiumPublicKeyFromBytes deserializa una clave pública.
func DilithiumPublicKeyFromBytes(b []byte) (*mode3.PublicKey, error) {
	pk := new(mode3.PublicKey)
	if err := pk.UnmarshalBinary(b); err != nil {
		return nil, fmt.Errorf("unmarshal dilithium public key: %w", err)
	}
	return pk, nil
}

// VerifyDilithium verifica una firma Dilithium3.
// Devuelve true si la firma es válida para el mensaje y la clave pública.
func VerifyDilithium(pubKey *mode3.PublicKey, message, signature []byte) bool {
	return mode3.Verify(pubKey, message, signature)
}

// VerifyDilithiumBytes es como VerifyDilithium pero acepta bytes en vez de
// un *mode3.PublicKey.
func VerifyDilithiumBytes(pubKeyBytes, message, signature []byte) (bool, error) {
	pk, err := DilithiumPublicKeyFromBytes(pubKeyBytes)
	if err != nil {
		return false, err
	}
	return VerifyDilithium(pk, message, signature), nil
}

// ErrDilithiumSignatureInvalid se devuelve cuando la firma no es válida.
var ErrDilithiumSignatureInvalid = errors.New("dilithium signature invalid")

// ErrDilithiumKeyInvalid se devuelve cuando la clave pública no es válida.
var ErrDilithiumKeyInvalid = errors.New("dilithium public key invalid")
