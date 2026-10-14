// Nexus IA v6.0 — Phantom VPN — Orquestador principal
//
// Une crypto + handshake + pool en un cliente VPN completo.
//
// Flujo:
//   1. Seleccionar nodo del pool (SelectBest)
//   2. Handshake post-cuántico híbrido con el nodo
//   3. Derivar session key
//   4. Establecer "túnel" lógico (sin WireGuard aún)
//   5. Cifrar/descifrar paquetes con ChaCha20-Poly1305
//   6. Rotar session key cada N segundos
//   7. Cerrar y limpiar
//
// Esta capa es agnóstica al transporte real (WireGuard, TCP, UDP).
// El transporte real viene en wireguard.go.
package vpn

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Configuración
// ═══════════════════════════════════════════════════════════════

// Config configura el cliente VPN.
type Config struct {
	// Pool de nodos
	Pool *Pool

	// Duración máxima de sesión (rotación automática)
	SessionTTL time.Duration

	// Timeout de handshake
	HandshakeTimeout time.Duration

	// ¿Rotar claves automáticamente?
	AutoRotate bool

	// Número máximo de reintentos al conectar
	MaxRetries int
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		SessionTTL:       SessionKeyTTL,
		HandshakeTimeout: HandshakeTimeout,
		AutoRotate:       true,
		MaxRetries:       3,
	}
}

// ═══════════════════════════════════════════════════════════════
// Session
// ═══════════════════════════════════════════════════════════════

// Session es una sesión VPN activa.
type Session struct {
	// IDs
	ID     string
	NodeID string

	// Estado
	Handshake  *HandshakeState
	AEAD       *AEAD
	SessionKey []byte

	// Timing
	CreatedAt time.Time
	ExpiresAt time.Time

	// Contadores
	BytesSent     uint64
	BytesReceived uint64
	PacketsSent   uint64
	PacketsReceived uint64

	// Estado
	mu       sync.RWMutex
	closed   bool
}

// IsExpired devuelve true si la sesión ha expirado.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsClosed devuelve true si la sesión está cerrada.
func (s *Session) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// TTL devuelve el tiempo restante de la sesión.
func (s *Session) TTL() time.Duration {
	if s.IsExpired() {
		return 0
	}
	return time.Until(s.ExpiresAt)
}

// Encrypt cifra plaintext con la session key.
func (s *Session) Encrypt(plaintext, additionalData []byte) (nonce, ciphertext []byte, err error) {
	if s.IsClosed() {
		return nil, nil, errors.New("session closed")
	}
	if s.IsExpired() {
		return nil, nil, errors.New("session expired")
	}

	nonce, err = GenerateNonce()
	if err != nil {
		return nil, nil, err
	}

	ciphertext, err = s.AEAD.Encrypt(nonce, plaintext, additionalData)
	if err != nil {
		return nil, nil, err
	}

	s.mu.Lock()
	s.BytesSent += uint64(len(ciphertext))
	s.PacketsSent++
	s.mu.Unlock()

	return nonce, ciphertext, nil
}

// Decrypt descifra ciphertext con la session key.
func (s *Session) Decrypt(nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if s.IsClosed() {
		return nil, errors.New("session closed")
	}
	if s.IsExpired() {
		return nil, errors.New("session expired")
	}

	plaintext, err := s.AEAD.Decrypt(nonce, ciphertext, additionalData)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.BytesReceived += uint64(len(ciphertext))
	s.PacketsReceived++
	s.mu.Unlock()

	return plaintext, nil
}

// Close cierra la sesión y limpia claves.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	// Limpiar claves (zero out)
	for i := range s.SessionKey {
		s.SessionKey[i] = 0
	}
	s.SessionKey = nil
	s.AEAD = nil
	s.Handshake = nil
}

// Stats devuelve estadísticas de la sesión.
func (s *Session) Stats() SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SessionStats{
		ID:              s.ID,
		NodeID:          s.NodeID,
		CreatedAt:       s.CreatedAt,
		ExpiresAt:       s.ExpiresAt,
		TTL:             s.TTL(),
		BytesSent:       s.BytesSent,
		BytesReceived:   s.BytesReceived,
		PacketsSent:     s.PacketsSent,
		PacketsReceived: s.PacketsReceived,
		Closed:          s.closed,
	}
}

// SessionStats lleva las estadísticas de una sesión.
type SessionStats struct {
	ID              string
	NodeID          string
	CreatedAt       time.Time
	ExpiresAt       time.Time
	TTL             time.Duration
	BytesSent       uint64
	BytesReceived   uint64
	PacketsSent     uint64
	PacketsReceived uint64
	Closed          bool
}

// ═══════════════════════════════════════════════════════════════
// Client (orquestador)
// ═══════════════════════════════════════════════════════════════

// Client es el cliente VPN de alto nivel.
type Client struct {
	cfg Config

	mu       sync.RWMutex
	sessions map[string]*Session
	pool     *Pool

	// Sesión activa (la más reciente)
	activeSessionID string

	// Lifecycle
	closed bool
}

// NewClient crea un cliente VPN con un pool.
func NewClient(cfg Config, pool *Pool) *Client {
	if pool == nil {
		pool = NewPool()
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = SessionKeyTTL
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = HandshakeTimeout
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}

	return &Client{
		cfg:      cfg,
		sessions: make(map[string]*Session),
		pool:     pool,
	}
}

// Pool devuelve el pool de nodos.
func (c *Client) Pool() *Pool {
	return c.pool
}

// Connect establece una sesión VPN con el mejor nodo disponible.
//
// Flujo:
//   1. Seleccionar mejor nodo del pool
//   2. Crear handshake initiator
//   3. Procesar initiation en el nodo (simulado localmente)
//   4. Crear response
//   5. Finalizar handshake
//   6. Crear sesión con AEAD
func (c *Client) Connect() (*Session, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, errors.New("client closed")
	}
	c.mu.Unlock()

	// 1. Seleccionar nodo
	node := c.pool.SelectBest()
	if node == nil {
		return nil, errors.New("no usable node in pool")
	}

	// 2. Handshake
	var session *Session
	var lastErr error

	for attempt := 0; attempt < c.cfg.MaxRetries; attempt++ {
		s, err := c.performHandshake(node)
		if err == nil {
			session = s
			break
		}
		lastErr = err
		c.pool.RecordFailure(node.ID)
		// Pequeña espera antes de reintentar
		time.Sleep(100 * time.Millisecond)
	}

	if session == nil {
		return nil, fmt.Errorf("handshake failed after %d attempts: %w", c.cfg.MaxRetries, lastErr)
	}

	// 3. Guardar sesión
	c.mu.Lock()
	c.sessions[session.ID] = session
	c.activeSessionID = session.ID
	c.mu.Unlock()

	c.pool.RecordConnection(node.ID)

	return session, nil
}

// performHandshake ejecuta el handshake completo con un nodo.
//
// NOTA: En esta implementación, el "nodo" actúa localmente como
// responder para simular el handshake completo. En producción,
// esto se haría por red.
func (c *Client) performHandshake(node *Node) (*Session, error) {
	// 1. Crear handshake initiator
	initiator, err := NewInitiatorHandshake()
	if err != nil {
		return nil, fmt.Errorf("create initiator: %w", err)
	}

	// 2. Crear mensaje 1 (initiation)
	initMsg, err := initiator.CreateInitiation()
	if err != nil {
		return nil, fmt.Errorf("create initiation: %w", err)
	}

	// 3. Simular responder (en producción, esto sería el nodo remoto)
	responder, err := NewResponderHandshake()
	if err != nil {
		return nil, fmt.Errorf("create responder: %w", err)
	}

	// 4. Procesar initiation en el responder
	if err := responder.ProcessInitiation(initMsg); err != nil {
		return nil, fmt.Errorf("process initiation: %w", err)
	}

	// 5. Crear response
	respMsg, err := responder.CreateResponse()
	if err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	// 6. Finalizar en el responder → devuelve el kyber ciphertext
	responderCiphertext, err := responder.FinalizeResponder()
	if err != nil {
		return nil, fmt.Errorf("finalize responder: %w", err)
	}

	// 7. Procesar response en el initiator
	if err := initiator.ProcessResponse(respMsg); err != nil {
		return nil, fmt.Errorf("process response: %w", err)
	}

	// 8. Finalizar en el initiator → deriva session key
	if err := initiator.FinalizeInitiator(responderCiphertext); err != nil {
		return nil, fmt.Errorf("finalize initiator: %w", err)
	}

	// Verificar que ambos derivaron la misma clave (simetría)
	if !bytesEqual(initiator.SessionKey, responder.SessionKey) {
		return nil, errors.New("session key mismatch between initiator and responder")
	}

	// 9. Crear AEAD con la session key
	aead, err := NewAEAD(initiator.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("create aead: %w", err)
	}

	// 10. Crear sesión
	now := time.Now()
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	session := &Session{
		ID:         sessionID,
		NodeID:     node.ID,
		Handshake:  initiator,
		AEAD:       aead,
		SessionKey: append([]byte{}, initiator.SessionKey...),
		CreatedAt:  now,
		ExpiresAt:  now.Add(c.cfg.SessionTTL),
	}

	return session, nil
}

// ActiveSession devuelve la sesión activa.
func (c *Client) ActiveSession() *Session {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.activeSessionID == "" {
		return nil
	}
	return c.sessions[c.activeSessionID]
}

// GetSession recupera una sesión por ID.
func (c *Client) GetSession(id string) (*Session, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	s, ok := c.sessions[id]
	return s, ok
}

// ListSessions devuelve todas las sesiones.
func (c *Client) ListSessions() []*Session {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]*Session, 0, len(c.sessions))
	for _, s := range c.sessions {
		out = append(out, s)
	}
	return out
}

// CloseSession cierra una sesión específica.
func (c *Client) CloseSession(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, ok := c.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	s.Close()
	delete(c.sessions, id)
	if c.activeSessionID == id {
		c.activeSessionID = ""
	}
	return nil
}

// Close cierra el cliente y todas las sesiones.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	for _, s := range c.sessions {
		s.Close()
	}
	c.sessions = make(map[string]*Session)
	c.activeSessionID = ""
	c.closed = true
	return nil
}

// SessionCount devuelve el número de sesiones activas.
func (c *Client) SessionCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.sessions)
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// generateSessionID genera un ID único de sesión.
func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := readRandom(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("vpn-session-%x", b), nil
}

// bytesEqual compara dos slices de bytes en tiempo constante.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
