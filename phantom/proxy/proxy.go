// Nexus IA v6.0 — Phantom Proxy (SOCKS5 puro)
//
// SOCKS5 TCP server. An explicit transport is required; no direct fallback.
// Network guarantees depend on the configured transport.
//
// Características:
//   - SOCKS5 completo (RFC 1928 + RFC 1929 auth)
//   - Circuit breaker a >1000 req/s
//   - Traffic padding entre peticiones
//   - Compatible con cualquier cliente SOCKS5
//
// El PDF lo describe así:
//
//	'SOCKS5 puro en Go. Todo tráfico → onion → mixnet.
//	 Circuit breaker >1000 req/s. Traffic padding entre peticiones.'
package proxy

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Constantes SOCKS5
// ═══════════════════════════════════════════════════════════════

const (
	// SOCKS5 version
	SOCKS5Version = 0x05

	// Auth methods
	AuthNone     = 0x00
	AuthUserPass = 0x02
	AuthNoAccept = 0xFF

	// Commands
	CmdConnect = 0x01
	CmdBind    = 0x02
	CmdUDP     = 0x03

	// Address types
	AddrIPv4   = 0x01
	AddrDomain = 0x03
	AddrIPv6   = 0x04

	// Reply codes
	RepSuccess          = 0x00
	RepGeneralFailure   = 0x01
	RepConnNotAllowed   = 0x02
	RepNetworkUnreach   = 0x03
	RepHostUnreach      = 0x04
	RepConnRefused      = 0x05
	RepTTLExpired       = 0x06
	RepCmdNotSupported  = 0x07
	RepAddrNotSupported = 0x08
)

// ═══════════════════════════════════════════════════════════════
// Configuración
// ═══════════════════════════════════════════════════════════════

// Config configura el proxy SOCKS5.
type Config struct {
	// Dirección de escucha (ej: "127.0.0.1:1080")
	ListenAddr string

	// Usuario y contraseña (opcional, si vacío = no auth)
	Username string
	Password string

	// Rate limiting
	MaxRequestsPerSecond int
	CircuitBreakerWindow time.Duration

	// Traffic padding
	EnablePadding   bool
	PaddingMinBytes int
	PaddingMaxBytes int

	// Dialer: cómo conectar al destino
	// Puede ser una función custom (onion, mixnet, vpn, etc.)
	Dialer Dialer

	// Timeouts
	HandshakeTimeout time.Duration
	ConnectTimeout   time.Duration
	IdleTimeout      time.Duration
	MaxConnections   int
}

// Dialer es la interfaz para conectar a un destino.
// Permite customizar el transporte (onion, mixnet, directo, etc.)
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		ListenAddr:           "127.0.0.1:1080",
		MaxRequestsPerSecond: 1000,
		CircuitBreakerWindow: 1 * time.Second,
		EnablePadding:        false, // Transport owns padding; never corrupt application bytes.
		PaddingMinBytes:      16,
		PaddingMaxBytes:      256,
		HandshakeTimeout:     10 * time.Second,
		ConnectTimeout:       30 * time.Second,
		IdleTimeout:          5 * time.Minute,
		MaxConnections:       256,
	}
}

// ═══════════════════════════════════════════════════════════════
// Proxy
// ═══════════════════════════════════════════════════════════════

// Proxy es un servidor SOCKS5.
type Proxy struct {
	cfg      Config
	listener net.Listener

	// Rate limiting
	requestCount atomic.Uint64
	lastReset    atomic.Int64

	// Estado
	mu     sync.RWMutex
	closed bool
	conns  map[net.Conn]struct{}

	// Stats
	stats ProxyStats
}

// ProxyStats lleva estadísticas del proxy.
type ProxyStats struct {
	TotalConnections   uint64
	ActiveConnections  int
	TotalBytesUp       uint64
	TotalBytesDown     uint64
	FailedConnections  uint64
	RateLimitedCount   uint64
	CircuitBrokenCount uint64
}

// NewProxy crea un proxy SOCKS5.
func NewProxy(cfg Config) *Proxy {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:1080"
	}
	if cfg.MaxRequestsPerSecond <= 0 {
		cfg.MaxRequestsPerSecond = 1000
	}
	if cfg.CircuitBreakerWindow <= 0 {
		cfg.CircuitBreakerWindow = 1 * time.Second
	}

	if cfg.MaxConnections <= 0 {
		cfg.MaxConnections = 256
	}
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 10 * time.Second
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 5 * time.Minute
	}
	p := &Proxy{
		cfg:   cfg,
		conns: make(map[net.Conn]struct{}),
	}
	p.lastReset.Store(time.Now().UnixNano())
	return p
}

// Start inicia el proxy.
func (p *Proxy) Start() error {
	if p.cfg.Dialer == nil {
		return errors.New("explicit transport required: direct egress is disabled")
	}
	if p.cfg.EnablePadding {
		return errors.New("proxy padding is unsupported: configure padding on the encrypted transport")
	}
	if (p.cfg.Username == "") != (p.cfg.Password == "") || len(p.cfg.Username) > 255 || len(p.cfg.Password) > 255 {
		return errors.New("invalid SOCKS credentials")
	}
	ln, err := net.Listen("tcp", p.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	p.listener = ln

	go p.acceptLoop()
	return nil
}

// Addr devuelve la dirección de escucha.
func (p *Proxy) Addr() net.Addr {
	if p.listener == nil {
		return nil
	}
	return p.listener.Addr()
}

// acceptLoop acepta conexiones entrantes.
func (p *Proxy) acceptLoop() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			p.mu.RLock()
			closed := p.closed
			p.mu.RUnlock()
			if closed {
				return
			}
			continue
		}

		// Rate limiting
		if !p.allowRequest() {
			_ = conn.Close()
			continue
		}

		// Track connection
		p.mu.Lock()
		if p.closed || len(p.conns) >= p.cfg.MaxConnections {
			p.mu.Unlock()
			_ = conn.Close()
			continue
		}
		p.conns[conn] = struct{}{}
		p.stats.TotalConnections++
		p.stats.ActiveConnections = len(p.conns)
		p.mu.Unlock()

		go p.handleConnection(conn)
	}
}

// allowRequest verifica si se permite la petición (rate limit).
func (p *Proxy) allowRequest() bool {
	now := time.Now().UnixNano()
	lastReset := p.lastReset.Load()
	window := int64(p.cfg.CircuitBreakerWindow)

	// Reset contador si pasó la ventana
	if now-lastReset > window {
		if p.lastReset.CompareAndSwap(lastReset, now) {
			p.requestCount.Store(0)
		}
	}

	count := p.requestCount.Add(1)
	limit := uint64(p.cfg.MaxRequestsPerSecond)

	if count > limit {
		p.mu.Lock()
		p.stats.RateLimitedCount++
		// Circuit breaker: si excede 2x el límite, activar circuit breaker
		if count > limit*2 {
			p.stats.CircuitBrokenCount++
		}
		p.mu.Unlock()
		return false
	}
	return true
}

// handleConnection maneja una conexión SOCKS5.
func (p *Proxy) handleConnection(conn net.Conn) {
	defer func() {
		_ = conn.Close()
		p.mu.Lock()
		delete(p.conns, conn)
		p.stats.ActiveConnections = len(p.conns)
		p.mu.Unlock()
	}()

	// Handshake
	targetAddr, err := p.handshake(conn)
	if err != nil {
		p.mu.Lock()
		p.stats.FailedConnections++
		p.mu.Unlock()
		return
	}

	// Conectar al destino usando el Dialer (onion, mixnet, etc.)
	dialer := p.cfg.Dialer

	target, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		p.sendReply(conn, RepHostUnreach)
		p.mu.Lock()
		p.stats.FailedConnections++
		p.mu.Unlock()
		return
	}
	defer func() { _ = target.Close() }()

	// Enviar reply OK
	if err := p.sendReply(conn, RepSuccess); err != nil {
		return
	}

	// Relay bidireccional
	p.relay(conn, target)
}

// handshake ejecuta el handshake SOCKS5.
func (p *Proxy) handshake(conn net.Conn) (string, error) {
	if p.cfg.HandshakeTimeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(p.cfg.HandshakeTimeout))
		defer func() { _ = conn.SetDeadline(time.Time{}) }()
	}

	// 1. Greeting: VER + NMETHODS + METHODS
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", fmt.Errorf("read header: %w", err)
	}
	if header[0] != SOCKS5Version {
		return "", fmt.Errorf("unsupported version: %d", header[0])
	}
	nMethods := int(header[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", fmt.Errorf("read methods: %w", err)
	}

	// 2. Elegir método
	wantAuth := p.cfg.Username != "" || p.cfg.Password != ""
	accepted := false
	for _, m := range methods {
		if wantAuth && m == AuthUserPass {
			accepted = true
			break
		}
		if !wantAuth && m == AuthNone {
			accepted = true
			break
		}
	}

	if !accepted {
		_, _ = conn.Write([]byte{SOCKS5Version, AuthNoAccept})
		return "", errors.New("client does not support required auth")
	}

	if _, err := conn.Write([]byte{SOCKS5Version, methodByte(wantAuth)}); err != nil {
		return "", fmt.Errorf("write method: %w", err)
	}

	// 3. Auth (si aplica)
	if wantAuth {
		if err := p.doAuth(conn); err != nil {
			return "", err
		}
	}

	// 4. Request: VER + CMD + RSV + ATYP + ADDR + PORT
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return "", fmt.Errorf("read request: %w", err)
	}
	if req[0] != SOCKS5Version || req[2] != 0 {
		return "", errors.New("bad version in request")
	}
	cmd := req[1]
	atyp := req[3]

	if cmd != CmdConnect {
		_ = p.sendReply(conn, RepCmdNotSupported)
		return "", fmt.Errorf("unsupported command: %d", cmd)
	}

	// Leer dirección
	addr, err := readAddress(conn, atyp)
	if err != nil {
		_ = p.sendReply(conn, RepAddrNotSupported)
		return "", err
	}

	return addr, nil
}

// doAuth ejecuta la autenticación usuario/contraseña (RFC 1929).
func (p *Proxy) doAuth(conn net.Conn) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return fmt.Errorf("read auth header: %w", err)
	}
	if header[0] != 0x01 {
		return errors.New("unsupported auth version")
	}

	uLen := int(header[1])
	username := make([]byte, uLen)
	if _, err := io.ReadFull(conn, username); err != nil {
		return err
	}

	pLenBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, pLenBuf); err != nil {
		return err
	}
	pLen := int(pLenBuf[0])
	password := make([]byte, pLen)
	if _, err := io.ReadFull(conn, password); err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(username, []byte(p.cfg.Username))&subtle.ConstantTimeCompare(password, []byte(p.cfg.Password)) != 1 {
		_, _ = conn.Write([]byte{0x01, 0x01})
		return errors.New("invalid credentials")
	}

	_, err := conn.Write([]byte{0x01, 0x00})
	return err
}

// sendReply envía un reply SOCKS5.
func (p *Proxy) sendReply(conn net.Conn, code byte) error {
	reply := []byte{SOCKS5Version, code, 0x00, AddrIPv4, 0, 0, 0, 0, 0, 0}
	_, err := conn.Write(reply)
	return err
}

// relay reenvía datos bidireccionalmente.
// activityConn refreshes both directions' idle deadlines on successful I/O.
type activityConn struct {
	net.Conn
	peer net.Conn
	idle time.Duration
}

func (c activityConn) touch() {
	deadline := time.Now().Add(c.idle)
	_ = c.Conn.SetDeadline(deadline)
	_ = c.peer.SetDeadline(deadline)
}
func (c activityConn) Read(b []byte) (int, error) {
	n, e := c.Conn.Read(b)
	if n > 0 {
		c.touch()
	}
	return n, e
}
func (c activityConn) Write(b []byte) (int, error) {
	n, e := c.Conn.Write(b)
	if n > 0 {
		c.touch()
	}
	return n, e
}
func (p *Proxy) relay(a, b net.Conn) {
	left := activityConn{a, b, p.cfg.IdleTimeout}
	right := activityConn{b, a, p.cfg.IdleTimeout}
	left.touch()
	type result struct {
		up bool
		n  int64
	}
	done := make(chan result, 2)
	copySide := func(dst activityConn, src net.Conn, up bool) {
		n, err := io.Copy(dst, src)
		if err != nil {
			_ = a.Close()
			_ = b.Close()
		} else if cw, ok := dst.Conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		} else {
			_ = dst.Close()
		}
		done <- result{up, n}
	}
	go copySide(right, left, true)
	go copySide(left, right, false)
	first, second := <-done, <-done
	p.mu.Lock()
	for _, r := range []result{first, second} {
		if r.up {
			p.stats.TotalBytesUp += uint64(r.n)
		} else {
			p.stats.TotalBytesDown += uint64(r.n)
		}
	}
	p.mu.Unlock()
}

// Stop detiene el proxy.
func (p *Proxy) Stop() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	ln := p.listener
	conns := make([]net.Conn, 0, len(p.conns))
	for c := range p.conns {
		conns = append(conns, c)
	}
	p.mu.Unlock()

	if ln != nil {
		_ = ln.Close()
	}
	for _, c := range conns {
		_ = c.Close()
	}
	return nil
}

// Stats devuelve las estadísticas del proxy.
func (p *Proxy) Stats() ProxyStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// methodByte devuelve el byte del método según si se usa auth.
func methodByte(wantAuth bool) byte {
	if wantAuth {
		return AuthUserPass
	}
	return AuthNone
}

// readAddress lee una dirección SOCKS5.
func readAddress(conn net.Conn, atyp byte) (string, error) {
	switch atyp {
	case AddrIPv4:
		buf := make([]byte, 4+2)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		ip := net.IP(buf[:4]).String()
		port := int(buf[4])<<8 | int(buf[5])
		return fmt.Sprintf("%s:%d", ip, port), nil

	case AddrDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		if lenBuf[0] == 0 {
			return "", errors.New("empty destination")
		}
		domain := make([]byte, int(lenBuf[0]))
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		portBuf := make([]byte, 2)
		if _, err := io.ReadFull(conn, portBuf); err != nil {
			return "", err
		}
		port := int(portBuf[0])<<8 | int(portBuf[1])
		return fmt.Sprintf("%s:%d", string(domain), port), nil

	case AddrIPv6:
		buf := make([]byte, 16+2)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return "", err
		}
		ip := net.IP(buf[:16]).String()
		port := int(buf[16])<<8 | int(buf[17])
		return fmt.Sprintf("[%s]:%d", ip, port), nil

	default:
		return "", fmt.Errorf("unsupported address type: %d", atyp)
	}
}
