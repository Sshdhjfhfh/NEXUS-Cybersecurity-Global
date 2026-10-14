// Nexus IA v6.0 — Phantom VPN — Handshake post-cuántico híbrido
//
// Implementa un handshake inspirado en Noise IK pero con:
//   - X25519 + Kyber768 híbrido (post-cuántico)
//   - Anti-replay (timestamps + nonces)
//   - Perfect Forward Secrecy (claves efímeras por sesión)
//   - Rotación de claves cada N segundos
//
// Flujo del handshake (2 mensajes):
//
//   1. INITIATOR → RESPONDER:
//      - X25519 pub efímera
//      - Kyber768 ciphertext
//      - Nonce + Timestamp
//      - (opcional) Firma del initiator con Dilithium
//
//   2. RESPONDER → INITIATOR:
//      - X25519 pub efímera
//      - Nonce + Timestamp
//      - (opcional) Firma del responder con Dilithium
//
//   Ambos derivan session_key = HKDF(X25519_ss || Kyber768_ss)
package vpn

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/cloudflare/circl/kem/kyber/kyber768"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// HandshakeTimeout es el tiempo máximo para completar un handshake.
	HandshakeTimeout = 10 * time.Second

	// SessionKeyTTL es la vida útil de una clave de sesión.
	// Rotación automática tras este tiempo.
	SessionKeyTTL = 2 * time.Minute

	// MaxTimestampDrift es la ventana máxima de tolerancia para timestamps.
	// Protección anti-replay.
	MaxTimestampDrift = 30 * time.Second

	// HandshakeTypeInitiation es el tipo de mensaje 1.
	HandshakeTypeInitiation = 1

	// HandshakeTypeResponse es el tipo de mensaje 2.
	HandshakeTypeResponse = 2
)

// ═══════════════════════════════════════════════════════════════
// Mensajes
// ═══════════════════════════════════════════════════════════════

// HandshakeInitiation es el primer mensaje del handshake.
type HandshakeInitiation struct {
	Type              uint8
	SenderIndex       uint32    // Índice aleatorio del initiator
	X25519PublicKey   [X25519KeySize]byte
	Kyber768Ciphertext []byte   // Ciphertext del KEM encapsulado
	Nonce             [12]byte  // Anti-replay
	Timestamp         int64     // Unix nanos
	Signature         []byte    // Firma Dilithium (opcional)
}

// HandshakeResponse es el segundo mensaje del handshake.
type HandshakeResponse struct {
	Type              uint8
	SenderIndex       uint32    // Índice del initiator (eco)
	ReceiverIndex     uint32    // Índice aleatorio del responder
	X25519PublicKey   [X25519KeySize]byte
	Nonce             [12]byte
	Timestamp         int64
	Signature         []byte    // Firma Dilithium (opcional)
}

// HandshakeState mantiene el estado del handshake en curso.
type HandshakeState struct {
	// Configuración
	Initiator bool // ¿Somos el initiator?

	// Claves locales
	LocalX25519    *X25519KeyPair
	LocalKyber     *Kyber768KeyPair
	LocalIndex     uint32

	// Claves del peer
	PeerX25519PublicKey [X25519KeySize]byte
	PeerIndex           uint32
	peerKyberPub        *kyber768.PublicKey

	// Estado intermedio
	SentInitiation  bool
	SentResponse    bool
	LastTimestamp   int64
	SeenNonces      map[[12]byte]bool // Anti-replay

	// Resultado
	SessionKey   []byte
	SessionKeyAt time.Time
}

// ═══════════════════════════════════════════════════════════════
// Initiation
// ═══════════════════════════════════════════════════════════════

// NewInitiatorHandshake crea un HandshakeState para iniciar.
func NewInitiatorHandshake() (*HandshakeState, error) {
	x, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate x25519: %w", err)
	}
	k, err := GenerateKyber768KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate kyber768: %w", err)
	}

	idx, err := randomUint32()
	if err != nil {
		return nil, fmt.Errorf("generate index: %w", err)
	}

	return &HandshakeState{
		Initiator:    true,
		LocalX25519:  x,
		LocalKyber:   k,
		LocalIndex:   idx,
		SeenNonces:   make(map[[12]byte]bool),
	}, nil
}

// NewResponderHandshake crea un HandshakeState para responder.
func NewResponderHandshake() (*HandshakeState, error) {
	x, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate x25519: %w", err)
	}
	k, err := GenerateKyber768KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate kyber768: %w", err)
	}

	idx, err := randomUint32()
	if err != nil {
		return nil, fmt.Errorf("generate index: %w", err)
	}

	return &HandshakeState{
		Initiator:    false,
		LocalX25519:  x,
		LocalKyber:   k,
		LocalIndex:   idx,
		SeenNonces:   make(map[[12]byte]bool),
	}, nil
}

// CreateInitiation crea el mensaje 1 del handshake.
// Solo puede llamarlo el initiator.
func (hs *HandshakeState) CreateInitiation() (*HandshakeInitiation, error) {
	if !hs.Initiator {
		return nil, errors.New("only initiator can create initiation")
	}
	if hs.SentInitiation {
		return nil, errors.New("initiation already sent")
	}

	// Encapsular un secret con Kyber768 usando nuestra clave pública
	// (el peer lo desencapsulará con nuestra clave privada... espera,
	// esto está mal. El peer encapsula con NUESTRA pubkey y nosotros
	// desencapsulamos con nuestra privkey.)
	//
	// CORRECTO: el initiator genera un par efímero y envía la PUBLIC key.
	// El responder encapsula con esa pubkey y envía el ciphertext.
	// El initiator desencapsula con su privkey.
	//
	// Por tanto, en Initiation enviamos la pubkey de Kyber768, NO el ciphertext.
	//
	// Vamos a corregirlo en el struct Initiation:
	//   Kyber768Ciphertext → Kyber768PublicKey

	nonce, err := generateNonce12()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	hs.SentInitiation = true
	hs.LastTimestamp = time.Now().UnixNano()

	// Serializar la pubkey de Kyber768
	kyberPubBytes := make([]byte, Kyber768PublicKeySize)
	hs.LocalKyber.PublicKey.Pack(kyberPubBytes)

	msg := &HandshakeInitiation{
		Type:               HandshakeTypeInitiation,
		SenderIndex:        hs.LocalIndex,
		X25519PublicKey:    hs.LocalX25519.PublicKey,
		Kyber768Ciphertext: kyberPubBytes, // NOTA: aquí va la PUBKEY, no ciphertext
		Nonce:              nonce,
		Timestamp:          hs.LastTimestamp,
	}

	return msg, nil
}

// ProcessInitiation procesa el mensaje 1 en el responder.
func (hs *HandshakeState) ProcessInitiation(msg *HandshakeInitiation) error {
	if hs.Initiator {
		return errors.New("initiator cannot process initiation")
	}
	if msg == nil {
		return errors.New("nil initiation message")
	}
	if msg.Type != HandshakeTypeInitiation {
		return fmt.Errorf("invalid message type: %d", msg.Type)
	}

	// Verificar timestamp (anti-replay)
	if err := hs.verifyTimestamp(msg.Timestamp); err != nil {
		return err
	}

	// Verificar nonce único (anti-replay)
	if hs.SeenNonces[msg.Nonce] {
		return errors.New("replay detected: nonce already seen")
	}
	hs.SeenNonces[msg.Nonce] = true

	// Guardar claves del peer
	hs.PeerX25519PublicKey = msg.X25519PublicKey
	hs.PeerIndex = msg.SenderIndex

	// Deserializar la pubkey de Kyber768 del peer
	if len(msg.Kyber768Ciphertext) != Kyber768PublicKeySize {
		return fmt.Errorf("invalid kyber768 pubkey size: %d", len(msg.Kyber768Ciphertext))
	}
	peerKyberPub := new(kyber768.PublicKey)
	peerKyberPub.Unpack(msg.Kyber768Ciphertext)
	hs.peerKyberPub = peerKyberPub

	return nil
}

// CreateResponse crea el mensaje 2 del handshake.
func (hs *HandshakeState) CreateResponse() (*HandshakeResponse, error) {
	if hs.Initiator {
		return nil, errors.New("initiator cannot create response")
	}
	if hs.SentResponse {
		return nil, errors.New("response already sent")
	}
	if hs.peerKyberPub == nil {
		return nil, errors.New("no peer kyber pubkey (process initiation first)")
	}

	nonce, err := generateNonce12()
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	hs.SentResponse = true
	hs.LastTimestamp = time.Now().UnixNano()

	msg := &HandshakeResponse{
		Type:            HandshakeTypeResponse,
		SenderIndex:     hs.PeerIndex,
		ReceiverIndex:   hs.LocalIndex,
		X25519PublicKey: hs.LocalX25519.PublicKey,
		Nonce:           nonce,
		Timestamp:       hs.LastTimestamp,
	}

	return msg, nil
}

// ProcessResponse procesa el mensaje 2 en el initiator.
func (hs *HandshakeState) ProcessResponse(msg *HandshakeResponse) error {
	if !hs.Initiator {
		return errors.New("responder cannot process response")
	}
	if msg == nil {
		return errors.New("nil response message")
	}
	if msg.Type != HandshakeTypeResponse {
		return fmt.Errorf("invalid message type: %d", msg.Type)
	}
	if msg.SenderIndex != hs.LocalIndex {
		return fmt.Errorf("wrong sender index: %d, expected %d", msg.SenderIndex, hs.LocalIndex)
	}

	// Verificar timestamp
	if err := hs.verifyTimestamp(msg.Timestamp); err != nil {
		return err
	}

	// Verificar nonce
	if hs.SeenNonces[msg.Nonce] {
		return errors.New("replay detected: nonce already seen")
	}
	hs.SeenNonces[msg.Nonce] = true

	// Guardar claves del peer
	hs.PeerX25519PublicKey = msg.X25519PublicKey
	hs.PeerIndex = msg.ReceiverIndex

	return nil
}

// FinalizeInitiator completa el handshake en el initiator.
func (hs *HandshakeState) FinalizeInitiator(responderKyberCiphertext []byte) error {
	if !hs.Initiator {
		return errors.New("only initiator can finalize as initiator")
	}

	// X25519 shared secret
	x25519SS, err := hs.LocalX25519.SharedSecret(hs.PeerX25519PublicKey)
	if err != nil {
		return fmt.Errorf("x25519 shared: %w", err)
	}

	// Kyber768 shared secret (desencapsular el ciphertext del responder)
	kyberSS, err := hs.LocalKyber.Decapsulate(responderKyberCiphertext)
	if err != nil {
		return fmt.Errorf("kyber768 decaps: %w", err)
	}

	// HKDF híbrido
	sessionKey, err := HybridSharedSecret(x25519SS, kyberSS, "phantom-vpn-session")
	if err != nil {
		return fmt.Errorf("hybrid shared secret: %w", err)
	}

	hs.SessionKey = sessionKey
	hs.SessionKeyAt = time.Now()
	return nil
}

// FinalizeResponder completa el handshake en el responder.
// Devuelve el ciphertext de Kyber768 que debe enviarse al initiator.
func (hs *HandshakeState) FinalizeResponder() ([]byte, error) {
	if hs.Initiator {
		return nil, errors.New("only responder can finalize as responder")
	}
	if hs.peerKyberPub == nil {
		return nil, errors.New("no peer kyber pubkey")
	}

	// X25519 shared secret
	x25519SS, err := hs.LocalX25519.SharedSecret(hs.PeerX25519PublicKey)
	if err != nil {
		return nil, fmt.Errorf("x25519 shared: %w", err)
	}

	// Encapsular un secret con la pubkey del initiator
	ciphertext := make([]byte, kyber768.CiphertextSize)
	kyberSS := make([]byte, Kyber768SharedSecretSize)
	hs.peerKyberPub.EncapsulateTo(ciphertext, kyberSS, nil)

	// HKDF híbrido
	sessionKey, err := HybridSharedSecret(x25519SS, kyberSS, "phantom-vpn-session")
	if err != nil {
		return nil, fmt.Errorf("hybrid shared secret: %w", err)
	}

	hs.SessionKey = sessionKey
	hs.SessionKeyAt = time.Now()

	return ciphertext, nil
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// verifyTimestamp comprueba que el timestamp esté dentro de la ventana.
func (hs *HandshakeState) verifyTimestamp(ts int64) error {
	now := time.Now().UnixNano()
	drift := now - ts
	if drift < 0 {
		drift = -drift
	}
	if drift > int64(MaxTimestampDrift) {
		return fmt.Errorf("timestamp drift too large: %v", time.Duration(drift))
	}
	return nil
}

// randomUint32 genera un uint32 aleatorio.
func randomUint32() (uint32, error) {
	b := make([]byte, 4)
	if _, err := readRandom(b); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

// generateNonce12 genera un nonce de 12 bytes.
func generateNonce12() ([12]byte, error) {
	var n [12]byte
	if _, err := readRandom(n[:]); err != nil {
		return n, err
	}
	return n, nil
}

// readRandom llena b con bytes aleatorios.
func readRandom(b []byte) (int, error) {
	return rand.Read(b)
}
