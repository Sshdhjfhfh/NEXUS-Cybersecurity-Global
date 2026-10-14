// Nexus IA v6.0 — Phantom VPN — Base criptográfica
//
// Criptografía post-cuántica híbrida para túneles VPN:
//   - X25519 (ECDH clásico)
//   - Kyber768 (KEM post-cuántico, NIST FIPS 203)
//   - ChaCha20-Poly1305 (cifrado simétrico AEAD)
//   - HKDF-SHA256 (derivación de claves)
//
// El híbrido X25519 + Kyber768 garantiza seguridad incluso si:
//   - X25519 es roto por un ordenador cuántico
//   - Kyber768 tiene un fallo desconocido
//
// Nivel 1-2 — Phantom — Sub-módulo VPN
package vpn

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// X25519KeySize es el tamaño de clave X25519 (32 bytes).
	X25519KeySize = 32

	// Kyber768PublicKeySize es el tamaño de clave pública Kyber768.
	Kyber768PublicKeySize = kyber768.PublicKeySize

	// Kyber768PrivateKeySize es el tamaño de clave privada Kyber768.
	Kyber768PrivateKeySize = kyber768.PrivateKeySize

	// Kyber768SharedSecretSize es el tamaño del secret compartido Kyber768.
	Kyber768SharedSecretSize = kyber768.SharedKeySize

	// ChaCha20Poly1305KeySize es el tamaño de clave ChaCha20-Poly1305 (32 bytes).
	ChaCha20Poly1305KeySize = chacha20poly1305.KeySize

	// NonceSize es el tamaño del nonce ChaCha20-Poly1305 (12 bytes).
	NonceSize = chacha20poly1305.NonceSize

	// SessionKeySize es el tamaño de la clave de sesión derivada (32 bytes).
	SessionKeySize = 32
)

// ═══════════════════════════════════════════════════════════════
// X25519 — ECDH clásico
// ═══════════════════════════════════════════════════════════════

// X25519KeyPair es un par de claves X25519 efímeras.
type X25519KeyPair struct {
	PublicKey  [X25519KeySize]byte
	PrivateKey [X25519KeySize]byte
}

// GenerateX25519KeyPair genera un par de claves X25519 fresco.
func GenerateX25519KeyPair() (*X25519KeyPair, error) {
	var priv [X25519KeySize]byte
	if _, err := io.ReadFull(rand.Reader, priv[:]); err != nil {
		return nil, fmt.Errorf("generate x25519 private key: %w", err)
	}

	var pub [X25519KeySize]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return &X25519KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// SharedSecret computa el shared secret ECDH con la clave pública del peer.
func (kp *X25519KeyPair) SharedSecret(peerPublicKey [X25519KeySize]byte) ([]byte, error) {
	shared, err := curve25519.X25519(kp.PrivateKey[:], peerPublicKey[:])
	if err != nil {
		return nil, fmt.Errorf("x25519 shared secret: %w", err)
	}
	if isAllZero(shared) {
		return nil, errors.New("x25519: all-zero shared secret (invalid public key)")
	}
	return shared, nil
}

// ═══════════════════════════════════════════════════════════════
// Kyber768 — KEM post-cuántico
// ═══════════════════════════════════════════════════════════════

// Kyber768KeyPair es un par de claves Kyber768.
type Kyber768KeyPair struct {
	PublicKey  *kyber768.PublicKey
	PrivateKey *kyber768.PrivateKey
}

// GenerateKyber768KeyPair genera un par de claves Kyber768 fresco.
func GenerateKyber768KeyPair() (*Kyber768KeyPair, error) {
	pub, priv, err := kyber768.GenerateKeyPair(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate kyber768 key pair: %w", err)
	}
	return &Kyber768KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// Encapsulate usa la clave pública para encapsular un secret.
// Devuelve (ciphertext, sharedSecret).
func (kp *Kyber768KeyPair) Encapsulate() ([]byte, []byte, error) {
	if kp.PublicKey == nil {
		return nil, nil, errors.New("nil public key")
	}
	ct := make([]byte, kyber768.CiphertextSize)
	ss := make([]byte, Kyber768SharedSecretSize)
	kp.PublicKey.EncapsulateTo(ct, ss, nil)
	return ct, ss, nil
}

// Decapsulate usa la clave privada para recuperar el secret.
// Devuelve sharedSecret.
func (kp *Kyber768KeyPair) Decapsulate(ciphertext []byte) ([]byte, error) {
	if kp.PrivateKey == nil {
		return nil, errors.New("nil private key")
	}
	if len(ciphertext) != kyber768.CiphertextSize {
		return nil, fmt.Errorf("invalid ciphertext size: %d, expected %d", len(ciphertext), kyber768.CiphertextSize)
	}
	ss := make([]byte, Kyber768SharedSecretSize)
	kp.PrivateKey.DecapsulateTo(ss, ciphertext)
	return ss, nil
}

// ═══════════════════════════════════════════════════════════════
// Híbrido X25519 + Kyber768
// ═══════════════════════════════════════════════════════════════

// HybridKeyPair es un par de claves híbrido X25519 + Kyber768.
type HybridKeyPair struct {
	X25519 *X25519KeyPair
	Kyber  *Kyber768KeyPair
}

// GenerateHybridKeyPair genera un par de claves híbrido fresco.
func GenerateHybridKeyPair() (*HybridKeyPair, error) {
	x, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("x25519: %w", err)
	}
	k, err := GenerateKyber768KeyPair()
	if err != nil {
		return nil, fmt.Errorf("kyber768: %w", err)
	}
	return &HybridKeyPair{
		X25519: x,
		Kyber:  k,
	}, nil
}

// HybridSharedSecret deriva el secret compartido híbrido.
// Combina X25519 + Kyber768 + nonces usando HKDF-SHA256.
//
// Devuelve 32 bytes de clave de sesión.
func HybridSharedSecret(
	x25519Secret []byte,
	kyberSecret []byte,
	info string,
) ([]byte, error) {
	if len(x25519Secret) == 0 {
		return nil, errors.New("empty x25519 secret")
	}
	if len(kyberSecret) == 0 {
		return nil, errors.New("empty kyber secret")
	}

	// Concatenar ambos secrets (X25519 || Kyber768)
	combined := append(append([]byte{}, x25519Secret...), kyberSecret...)

	// HKDF-SHA256 para derivar clave de sesión
	hkdfReader := hkdf.New(sha256.New, combined, nil, []byte(info))
	sessionKey := make([]byte, SessionKeySize)
	if _, err := io.ReadFull(hkdfReader, sessionKey); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}

	return sessionKey, nil
}

// ═══════════════════════════════════════════════════════════════
// ChaCha20-Poly1305 — AEAD simétrico
// ═══════════════════════════════════════════════════════════════

// AEAD encapsula ChaCha20-Poly1305.
type AEAD struct {
	cipher interface {
		Seal(dst, nonce, plaintext, additionalData []byte) []byte
		Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
		NonceSize() int
		Overhead() int
	}
}

// NewAEAD crea un AEAD desde una clave de sesión.
func NewAEAD(sessionKey []byte) (*AEAD, error) {
	if len(sessionKey) != ChaCha20Poly1305KeySize {
		return nil, fmt.Errorf("invalid key size: %d, expected %d", len(sessionKey), ChaCha20Poly1305KeySize)
	}
	c, err := chacha20poly1305.New(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("create chacha20poly1305: %w", err)
	}
	return &AEAD{cipher: c}, nil
}

// Encrypt cifra plaintext con nonce y additionalData.
func (a *AEAD) Encrypt(nonce, plaintext, additionalData []byte) ([]byte, error) {
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: %d, expected %d", len(nonce), NonceSize)
	}
	return a.cipher.Seal(nil, nonce, plaintext, additionalData), nil
}

// Decrypt descifra ciphertext con nonce y additionalData.
func (a *AEAD) Decrypt(nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: %d, expected %d", len(nonce), NonceSize)
	}
	plaintext, err := a.cipher.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// GenerateNonce genera un nonce aleatorio de 12 bytes.
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return nonce, nil
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// isAllZero comprueba si un slice es todo ceros (indicador de clave inválida).
func isAllZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}


// PublicKeyBytes serializa la clave pública para almacenamiento.
func (kp *Kyber768KeyPair) PublicKeyBytes() []byte {
	buf := make([]byte, Kyber768PublicKeySize)
	kp.PublicKey.Pack(buf)
	return buf
}

// Kyber768PublicKeyFromBytes deserializa una clave pública.
func Kyber768PublicKeyFromBytes(b []byte) (*kyber768.PublicKey, error) {
	if len(b) != Kyber768PublicKeySize {
		return nil, errors.New("invalid kyber768 public key size")
	}
	pk := new(kyber768.PublicKey)
	pk.Unpack(b)
	return pk, nil
}
