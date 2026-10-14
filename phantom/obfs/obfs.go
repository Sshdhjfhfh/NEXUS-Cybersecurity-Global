// Nexus IA v6.0 — Phantom Obfuscators
//
// Obfuscadores de tráfico que hacen que el tráfico de Phantom
// parezca tráfico legítimo:
//
//   - HTTPSObfuscator: simula tráfico HTTPS normal (parece CDN)
//   - DNSObfuscator:  mete datos en queries DNS
//   - TLSObfuscator:  simula fingerprint TLS de Chrome/Firefox
//
// El PDF lo describe así:
//   'HTTPSObfuscator (parece CDN) · DNSObfuscator (datos en DNS)
//    TLSObfuscator (fingerprint Chrome/Firefox). LLM elige según contexto.'
package obfs

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ═══════════════════════════════════════════════════════════════
// Tipos base
// ═══════════════════════════════════════════════════════════════

// Obfuscator transforma datos en tráfico camuflado.
type Obfuscator interface {
	// Name devuelve el nombre del obfuscador.
	Name() string

	// Wrap envuelve un payload en tráfico camuflado.
	Wrap(payload []byte) ([]byte, error)

	// Unwrap extrae el payload de tráfico camuflado.
	Unwrap(wrapped []byte) ([]byte, error)

	// Overhead devuelve el overhead en bytes que añade.
	Overhead() int
}

// Type clasifica el tipo de obfuscator.
type Type string

const (
	TypeHTTPS Type = "HTTPS"
	TypeDNS   Type = "DNS"
	TypeTLS   Type = "TLS"
)

// ═══════════════════════════════════════════════════════════════
// HTTPS Obfuscator
// ═══════════════════════════════════════════════════════════════

// HTTPSObfuscator hace que el tráfico parezca HTTPS normal.
type HTTPSObfuscator struct {
	// HeaderPrefix simula headers HTTP reales.
	HeaderPrefix []byte
}

// NewHTTPSObfuscator crea un obfuscator HTTPS.
func NewHTTPSObfuscator() *HTTPSObfuscator {
	return &HTTPSObfuscator{
		HeaderPrefix: []byte("HTTP/1.1 200 OK\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"Content-Length: "),
	}
}

func (o *HTTPSObfuscator) Name() string { return "HTTPS" }
func (o *HTTPSObfuscator) Overhead() int { return len(o.HeaderPrefix) + 32 }

// Wrap envuelve payload como body HTTP.
func (o *HTTPSObfuscator) Wrap(payload []byte) ([]byte, error) {
	// Header: HTTP/1.1 200 OK + Content-Length
	out := make([]byte, 0, len(o.HeaderPrefix)+32+len(payload)+32)
	out = append(out, o.HeaderPrefix...)

	// Content-Length
	lenBuf := make([]byte, 20)
	n := binary.PutUvarint(lenBuf, uint64(len(payload)))
	out = append(out, lenBuf[:n]...)
	out = append(out, []byte("\r\n\r\n")...)

	// Body
	out = append(out, payload...)

	// Footer: trailing headers
	out = append(out, []byte("\r\n0\r\n\r\n")...)

	return out, nil
}

// Unwrap extrae el payload del body HTTP.
func (o *HTTPSObfuscator) Unwrap(wrapped []byte) ([]byte, error) {
	// Buscar \r\n\r\n que separa headers del body
	idx := findSequence(wrapped, []byte("\r\n\r\n"))
	if idx < 0 {
		return nil, errors.New("invalid HTTP wrapper: no header/body separator")
	}
	body := wrapped[idx+4:]
	// Quitar footer
	footerIdx := findSequence(body, []byte("\r\n0\r\n\r\n"))
	if footerIdx > 0 {
		body = body[:footerIdx]
	}
	return body, nil
}

// ═══════════════════════════════════════════════════════════════
// DNS Obfuscator
// ═══════════════════════════════════════════════════════════════

// DNSObfuscator mete datos en queries DNS.
type DNSObfuscator struct {
	// BaseDomain es el dominio base para las queries.
	BaseDomain string
}

// NewDNSObfuscator crea un obfuscator DNS.
func NewDNSObfuscator(baseDomain string) *DNSObfuscator {
	if baseDomain == "" {
		baseDomain = "cdn.example.com"
	}
	return &DNSObfuscator{BaseDomain: baseDomain}
}

func (o *DNSObfuscator) Name() string { return "DNS" }
func (o *DNSObfuscator) Overhead() int { return 12 }

// Wrap mete datos en una query DNS.
func (o *DNSObfuscator) Wrap(payload []byte) ([]byte, error) {
	// Formato: [2 bytes length][payload base32][.basedomain]
	out := make([]byte, 0, 2+len(payload)*2+len(o.BaseDomain)+1)

	// Length
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(payload)))
	out = append(out, lenBuf...)

	// Payload (encoded simple: byte + 'A')
	for _, b := range payload {
		out = append(out, 'A'+b%26)
	}

	return out, nil
}

// Unwrap extrae el payload.
func (o *DNSObfuscator) Unwrap(wrapped []byte) ([]byte, error) {
	if len(wrapped) < 2 {
		return nil, errors.New("DNS wrapper too short")
	}
	length := int(binary.BigEndian.Uint16(wrapped[:2]))
	if length+2 > len(wrapped) {
		return nil, errors.New("DNS wrapper length mismatch")
	}
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		out[i] = wrapped[2+i] - 'A'
	}
	return out, nil
}

// ═══════════════════════════════════════════════════════════════
// TLS Obfuscator
// ═══════════════════════════════════════════════════════════════

// TLSObfuscator simula un handshake TLS.
type TLSObfuscator struct {
	// Fingerprint: "chrome" o "firefox"
	Fingerprint string
}

// NewTLSObfuscator crea un obfuscator TLS.
func NewTLSObfuscator(fingerprint string) *TLSObfuscator {
	if fingerprint == "" {
		fingerprint = "chrome"
	}
	return &TLSObfuscator{Fingerprint: fingerprint}
}

func (o *TLSObfuscator) Name() string { return "TLS" }
func (o *TLSObfuscator) Overhead() int { return 64 }

// Wrap simula un record TLS.
func (o *TLSObfuscator) Wrap(payload []byte) ([]byte, error) {
	// TLS record header (5 bytes): type(1) + version(2) + length(2)
	out := make([]byte, 0, 5+len(payload)+32)

	// ContentType: 0x17 (application_data)
	out = append(out, 0x17)

	// Version: 0x0303 (TLS 1.2)
	out = append(out, 0x03, 0x03)

	// Length
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(payload)))
	out = append(out, lenBuf...)

	// Payload
	out = append(out, payload...)

	// Padding aleatorio (simula cifrado TLS)
	padding := make([]byte, 16)
	_, _ = rand.Read(padding)
	out = append(out, padding...)

	return out, nil
}

// Unwrap extrae el payload del record TLS.
func (o *TLSObfuscator) Unwrap(wrapped []byte) ([]byte, error) {
	if len(wrapped) < 5 {
		return nil, errors.New("TLS wrapper too short")
	}
	if wrapped[0] != 0x17 {
		return nil, fmt.Errorf("not an application_data record: 0x%02x", wrapped[0])
	}
	length := int(binary.BigEndian.Uint16(wrapped[3:5]))
	if 5+length > len(wrapped) {
		return nil, errors.New("TLS wrapper length mismatch")
	}
	return wrapped[5 : 5+length], nil
}

// ═══════════════════════════════════════════════════════════════
// Manager
// ═══════════════════════════════════════════════════════════════

// Manager gestiona los obfuscadores disponibles.
type Manager struct {
	obfuscators map[Type]Obfuscator
	current     Type
}

// NewManager crea un manager con los obfuscadores por defecto.
func NewManager() *Manager {
	return &Manager{
		obfuscators: map[Type]Obfuscator{
			TypeHTTPS: NewHTTPSObfuscator(),
			TypeDNS:   NewDNSObfuscator("cdn.example.com"),
			TypeTLS:   NewTLSObfuscator("chrome"),
		},
		current: TypeTLS,
	}
}

// Get devuelve un obfuscator por tipo.
func (m *Manager) Get(t Type) (Obfuscator, bool) {
	o, ok := m.obfuscators[t]
	return o, ok
}

// Current devuelve el obfuscator activo.
func (m *Manager) Current() Obfuscator {
	return m.obfuscators[m.current]
}

// SetCurrent cambia el obfuscator activo.
func (m *Manager) SetCurrent(t Type) error {
	if _, ok := m.obfuscators[t]; !ok {
		return fmt.Errorf("unknown obfuscator: %s", t)
	}
	m.current = t
	return nil
}

// SelectByContext elige un obfuscator según el contexto.
// En el PDF esto lo hace el LLM.
func (m *Manager) SelectByContext(context string) Obfuscator {
	// Heurística simple: HTTPS para web, DNS para stealth, TLS por defecto
	switch context {
	case "web", "browsing":
		return m.obfuscators[TypeHTTPS]
	case "stealth", "recon":
		return m.obfuscators[TypeDNS]
	default:
		return m.obfuscators[TypeTLS]
	}
}

// List devuelve todos los tipos disponibles.
func (m *Manager) List() []Type {
	out := make([]Type, 0, len(m.obfuscators))
	for t := range m.obfuscators {
		out = append(out, t)
	}
	return out
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// findSequence busca una subsecuencia en b.
func findSequence(b, sub []byte) int {
	if len(sub) == 0 || len(b) < len(sub) {
		return -1
	}
	for i := 0; i <= len(b)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// _ asegura que io no está sin usar
var _ = io.EOF
