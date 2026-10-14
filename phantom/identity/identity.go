// Nexus IA v6.0 — Phantom Identity
//
// Pool de identidades sintéticas que el sistema usa para no exponer
// la identidad real. Cada identidad tiene:
//   - User-Agent realista (Chrome, Firefox, Safari, Edge)
//   - Headers HTTP completos (Accept, Accept-Language, etc.)
//   - Fingerprint TLS coherente
//   - Cookies típicas
//   - Geolocalización plausible (timezone, locale)
//
// El PDF lo describe así:
//   'Pool 20 identidades. ShouldRotate() → LLM decide.
//    Fileless: todo en RAM. WipeTraces() al terminar sesión.'
package identity

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// DefaultPoolSize es el tamaño del pool.
	DefaultPoolSize = 20

	// DefaultRotationInterval es cada cuánto rotar.
	DefaultRotationInterval = 10 * time.Minute

	// DefaultMaxUses es cuántas veces usar una identidad antes de rotar.
	DefaultMaxUses = 100

	// DefaultMaxAge es la edad máxima de una identidad.
	DefaultMaxAge = 1 * time.Hour
)

// ═══════════════════════════════════════════════════════════════
// Identidad
// ═══════════════════════════════════════════════════════════════

// Browser clasifica el navegador simulado.
type Browser string

const (
	BrowserChrome  Browser = "Chrome"
	BrowserFirefox Browser = "Firefox"
	BrowserSafari  Browser = "Safari"
	BrowserEdge    Browser = "Edge"
)

// OS clasifica el sistema operativo simulado.
type OS string

const (
	OSWindows OS = "Windows"
	OSMacOS   OS = "macOS"
	OSLinux   OS = "Linux"
	OSiOS     OS = "iOS"
	OSAndroid OS = "Android"
)

// Identity es una identidad sintética.
type Identity struct {
	// ID único
	ID string

	// Navegador y OS
	Browser Browser
	OS      OS

	// User-Agent
	UserAgent string

	// Headers HTTP completos
	Headers map[string]string

	// Geolocalización lógica (NO real)
	Locale       string // ej: "es-PE", "en-US"
	Timezone     string // ej: "America/Lima"
	AcceptLang   string // ej: "es-PE,es;q=0.9,en;q=0.8"

	// Fingerprint TLS (J3/J4 hash simulado)
	TLSFingerprint string

	// Cookies típicas
	Cookies map[string]string

	// Ciclo de vida
	CreatedAt time.Time
	LastUsed  time.Time
	UseCount  int

	// Estado
	ExpiresAt time.Time
	Rotated   bool
}

// Age devuelve la edad de la identidad.
func (i *Identity) Age() time.Duration {
	return time.Since(i.CreatedAt)
}

// IsExpired devuelve true si la identidad expiró.
func (i *Identity) IsExpired() bool {
	return time.Now().After(i.ExpiresAt) || i.Rotated
}

// MarkUsed marca la identidad como usada.
func (i *Identity) MarkUsed() {
	i.LastUsed = time.Now()
	i.UseCount++
}

// ═══════════════════════════════════════════════════════════════
// Builder de identidades
// ═══════════════════════════════════════════════════════════════

// builders por navegador
var userAgents = map[Browser][]string{
	BrowserChrome: {
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	},
	BrowserFirefox: {
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
	},
	BrowserSafari: {
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	},
	BrowserEdge: {
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	},
}

var locales = []string{"es-PE", "en-US", "es-ES", "en-GB", "pt-BR", "fr-FR", "de-DE"}
var timezones = map[string]string{
	"es-PE": "America/Lima",
	"en-US": "America/New_York",
	"es-ES": "Europe/Madrid",
	"en-GB": "Europe/London",
	"pt-BR": "America/Sao_Paulo",
	"fr-FR": "Europe/Paris",
	"de-DE": "Europe/Berlin",
}

// BuildIdentity construye una identidad aleatoria.
func BuildIdentity(rng *mathrand.Rand) *Identity {
	// Elegir navegador
	browsers := []Browser{BrowserChrome, BrowserFirefox, BrowserSafari, BrowserEdge}
	browser := browsers[rng.Intn(len(browsers))]

	// Elegir UA coherente
	agents := userAgents[browser]
	ua := agents[rng.Intn(len(agents))]

	// Detectar OS del UA
	var os OS
	switch {
	case strings.Contains(ua, "Windows"):
		os = OSWindows
	case strings.Contains(ua, "Macintosh"):
		os = OSMacOS
	case strings.Contains(ua, "Linux"):
		os = OSLinux
	default:
		os = OSWindows
	}

	// Elegir locale
	locale := locales[rng.Intn(len(locales))]
	tz := timezones[locale]

	// User-Agent realista
	headers := map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":           buildAcceptLang(locale),
		"Accept-Encoding":           "gzip, deflate, br",
		"DNT":                       "1",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
	}

	// Fingerprint TLS simulado (J3-like)
	tlsFP := randomHex(rng, 16)

	// Cookies típicas
	cookies := map[string]string{
		"_ga":   "GA1.2." + randomDigits(rng, 10) + "." + randomDigits(rng, 10),
		"_gid":  "GA1.2." + randomDigits(rng, 10) + "." + randomDigits(rng, 10),
		"_fbp":  "fb.1." + randomDigits(rng, 13) + "." + randomDigits(rng, 10),
		"_hjSession": randomHex(rng, 16),
	}

	now := time.Now()
	id := &Identity{
		ID:             "id-" + randomHex(rng, 8),
		Browser:        browser,
		OS:             os,
		UserAgent:      ua,
		Headers:        headers,
		Locale:         locale,
		Timezone:       tz,
		AcceptLang:     buildAcceptLang(locale),
		TLSFingerprint: tlsFpOrDefault(tlsFP),
		Cookies:        cookies,
		CreatedAt:      now,
		LastUsed:       now,
		ExpiresAt:      now.Add(DefaultMaxAge),
	}
	return id
}

// buildAcceptLang construye el Accept-Language header.
func buildAcceptLang(locale string) string {
	lang := strings.Split(locale, "-")[0]
	if lang == "en" {
		return locale + ",en;q=0.9"
	}
	return locale + "," + lang + ";q=0.9,en;q=0.8"
}

// tlsFpOrDefault asegura que el fingerprint no esté vacío.
func tlsFpOrDefault(s string) string {
	if s == "" {
		return "chrome-120"
	}
	return s
}

// ═══════════════════════════════════════════════════════════════
// Pool
// ═══════════════════════════════════════════════════════════════

// Pool gestiona el pool de identidades.
type Pool struct {
	mu         sync.RWMutex
	identities map[string]*Identity
	size       int

	// Config
	rotationInterval time.Duration
	maxUses          int
	maxAge           time.Duration

	rng *mathrand.Rand

	// Stats
	totalRotations uint64
	stats          PoolStats
}

// PoolStats lleva estadísticas del pool.
type PoolStats struct {
	CurrentSize    int
	TotalRotations uint64
	TotalUses      uint64
	ByBrowser      map[Browser]int
}

// NewPool crea un pool con N identidades.
func NewPool(size int) *Pool {
	if size <= 0 {
		size = DefaultPoolSize
	}
	p := &Pool{
		identities:       make(map[string]*Identity),
		size:             size,
		rotationInterval: DefaultRotationInterval,
		maxUses:          DefaultMaxUses,
		maxAge:           DefaultMaxAge,
		rng:              mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
	p.fill()
	return p
}

// fill llena el pool con identidades nuevas.
func (p *Pool) fill() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(p.identities) < p.size {
		id := BuildIdentity(p.rng)
		p.identities[id.ID] = id
	}
}

// Get devuelve una identidad disponible del pool.
//
// Heurística:
//   - Prefiere identidades con menos uso
//   - Evita identidades expiradas
//   - Rota si es necesario
func (p *Pool) Get() *Identity {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Buscar la mejor identidad
	var best *Identity
	for _, id := range p.identities {
		if id.IsExpired() {
			continue
		}
		if best == nil || id.UseCount < best.UseCount {
			best = id
		}
	}

	// Si no hay ninguna válida, crear una nueva
	if best == nil {
		id := BuildIdentity(p.rng)
		p.identities[id.ID] = id
		best = id
	}

	best.MarkUsed()
	p.stats.TotalUses++
	return best
}

// ShouldRotate decide si hay que rotar la identidad actual.
// En el PDF esto lo hace el LLM.
func (p *Pool) ShouldRotate(id *Identity) bool {
	if id == nil {
		return true
	}
	if id.IsExpired() {
		return true
	}
	if id.UseCount >= p.maxUses {
		return true
	}
	if id.Age() >= p.maxAge {
		return true
	}
	return false
}

// Rotate marca la identidad como rotada y la reemplaza.
func (p *Pool) Rotate(id *Identity) *Identity {
	if id == nil {
		return p.Get()
	}
	p.mu.Lock()
	id.Rotated = true
	delete(p.identities, id.ID)
	p.totalRotations++
	p.stats.TotalRotations = p.totalRotations

	newID := BuildIdentity(p.rng)
	p.identities[newID.ID] = newID
	p.mu.Unlock()

	return newID
}

// RotateAll rota TODAS las identidades del pool.
func (p *Pool) RotateAll() {
	p.mu.Lock()
	old := p.identities
	p.identities = make(map[string]*Identity, p.size)
	p.totalRotations += uint64(len(old))
	p.stats.TotalRotations = p.totalRotations
	p.mu.Unlock()

	p.fill()
}

// WipeTraces borra todas las identidades (fileless).
func (p *Pool) WipeTraces() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Zero out todas las identidades
	for _, id := range p.identities {
		wipeIdentity(id)
	}
	p.identities = make(map[string]*Identity)
}

// wipeIdentity borra los datos sensibles de una identidad.
func wipeIdentity(id *Identity) {
	if id == nil {
		return
	}
	// Sobrescribir strings con ceros
	id.UserAgent = strings.Repeat("\x00", len(id.UserAgent))
	id.TLSFingerprint = strings.Repeat("\x00", len(id.TLSFingerprint))
	for k := range id.Headers {
		id.Headers[k] = ""
	}
	for k := range id.Cookies {
		id.Cookies[k] = ""
	}
	id.Headers = nil
	id.Cookies = nil
}

// Size devuelve el tamaño actual del pool.
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.identities)
}

// List devuelve todas las identidades.
func (p *Pool) List() []*Identity {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Identity, 0, len(p.identities))
	for _, id := range p.identities {
		out = append(out, id)
	}
	return out
}

// Stats devuelve las estadísticas del pool.
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		CurrentSize:    len(p.identities),
		TotalRotations: p.totalRotations,
		TotalUses:      p.stats.TotalUses,
		ByBrowser:      make(map[Browser]int),
	}
	for _, id := range p.identities {
		stats.ByBrowser[id.Browser]++
	}
	return stats
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

func randomHex(rng *mathrand.Rand, n int) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, n)
	for i := range out {
		out[i] = hexDigits[rng.Intn(16)]
	}
	return string(out)
}

func randomDigits(rng *mathrand.Rand, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte('0' + rng.Intn(10))
	}
	return string(out)
}

// _ asegura que crypto/rand está importado
var _ = rand.Read

// ═══════════════════════════════════════════════════════════════
// Config
// ═══════════════════════════════════════════════════════════════

// Config configura el pool.
type Config struct {
	Size             int
	RotationInterval time.Duration
	MaxUses          int
	MaxAge           time.Duration
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		Size:             DefaultPoolSize,
		RotationInterval: DefaultRotationInterval,
		MaxUses:          DefaultMaxUses,
		MaxAge:           DefaultMaxAge,
	}
}

// NewPoolWithConfig crea un pool con config.
func NewPoolWithConfig(cfg Config) *Pool {
	p := NewPool(cfg.Size)
	p.rotationInterval = cfg.RotationInterval
	p.maxUses = cfg.MaxUses
	p.maxAge = cfg.MaxAge
	return p
}

// _ asegura que fmt está usado
var _ = fmt.Sprintf
