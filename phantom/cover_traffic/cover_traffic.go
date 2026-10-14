// Nexus IA v6.0 — Phantom Cover Traffic
//
// Cover traffic (tráfico de cobertura) son paquetes dummy que el
// sistema envía continuamente para ocultar el tráfico real.
//
// Sin cover traffic:
//   - Un observador ve SILENCIO cuando no hay actividad
//   - Cuando hay actividad, ve un pico → sabe que ALGO pasó
//
// Con cover traffic:
//   - El canal parece tener actividad constante
//   - Es imposible distinguir tráfico real de tráfico dummy
//   - El timing de eventos reales queda oculto
//
// Basado en Loopix (el estado del arte).
//
// El PDF lo describe así:
//   'Cover traffic adaptativo (Loopix). Dummy packets continuos.
//    Rate: recon=50pkt/s · exploit=20pkt/s · idle=5pkt/s.'
package cover_traffic

import (
	"context"
	"errors"
	"math"
	mathrand "math/rand"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// ActivityIdle: sin actividad real → 5 pkt/s de cover
	RateIdle float64 = 5.0

	// ActivityRecon: reconocimiento → 50 pkt/s
	RateRecon float64 = 50.0

	// ActivityExploit: explotación → 20 pkt/s
	RateExploit float64 = 20.0

	// ActivityLateral: movimiento lateral → 30 pkt/s
	RateLateral float64 = 30.0

	// ActivityExfil: exfiltración → 100 pkt/s (máxima cobertura)
	RateExfil float64 = 100.0

	// DefaultAdaptInterval: cada cuánto se recalcula la tasa
	DefaultAdaptInterval = 5 * time.Second

	// JitterFactor: variabilidad del intervalo (0.0-1.0)
	// 0.2 = ±20% alrededor del intervalo base
	JitterFactor = 0.2
)

// ═══════════════════════════════════════════════════════════════
// ActivityLevel
// ═══════════════════════════════════════════════════════════════

// ActivityLevel describe el nivel de actividad real.
type ActivityLevel string

const (
	LevelIdle    ActivityLevel = "IDLE"
	LevelRecon   ActivityLevel = "RECON"
	LevelExploit ActivityLevel = "EXPLOIT"
	LevelLateral ActivityLevel = "LATERAL"
	LevelExfil   ActivityLevel = "EXFIL"
)

// RateForLevel devuelve la tasa de cover traffic para un nivel.
func RateForLevel(level ActivityLevel) float64 {
	switch level {
	case LevelIdle:
		return RateIdle
	case LevelRecon:
		return RateRecon
	case LevelExploit:
		return RateExploit
	case LevelLateral:
		return RateLateral
	case LevelExfil:
		return RateExfil
	default:
		return RateIdle
	}
}

// ═══════════════════════════════════════════════════════════════
// CoverPacket
// ═══════════════════════════════════════════════════════════════

// CoverPacket es un paquete de cobertura.
type CoverPacket struct {
	// Identificador (solo en memoria, no viaja)
	ID string

	// Contenido (dummy, aleatorio)
	Payload []byte

	// Tamaño del packet (fijo, igual que el real)
	Size int

	// Metadatos
	GeneratedAt time.Time
	Level       ActivityLevel
}

// ═══════════════════════════════════════════════════════════════
// Generator
// ═══════════════════════════════════════════════════════════════

// Generator genera cover traffic continuo.
type Generator struct {
	// Configuración
	packetSize     int
	adaptInterval  time.Duration
	jitterFactor   float64

	// Estado
	mu           sync.RWMutex
	level        ActivityLevel
	rate         float64
	enabled      bool
	closed       bool

	// RNG
	rng *mathrand.Rand

	// Callbacks
	onPacket func(*CoverPacket)

	// Stats
	stats GeneratorStats

	// Control
	stopCh  chan struct{}
	stopped chan struct{}
}

// GeneratorStats lleva estadísticas del generador.
type GeneratorStats struct {
	PacketsGenerated uint64
	BytesGenerated   uint64
	RealPackets      uint64 // Contador de paquetes reales (para adaptar)
	LastAdaptAt      time.Time
	CurrentRate      float64
	CurrentLevel     ActivityLevel
}

// Config configura el generador.
type Config struct {
	PacketSize     int
	AdaptInterval  time.Duration
	JitterFactor   float64
	InitialLevel   ActivityLevel
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		PacketSize:    32768, // Igual que Sphinx
		AdaptInterval: DefaultAdaptInterval,
		JitterFactor:  JitterFactor,
		InitialLevel:  LevelIdle,
	}
}

// NewGenerator crea un generador.
func NewGenerator(cfg Config) *Generator {
	if cfg.PacketSize <= 0 {
		cfg.PacketSize = 32768
	}
	if cfg.AdaptInterval <= 0 {
		cfg.AdaptInterval = DefaultAdaptInterval
	}
	if cfg.JitterFactor <= 0 {
		cfg.JitterFactor = JitterFactor
	}
	if cfg.InitialLevel == "" {
		cfg.InitialLevel = LevelIdle
	}

	g := &Generator{
		packetSize:    cfg.PacketSize,
		adaptInterval: cfg.AdaptInterval,
		jitterFactor:  cfg.JitterFactor,
		level:         cfg.InitialLevel,
		rate:          RateForLevel(cfg.InitialLevel),
		enabled:       true,
		rng:           mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
		stopCh:        make(chan struct{}),
		stopped:       make(chan struct{}),
		stats: GeneratorStats{
			CurrentLevel: cfg.InitialLevel,
			CurrentRate:  RateForLevel(cfg.InitialLevel),
		},
	}
	return g
}

// SetCallback establece el callback cuando se genera un paquete.
func (g *Generator) SetCallback(fn func(*CoverPacket)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onPacket = fn
}

// SetLevel cambia el nivel de actividad.
func (g *Generator) SetLevel(level ActivityLevel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.level = level
	g.rate = RateForLevel(level)
	g.stats.CurrentLevel = level
	g.stats.CurrentRate = g.rate
}

// Level devuelve el nivel actual.
func (g *Generator) Level() ActivityLevel {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.level
}

// Rate devuelve la tasa actual.
func (g *Generator) Rate() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.rate
}

// RecordRealPacket registra un paquete real (no cover).
// Ayuda a adaptar la tasa de cover.
func (g *Generator) RecordRealPacket() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.stats.RealPackets++
}

// Start inicia el generador en background.
func (g *Generator) Start() error {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return errors.New("generator closed")
	}
	g.mu.Unlock()

	go g.run()
	return nil
}

// run es el bucle principal del generador.
func (g *Generator) run() {
	defer close(g.stopped)

	adaptTicker := time.NewTicker(g.adaptInterval)
	defer adaptTicker.Stop()

	nextTick := time.Now()
	for {
		select {
		case <-g.stopCh:
			return
		case <-adaptTicker.C:
			g.adapt()
		default:
			// Generar paquete según la tasa actual
			now := time.Now()
			if now.After(nextTick) {
				g.generateOne()
				interval := g.nextInterval()
				nextTick = now.Add(interval)
			} else {
				// Dormir un poco antes del siguiente tick
				sleepFor := time.Until(nextTick)
				if sleepFor > time.Millisecond {
					time.Sleep(time.Millisecond)
				}
			}
		}
	}
}

// nextInterval calcula el intervalo hasta el próximo paquete.
func (g *Generator) nextInterval() time.Duration {
	g.mu.RLock()
	rate := g.rate
	jitter := g.jitterFactor
	g.mu.RUnlock()

	if rate <= 0 {
		rate = 1.0
	}

	base := time.Duration(float64(time.Second) / rate)

	// Aplicar jitter (±jitterFactor)
	g.mu.Lock()
	jitterMult := 1.0 + (g.rng.Float64()*2-1)*jitter
	g.mu.Unlock()

	interval := time.Duration(float64(base) * jitterMult)
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	return interval
}

// generateOne genera un paquete de cobertura.
func (g *Generator) generateOne() {
	g.mu.Lock()
	if !g.enabled || g.closed {
		g.mu.Unlock()
		return
	}

	// Generar payload aleatorio
	payload := make([]byte, g.packetSize)
	_, _ = g.rng.Read(payload)

	// Crear packet
	id := make([]byte, 8)
	_, _ = g.rng.Read(id)

	packet := &CoverPacket{
		ID:          fmtHex(id),
		Payload:     payload,
		Size:        g.packetSize,
		GeneratedAt: time.Now(),
		Level:       g.level,
	}

	// Actualizar stats
	g.stats.PacketsGenerated++
	g.stats.BytesGenerated += uint64(g.packetSize)

	cb := g.onPacket
	g.mu.Unlock()

	// Disparar callback
	if cb != nil {
		cb(packet)
	}
}

// adapt ajusta la tasa según actividad reciente.
//
// Si hay muchos paquetes reales recientes, sube la tasa de cover.
// Si no hay actividad, la baja.
func (g *Generator) adapt() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Calcular actividad reciente
	// Si en el último intervalo hubo >5 paquetes reales, subir nivel
	realRecent := g.stats.RealPackets
	g.stats.RealPackets = 0
	g.stats.LastAdaptAt = time.Now()

	// Heurística simple: adaptar según paquetes reales en el intervalo
	switch {
	case realRecent > 20:
		g.level = LevelExfil
	case realRecent > 10:
		g.level = LevelLateral
	case realRecent > 5:
		g.level = LevelExploit
	case realRecent > 1:
		g.level = LevelRecon
	default:
		g.level = LevelIdle
	}
	g.rate = RateForLevel(g.level)
	g.stats.CurrentLevel = g.level
	g.stats.CurrentRate = g.rate
}

// Stats devuelve las estadísticas del generador.
func (g *Generator) Stats() GeneratorStats {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.stats
}

// Stop detiene el generador.
func (g *Generator) Stop() {
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	g.closed = true
	close(g.stopCh)
	g.mu.Unlock()

	// Esperar a que termine el bucle
	<-g.stopped
}

// Enabled devuelve si el generador está activo.
func (g *Generator) Enabled() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enabled && !g.closed
}

// SetEnabled activa/desactiva el generador.
func (g *Generator) SetEnabled(enabled bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.enabled = enabled
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// fmtHex convierte bytes a string hexadecimal simple.
func fmtHex(b []byte) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexDigits[v>>4]
		out[i*2+1] = hexDigits[v&0x0f]
	}
	return string(out)
}

// PoissonInterval genera un intervalo según distribución Poisson.
// Útil para timing más realista.
func PoissonInterval(rate float64, rng *mathrand.Rand) time.Duration {
	if rate <= 0 {
		rate = 1.0
	}
	u := rng.Float64()
	if u <= 0 {
		u = 0.0001
	}
	// Intervalo = -ln(U) / rate (segundos)
	seconds := -math.Log(u) / rate
	// Limitar a 1ms mínimo
	d := time.Duration(seconds * float64(time.Second))
	if d < time.Millisecond {
		d = time.Millisecond
	}
	return d
}

// contextContext es un alias para no importar context aquí.
type contextContext interface {
	Done() <-chan struct{}
	Err() error
}

// StartWithContext inicia el generador con cancelación por contexto.
func (g *Generator) StartWithContext(ctx contextContext) error {
	err := g.Start()
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		g.Stop()
	}()
	return nil
}

// _ asegura que context no está sin usar
var _ = context.Background
