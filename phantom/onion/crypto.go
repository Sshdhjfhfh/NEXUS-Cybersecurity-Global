// Nexus IA v6.0 — Phantom Onion — Criptografía por capa
//
// Cada capa del circuito onion usa AES-256-GCM para cifrar su payload.
// El cifrado es autenticado (AEAD) — detecta tampering.
//
// Propiedades:
//   - AES-256-GCM (AEAD)
//   - Nonce único por celda (evita reutilización)
//   - Clave por capa (32 bytes)
//   - MAC implícito (autenticación)
package onion

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// AESKeySize es el tamaño de clave AES-256 (32 bytes).
	AESKeySize = 32

	// GCMNonceSize es el tamaño del nonce GCM (12 bytes).
	GCMNonceSize = 12

	// GCMTagSize es el tamaño del tag de autenticación (16 bytes).
	GCMTagSize = 16
)

// ═══════════════════════════════════════════════════════════════
// LayerCipher
// ═══════════════════════════════════════════════════════════════

// LayerCipher encapsula el cifrado AES-256-GCM de una capa.
type LayerCipher struct {
	key       []byte
	aead      cipher.AEAD
	hopIndex  HopIndex
}

// NewLayerCipher crea un cifrador de capa desde una clave maestra.
// Deriva la clave de capa desde la clave maestra + hopIndex usando HKDF.
func NewLayerCipher(masterKey []byte, hopIndex HopIndex) (*LayerCipher, error) {
	if len(masterKey) < 32 {
		return nil, errors.New("master key must be at least 32 bytes")
	}

	// Derivar clave de capa con HKDF
	info := []byte(fmt.Sprintf("onion-layer-%d", hopIndex))
	hkdfReader := hkdf.New(sha256.New, masterKey, nil, info)
	layerKey := make([]byte, AESKeySize)
	if _, err := io.ReadFull(hkdfReader, layerKey); err != nil {
		return nil, fmt.Errorf("derive layer key: %w", err)
	}

	// Crear AEAD
	block, err := aes.NewCipher(layerKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	return &LayerCipher{
		key:      layerKey,
		aead:     aead,
		hopIndex: hopIndex,
	}, nil
}

// NewLayerCipherFromKey crea un cifrador desde una clave de capa directa.
func NewLayerCipherFromKey(layerKey []byte, hopIndex HopIndex) (*LayerCipher, error) {
	if len(layerKey) != AESKeySize {
		return nil, fmt.Errorf("layer key must be %d bytes, got %d", AESKeySize, len(layerKey))
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
		key:      layerKey,
		aead:     aead,
		hopIndex: hopIndex,
	}, nil
}

// Encrypt cifra un payload con AES-256-GCM.
// Devuelve nonce || ciphertext.
func (lc *LayerCipher) Encrypt(plaintext []byte) ([]byte, error) {
	// Nonce aleatorio
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Cifrar
	ciphertext := lc.aead.Seal(nil, nonce, plaintext, nil)

	// Devolver nonce || ciphertext
	out := make([]byte, 0, GCMNonceSize+len(ciphertext))
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt descifra un payload con AES-256-GCM.
// Espera nonce || ciphertext.
func (lc *LayerCipher) Decrypt(data []byte) ([]byte, error) {
	if len(data) < GCMNonceSize+GCMTagSize {
		return nil, errors.New("data too short")
	}

	nonce := data[:GCMNonceSize]
	ciphertext := data[GCMNonceSize:]

	plaintext, err := lc.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// HopIndex devuelve el índice del hop de esta capa.
func (lc *LayerCipher) HopIndex() HopIndex {
	return lc.hopIndex
}

// ═══════════════════════════════════════════════════════════════
// MultiLayer — cifrado/descifrado de capas múltiples
// ═══════════════════════════════════════════════════════════════

// MultiLayerCipher maneja todas las capas de un circuito.
type MultiLayerCipher struct {
	layers []*LayerCipher // orden: 0 = outermost, N-1 = innermost
}

// NewMultiLayerCipher crea un cifrador multi-capa desde un master key.
func NewMultiLayerCipher(masterKey []byte, hops int) (*MultiLayerCipher, error) {
	if hops < MinHops || hops > MaxHops {
		return nil, fmt.Errorf("invalid hops: %d", hops)
	}

	layers := make([]*LayerCipher, hops)
	for i := 0; i < hops; i++ {
		lc, err := NewLayerCipher(masterKey, HopIndex(i))
		if err != nil {
			return nil, fmt.Errorf("layer %d: %w", i, err)
		}
		layers[i] = lc
	}

	return &MultiLayerCipher{layers: layers}, nil
}

// Hops devuelve el número de capas.
func (mlc *MultiLayerCipher) Hops() int {
	return len(mlc.layers)
}

// PeelOnion aplica cifrado capa por capa (construcción del onion).
// El orden es desde el más interno (N-1) hasta el más externo (0).
//
// Ejemplo con 3 hops:
//   1. Cifrar con layer[2] (innermost)
//   2. Cifrar con layer[1]
//   3. Cifrar con layer[0] (outermost)
//
// Cuando el primer nodo recibe esto, descifra layer[0] y ve layer[1].
// Luego pasa al segundo nodo, que descifra layer[1] y ve layer[2].
// Luego al tercer nodo, que descifra layer[2] y ve el plaintext.
func (mlc *MultiLayerCipher) PeelOnion(plaintext []byte) ([]byte, error) {
	data := plaintext

	// Cifrar desde la capa más interna (última) hasta la más externa (primera)
	for i := len(mlc.layers) - 1; i >= 0; i-- {
		var err error
		data, err = mlc.layers[i].Encrypt(data)
		if err != nil {
			return nil, fmt.Errorf("encrypt layer %d: %w", i, err)
		}
	}

	return data, nil
}

// UnpeelLayer descifra UNA capa (la más externa).
// Lo usa un relay para procesar una capa.
func (mlc *MultiLayerCipher) UnpeelLayer(data []byte) ([]byte, error) {
	if len(mlc.layers) == 0 {
		return nil, errors.New("no layers")
	}
	return mlc.layers[0].Decrypt(data)
}

// UnpeelAll descifra TODAS las capas (uso interno / tests).
func (mlc *MultiLayerCipher) UnpeelAll(data []byte) ([]byte, error) {
	current := data
	for i := 0; i < len(mlc.layers); i++ {
		var err error
		current, err = mlc.layers[i].Decrypt(current)
		if err != nil {
			return nil, fmt.Errorf("decrypt layer %d: %w", i, err)
		}
	}
	return current, nil
}

// Layer devuelve el cifrador de una capa específica.
func (mlc *MultiLayerCipher) Layer(hop HopIndex) (*LayerCipher, error) {
	if int(hop) < 0 || int(hop) >= len(mlc.layers) {
		return nil, fmt.Errorf("invalid hop index: %d", hop)
	}
	return mlc.layers[hop], nil
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// GenerateMasterKey genera una clave maestra de 32 bytes.
func GenerateMasterKey() ([]byte, error) {
	key := make([]byte, AESKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}
	return key, nil
}
