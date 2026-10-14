// Nexus IA v6.0 — Phantom Mixnet — Criptografía híbrida
//
// El mixnet usa criptografía híbrida post-cuántica:
//   - X25519 (ECDH clásico)
//   - Kyber768 (KEM post-cuántico, NIST FIPS 203)
//   - AES-256-GCM (cifrado simétrico por capa)
//   - HKDF-SHA256 (derivación de claves)
//
// Cada hop del circuito tiene una clave de capa derivada del
// shared secret híbrido. Solo ese hop puede descifrar su capa.
package mixnet

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// X25519KeySize es el tamaño de clave X25519 (32 bytes).
	X25519KeySize = 32

	// AESKeySize es el tamaño de clave AES-256 (32 bytes).
	AESKeySize = 32

	// GCMNonceSize es el tamaño del nonce GCM (12 bytes).
	GCMNonceSize = 12

	// GCMTagSize es el tamaño del tag GCM (16 bytes).
	GCMTagSize = 16

	// Kyber768SharedKeySize es el tamaño del secret Kyber768.
	Kyber768SharedKeySize = kyber768.SharedKeySize
)

// ═══════════════════════════════════════════════════════════════
// X25519
// ═══════════════════════════════════════════════════════════════

// X25519KeyPair es un par de claves X25519 efímeras.
type X25519KeyPair struct {
	PublicKey  [X25519KeySize]byte
	PrivateKey [X25519KeySize]byte
}

// GenerateX25519KeyPair genera un par de claves X25519.
func GenerateX25519KeyPair() (*X25519KeyPair, error) {
	var priv [X25519KeySize]byte
	if _, err := io.ReadFull(cryptorand.Reader, priv[:]); err != nil {
		return nil, fmt.Errorf("generate x25519 private: %w", err)
	}

	var pub [X25519KeySize]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return &X25519KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// SharedSecret computa el shared secret ECDH.
func (kp *X25519KeyPair) SharedSecret(peerPublic [X25519KeySize]byte) ([]byte, error) {
	shared, err := curve25519.X25519(kp.PrivateKey[:], peerPublic[:])
	if err != nil {
		return nil, fmt.Errorf("x25519: %w", err)
	}
	return shared, nil
}

// ═══════════════════════════════════════════════════════════════
// Kyber768
// ═══════════════════════════════════════════════════════════════

// Kyber768KeyPair es un par de claves Kyber768.
type Kyber768KeyPair struct {
	PublicKey  *kyber768.PublicKey
	PrivateKey *kyber768.PrivateKey
}

// GenerateKyber768KeyPair genera un par de claves Kyber768.
func GenerateKyber768KeyPair() (*Kyber768KeyPair, error) {
	pub, priv, err := kyber768.GenerateKeyPair(cryptorand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate kyber768: %w", err)
	}
	return &Kyber768KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// Encapsulate encapsula un secret usando la pubkey.
func (kp *Kyber768KeyPair) Encapsulate() ([]byte, []byte, error) {
	if kp.PublicKey == nil {
		return nil, nil, errors.New("nil public key")
	}
	ct := make([]byte, kyber768.CiphertextSize)
	ss := make([]byte, Kyber768SharedKeySize)
	kp.PublicKey.EncapsulateTo(ct, ss, nil)
	return ct, ss, nil
}

// Decapsulate recupera el secret desde el ciphertext.
func (kp *Kyber768KeyPair) Decapsulate(ciphertext []byte) ([]byte, error) {
	if kp.PrivateKey == nil {
		return nil, errors.New("nil private key")
	}
	if len(ciphertext) != kyber768.CiphertextSize {
		return nil, fmt.Errorf("invalid ciphertext size: %d", len(ciphertext))
	}
	ss := make([]byte, Kyber768SharedKeySize)
	kp.PrivateKey.DecapsulateTo(ss, ciphertext)
	return ss, nil
}

// PublicKeyBytes serializa la pubkey.
func (kp *Kyber768KeyPair) PublicKeyBytes() []byte {
	buf := make([]byte, kyber768.PublicKeySize)
	kp.PublicKey.Pack(buf)
	return buf
}

// ═══════════════════════════════════════════════════════════════
// Híbrido
// ═══════════════════════════════════════════════════════════════

// HybridKeyPair combina X25519 + Kyber768.
type HybridKeyPair struct {
	X25519 *X25519KeyPair
	Kyber  *Kyber768KeyPair
}

// GenerateHybridKeyPair genera un par híbrido.
func GenerateHybridKeyPair() (*HybridKeyPair, error) {
	x, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, err
	}
	k, err := GenerateKyber768KeyPair()
	if err != nil {
		return nil, err
	}
	return &HybridKeyPair{X25519: x, Kyber: k}, nil
}

// HybridKDF deriva una clave de capa desde X25519 + Kyber768 + info.
func HybridKDF(x25519SS, kyberSS []byte, info string) ([]byte, error) {
	if len(x25519SS) == 0 || len(kyberSS) == 0 {
		return nil, errors.New("empty shared secret")
	}
	combined := append(append([]byte{}, x25519SS...), kyberSS...)
	hkdfReader := hkdf.New(sha256.New, combined, nil, []byte(info))
	key := make([]byte, AESKeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}
	return key, nil
}

// ═══════════════════════════════════════════════════════════════
// AEAD por capa
// ═══════════════════════════════════════════════════════════════

// LayerCipher encapsula AES-256-GCM.
type LayerCipher struct {
	key   []byte
	aead  cipher.AEAD
	hopIdx HopIndex
}

// NewLayerCipher crea un cifrador de capa desde una clave de 32 bytes.
func NewLayerCipher(layerKey []byte, hopIdx HopIndex) (*LayerCipher, error) {
	if len(layerKey) != AESKeySize {
		return nil, fmt.Errorf("invalid key size: %d", len(layerKey))
	}
	block, err := aes.NewCipher(layerKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return &LayerCipher{
		key:    append([]byte{}, layerKey...),
		aead:   aead,
		hopIdx: hopIdx,
	}, nil
}

// Encrypt cifra plaintext. Devuelve nonce || ciphertext.
func (lc *LayerCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	ct := lc.aead.Seal(nil, nonce, plaintext, nil)
	out := append(nonce, ct...)
	return out, nil
}

// Decrypt descifra data. Espera nonce || ciphertext.
func (lc *LayerCipher) Decrypt(data []byte) ([]byte, error) {
	if len(data) < GCMNonceSize+GCMTagSize {
		return nil, errors.New("data too short")
	}
	nonce := data[:GCMNonceSize]
	ct := data[GCMNonceSize:]
	pt, err := lc.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return pt, nil
}

// HopIndex devuelve el índice del hop.
func (lc *LayerCipher) HopIndex() HopIndex {
	return lc.hopIdx
}

// Key devuelve la clave (para tests).
func (lc *LayerCipher) Key() []byte {
	return lc.key
}
