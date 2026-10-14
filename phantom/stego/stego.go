// Nexus IA v6.0 — Phantom Steganography
//
// Esteganografía: esconder datos en tráfico/datos aparentemente
// inocentes. Técnicas implementadas:
//
//   1. LSB en PNG:      bits menos significativos de píxeles
//   2. Headers HTTP:    campos custom en headers
//   3. Timing TCP:      inter-packet delays codifican bits
//   4. Cookies:         cookies HTTP con payload
//   5. Metadata CDN:    headers de CDN como Cache-Control
//
// El PDF lo describe así:
//   'LSB en PNG, headers HTTP, timing TCP, cookies, metadata CDN.
//    Datos del agente viajan invisibles en HTTPS legítimo.'
package stego

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// ═══════════════════════════════════════════════════════════════
// Tipos base
// ═══════════════════════════════════════════════════════════════

// Technique clasifica las técnicas disponibles.
type Technique string

const (
	TechLSB      Technique = "LSB_PNG"
	TechHTTPHdr  Technique = "HTTP_HEADERS"
	TechTiming   Technique = "TIMING_TCP"
	TechCookies  Technique = "COOKIES"
	TechCDNMeta  Technique = "CDN_METADATA"
)

// Encoder transforma datos en un canal encubierto.
type Encoder interface {
	// Name devuelve el nombre de la técnica.
	Name() string

	// Capacity devuelve la capacidad en bytes del contenedor.
	Capacity(container []byte) int

	// Encode esconde payload en container.
	Encode(payload, container []byte) ([]byte, error)

	// Decode extrae payload del container.
	Decode(container []byte) ([]byte, error)
}

// ═══════════════════════════════════════════════════════════════
// 1. LSB en PNG
// ═══════════════════════════════════════════════════════════════

// LSBEncoder esconde datos en los bits LSB de un contenedor.
type LSBEncoder struct {
	// Delimiter: bytes que marcan el final del payload.
	Delimiter []byte
}

// NewLSBEncoder crea un encoder LSB.
func NewLSBEncoder() *LSBEncoder {
	return &LSBEncoder{
		Delimiter: []byte{0x00, 0xFF, 0x00, 0xFF},
	}
}

func (e *LSBEncoder) Name() string { return "LSB_PNG" }

func (e *LSBEncoder) Capacity(container []byte) int {
	// 1 bit por byte del container
	return len(container) / 8
}

// Encode escribe payload en los LSBs.
//
// Formato: [4 bytes length BE][payload][delimiter][padding random]
func (e *LSBEncoder) Encode(payload, container []byte) ([]byte, error) {
	// Preparar mensaje: length + payload + delimiter
	msg := make([]byte, 0, 4+len(payload)+len(e.Delimiter))
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))
	msg = append(msg, lenBuf...)
	msg = append(msg, payload...)
	msg = append(msg, e.Delimiter...)

	// Comprobar capacidad
	maxBytes := e.Capacity(container)
	if len(msg) > maxBytes {
		return nil, fmt.Errorf("payload too large: %d > %d", len(msg), maxBytes)
	}

	// Copiar container
	out := make([]byte, len(container))
	copy(out, container)

	// Escribir bits en LSBs
	bitIdx := 0
	for _, b := range msg {
		for bit := 7; bit >= 0; bit-- {
			if bitIdx >= len(out) {
				break
			}
			// Extraer bit
			val := (b >> uint(bit)) & 1
			// Escribir en LSB
			out[bitIdx] = (out[bitIdx] & 0xFE) | val
			bitIdx++
		}
	}

	// Rellenar LSBs restantes con bits aleatorios (para no dejar patrones)
	for ; bitIdx < len(out); bitIdx++ {
		var r [1]byte
		_, _ = rand.Read(r[:])
		out[bitIdx] = (out[bitIdx] & 0xFE) | (r[0] & 1)
	}

	return out, nil
}

// Decode extrae payload de los LSBs.
func (e *LSBEncoder) Decode(container []byte) ([]byte, error) {
	if len(container) < 4*8 {
		return nil, errors.New("container too small")
	}

	// Leer bytes bit a bit
	var msg []byte
	var current byte
	bitCount := 0

	for i := 0; i < len(container); i++ {
		bit := container[i] & 1
		current = (current << 1) | bit
		bitCount++

		if bitCount == 8 {
			msg = append(msg, current)
			current = 0
			bitCount = 0
		}
	}

	// Extraer payload
	if len(msg) < 4 {
		return nil, errors.New("no payload length")
	}
	length := int(binary.BigEndian.Uint32(msg[:4]))
	if length < 0 || 4+length > len(msg) {
		return nil, errors.New("invalid payload length")
	}

	return msg[4 : 4+length], nil
}

// ═══════════════════════════════════════════════════════════════
// 2. HTTP Headers
// ═══════════════════════════════════════════════════════════════

// HTTPHeadersEncoder esconde datos en headers HTTP.
type HTTPHeadersEncoder struct {
	// HeaderNames: nombres de headers permitidos para esconder datos.
	HeaderNames []string
}

// NewHTTPHeadersEncoder crea un encoder.
func NewHTTPHeadersEncoder() *HTTPHeadersEncoder {
	return &HTTPHeadersEncoder{
		HeaderNames: []string{
			"X-Cache-Key",
			"X-Request-ID",
			"X-CDN-Edge",
			"X-Trace-ID",
		},
	}
}

func (e *HTTPHeadersEncoder) Name() string { return "HTTP_HEADERS" }

func (e *HTTPHeadersEncoder) Capacity(container []byte) int {
	// Aproximado: 32 bytes por header
	return len(e.HeaderNames) * 32
}

// Encode esconde payload en headers.
//
// Formato: cada header contiene un chunk base64-like.
func (e *HTTPHeadersEncoder) Encode(payload, container []byte) ([]byte, error) {
	if len(payload) == 0 {
		return container, nil
	}

	// Convertir a string base64 seguro
	encoded := encodeBase64Safe(payload)

	// Dividir en chunks de 32 bytes
	chunks := splitChunks(encoded, 32)

	// Añadir headers al container (que debería ser una request/response HTTP)
	out := make([]byte, 0, len(container)+len(chunks)*64)
	out = append(out, container...)

	for i, chunk := range chunks {
		if i >= len(e.HeaderNames) {
			break
		}
		header := fmt.Sprintf("%s: %s\r\n", e.HeaderNames[i], chunk)
		out = append(out, []byte(header)...)
	}

	return out, nil
}

// Decode extrae payload de headers.
func (e *HTTPHeadersEncoder) Decode(container []byte) ([]byte, error) {
	text := string(container)
	var chunks []string

	for _, name := range e.HeaderNames {
		prefix := name + ": "
		idx := strings.Index(text, prefix)
		if idx < 0 {
			continue
		}
		lineStart := idx + len(prefix)
		lineEnd := strings.Index(text[lineStart:], "\r\n")
		if lineEnd < 0 {
			continue
		}
		chunk := text[lineStart : lineStart+lineEnd]
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return nil, errors.New("no stego headers found")
	}

	// Concatenar chunks
	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString(c)
	}

	return decodeBase64Safe(sb.String())
}

// ═══════════════════════════════════════════════════════════════
// 3. Timing TCP
// ═══════════════════════════════════════════════════════════════

// TimingEncoder codifica datos en inter-packet delays.
type TimingEncoder struct {
	// Threshold: duración mínima (ms) para considerar un bit "1".
	ThresholdMs int

	// BaseDelayMs: delay base entre packets.
	BaseDelayMs int
}

// NewTimingEncoder crea un encoder.
func NewTimingEncoder() *TimingEncoder {
	return &TimingEncoder{
		ThresholdMs: 100,
		BaseDelayMs: 50,
	}
}

func (e *TimingEncoder) Name() string { return "TIMING_TCP" }

func (e *TimingEncoder) Capacity(container []byte) int {
	// 1 bit por packet. Asumimos que container es la lista de delays.
	return len(container)
}

// Encode convierte payload en un array de delays.
//
// Cada byte se convierte en 8 delays (uno por bit).
// Delay = BaseDelayMs + (bit * (ThresholdMs - BaseDelayMs))
func (e *TimingEncoder) Encode(payload, container []byte) ([]byte, error) {
	// Generar array de delays como uint16 (ms)
	delays := make([]uint16, 0, len(payload)*8)
	for _, b := range payload {
		for bit := 7; bit >= 0; bit-- {
			val := (b >> uint(bit)) & 1
			var delay uint16
			if val == 1 {
				delay = uint16(e.ThresholdMs)
			} else {
				delay = uint16(e.BaseDelayMs)
			}
			delays = append(delays, delay)
		}
	}

	// Serializar como bytes BE
	out := make([]byte, len(delays)*2)
	for i, d := range delays {
		binary.BigEndian.PutUint16(out[i*2:], d)
	}
	return out, nil
}

// Decode extrae payload de los delays.
func (e *TimingEncoder) Decode(container []byte) ([]byte, error) {
	if len(container)%2 != 0 {
		return nil, errors.New("invalid timing data")
	}

	numDelays := len(container) / 2
	var payload []byte
	var current byte
	bitCount := 0

	for i := 0; i < numDelays; i++ {
		delay := binary.BigEndian.Uint16(container[i*2:])
		var bit byte
		if int(delay) >= e.ThresholdMs {
			bit = 1
		}
		current = (current << 1) | bit
		bitCount++
		if bitCount == 8 {
			payload = append(payload, current)
			current = 0
			bitCount = 0
		}
	}

	return payload, nil
}

// ═══════════════════════════════════════════════════════════════
// 4. Cookies
// ═══════════════════════════════════════════════════════════════

// CookiesEncoder esconde datos en cookies HTTP.
type CookiesEncoder struct {
	CookieNames []string
}

// NewCookiesEncoder crea un encoder.
func NewCookiesEncoder() *CookiesEncoder {
	return &CookiesEncoder{
		CookieNames: []string{
			"_ga",
			"_gid",
			"_fbp",
			"session_hint",
		},
	}
}

func (e *CookiesEncoder) Name() string { return "COOKIES" }

func (e *CookiesEncoder) Capacity(container []byte) int {
	return len(e.CookieNames) * 40
}

// Encode esconde payload en cookies.
func (e *CookiesEncoder) Encode(payload, container []byte) ([]byte, error) {
	if len(payload) == 0 {
		return container, nil
	}

	encoded := encodeBase64Safe(payload)
	chunks := splitChunks(encoded, 40)

	out := make([]byte, 0, len(container)+len(chunks)*64)
	out = append(out, container...)

	for i, chunk := range chunks {
		if i >= len(e.CookieNames) {
			break
		}
		cookie := fmt.Sprintf("Set-Cookie: %s=%s; Path=/; HttpOnly\r\n",
			e.CookieNames[i], chunk)
		out = append(out, []byte(cookie)...)
	}

	return out, nil
}

// Decode extrae payload de cookies.
func (e *CookiesEncoder) Decode(container []byte) ([]byte, error) {
	text := string(container)
	var chunks []string

	for _, name := range e.CookieNames {
		prefix := name + "="
		idx := strings.Index(text, prefix)
		if idx < 0 {
			continue
		}
		valStart := idx + len(prefix)
		valEnd := strings.IndexAny(text[valStart:], ";\r\n")
		if valEnd < 0 {
			valEnd = len(text) - valStart
		}
		chunks = append(chunks, text[valStart:valStart+valEnd])
	}

	if len(chunks) == 0 {
		return nil, errors.New("no stego cookies")
	}

	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString(c)
	}
	return decodeBase64Safe(sb.String())
}

// ═══════════════════════════════════════════════════════════════
// 5. CDN Metadata
// ═══════════════════════════════════════════════════════════════

// CDNMetadataEncoder esconde datos en headers CDN típicos.
type CDNMetadataEncoder struct {
	HeaderNames []string
}

// NewCDNMetadataEncoder crea un encoder.
func NewCDNMetadataEncoder() *CDNMetadataEncoder {
	return &CDNMetadataEncoder{
		HeaderNames: []string{
			"Cache-Control",
			"ETag",
			"Last-Modified",
			"X-Cache-Status",
		},
	}
}

func (e *CDNMetadataEncoder) Name() string { return "CDN_METADATA" }

func (e *CDNMetadataEncoder) Capacity(container []byte) int {
	return len(e.HeaderNames) * 32
}

// Encode esconde payload en headers CDN.
func (e *CDNMetadataEncoder) Encode(payload, container []byte) ([]byte, error) {
	if len(payload) == 0 {
		return container, nil
	}

	encoded := encodeBase64Safe(payload)
	chunks := splitChunks(encoded, 32)

	out := make([]byte, 0, len(container)+len(chunks)*64)
	out = append(out, container...)

	// Formatos realistas de CDN
	for i, chunk := range chunks {
		if i >= len(e.HeaderNames) {
			break
		}
		var line string
		switch e.HeaderNames[i] {
		case "Cache-Control":
			line = fmt.Sprintf("Cache-Control: max-age=%s\r\n", chunk)
		case "ETag":
			line = fmt.Sprintf("ETag: \"%s\"\r\n", chunk)
		case "Last-Modified":
			line = fmt.Sprintf("Last-Modified: %s\r\n", chunk)
		case "X-Cache-Status":
			line = fmt.Sprintf("X-Cache-Status: HIT-%s\r\n", chunk)
		default:
			line = fmt.Sprintf("%s: %s\r\n", e.HeaderNames[i], chunk)
		}
		out = append(out, []byte(line)...)
	}

	return out, nil
}

// Decode extrae payload de headers CDN.
func (e *CDNMetadataEncoder) Decode(container []byte) ([]byte, error) {
	text := string(container)
	var chunks []string

	for _, name := range e.HeaderNames {
		prefix := name + ": "
		idx := strings.Index(text, prefix)
		if idx < 0 {
			continue
		}
		lineStart := idx + len(prefix)
		lineEnd := strings.Index(text[lineStart:], "\r\n")
		if lineEnd < 0 {
			continue
		}
		raw := text[lineStart : lineStart+lineEnd]

		// Limpiar según el formato
		chunk := raw
		switch name {
		case "Cache-Control":
			chunk = strings.TrimPrefix(raw, "max-age=")
		case "ETag":
			chunk = strings.Trim(raw, "\"")
		case "Last-Modified":
			chunk = raw
		case "X-Cache-Status":
			chunk = strings.TrimPrefix(raw, "HIT-")
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return nil, errors.New("no stego CDN metadata")
	}

	var sb strings.Builder
	for _, c := range chunks {
		sb.WriteString(c)
	}
	return decodeBase64Safe(sb.String())
}

// ═══════════════════════════════════════════════════════════════
// Manager
// ═══════════════════════════════════════════════════════════════

// Manager gestiona los encoders disponibles.
type Manager struct {
	encoders map[Technique]Encoder
}

// NewManager crea un manager con todos los encoders.
func NewManager() *Manager {
	return &Manager{
		encoders: map[Technique]Encoder{
			TechLSB:     NewLSBEncoder(),
			TechHTTPHdr: NewHTTPHeadersEncoder(),
			TechTiming:  NewTimingEncoder(),
			TechCookies: NewCookiesEncoder(),
			TechCDNMeta: NewCDNMetadataEncoder(),
		},
	}
}

// Get devuelve un encoder.
func (m *Manager) Get(t Technique) (Encoder, bool) {
	e, ok := m.encoders[t]
	return e, ok
}

// List devuelve las técnicas disponibles.
func (m *Manager) List() []Technique {
	out := make([]Technique, 0, len(m.encoders))
	for t := range m.encoders {
		out = append(out, t)
	}
	return out
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// encodeBase64Safe codifica usando solo caracteres alfanuméricos.
// Menos eficiente que base64 pero más seguro para headers.
func encodeBase64Safe(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	var sb strings.Builder
	for _, b := range data {
		sb.WriteByte(alphabet[b>>2])
		sb.WriteByte(alphabet[(b&0x03)<<4])
	}
	return sb.String()
}

// decodeBase64Safe decodifica la codificación anterior.
func decodeBase64Safe(s string) ([]byte, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	table := [128]int{}
	for i := range table {
		table[i] = -1
	}
	for i, c := range alphabet {
		table[c] = i
	}

	if len(s)%2 != 0 {
		return nil, errors.New("invalid encoded length")
	}

	out := make([]byte, 0, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		c1 := table[s[i]]
		c2 := table[s[i+1]]
		if c1 < 0 || c2 < 0 {
			return nil, fmt.Errorf("invalid char at %d", i)
		}
		out = append(out, byte(c1<<2|c2>>4))
	}
	return out, nil
}

// splitChunks divide s en chunks de tamaño size.
func splitChunks(s string, size int) []string {
	if size <= 0 {
		size = 32
	}
	var out []string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}
