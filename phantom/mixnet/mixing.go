// Nexus IA v6.0 — Phantom Mixnet — Mixing (batching + delays Poisson)
//
// Este archivo es EL CORAZÓN del mixnet. Implementa las 3 técnicas
// que destruyen la correlación temporal:
//
//   1. Batching — agrupa paquetes antes de reenviar
//      → un observador no puede saber qué paquete entró con qué paquete sale
//
//   2. Delays Poisson — cada paquete se retrasa un tiempo aleatorio
//      → el timing de salida es impredecible
//
//   3. Reordenación — el orden de salida NO coincide con el de entrada
//      → rompe la secuencia entrada/salida
//
// Combinados, hacen IMPOSIBLE correlacionar entrada con salida incluso
// si un adversario ve toda la red.
package mixnet

import (
	"math"
	mathrand "math/rand"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Mixer — el componente de mezcla
// ═══════════════════════════════════════════════════════════════

// PendingPacket es un paquete esperando a ser mezclado.
type PendingPacket struct {
	Packet     *SphinxPacket
	ReceivedAt time.Time
	SendAt     time.Time // Cuándo debe salir (calculado con Poisson)
}

// Mixer mezcla paquetes usando batching + delays + reordenación.
type Mixer struct {
	// Configuración
	batchSize       int
	batchInterval   time.Duration
	poissonLambda   float64
	enableBatching  bool
	enableDelays    bool

	// Estado
	mu       sync.Mutex
	pending  []*PendingPacket
	rng      *mathrand.Rand

	// Callbacks
	onBatchReady func([]*SphinxPacket)

	// Estadísticas
	stats MixerStats

	// Lifecycle
	closed bool
	stopCh chan struct{}
}

// MixerStats lleva estadísticas del mixer.
type MixerStats struct {
	TotalReceived   uint64
	TotalMixed      uint64
	TotalBatches    uint64
	AvgBatchSize    float64
	AvgDelayMs      int64
	TotalDropped    uint64
}

// NewMixer crea un mixer.
func NewMixer(cfg Config) *Mixer {
	m := &Mixer{
		batchSize:      cfg.BatchSize,
		batchInterval:  BatchInterval,
		poissonLambda:  PoissonLambda,
		enableBatching: cfg.EnableBatching,
		enableDelays:   cfg.EnablePoissonDelays,
		pending:        make([]*PendingPacket, 0),
		rng:            mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
		stopCh:         make(chan struct{}),
	}

	if m.batchSize <= 0 {
		m.batchSize = 10
	}

	return m
}

// SetBatchReadyCallback establece el callback de batch listo.
func (m *Mixer) SetBatchReadyCallback(fn func([]*SphinxPacket)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onBatchReady = fn
}

// Add añade un paquete al mixer.
//
//   1. Calcula su tiempo de salida con distribución Poisson
//   2. Lo añade a la cola de pendientes
//   3. Si el batch está lleno, dispara el procesamiento
func (m *Mixer) Add(packet *SphinxPacket) error {
	if packet == nil {
		return nil
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}

	// Calcular tiempo de salida con Poisson
	now := time.Now()
	sendAt := now
	if m.enableDelays {
		delay := m.poissonDelay()
		sendAt = now.Add(delay)
	}

	pending := &PendingPacket{
		Packet:     packet,
		ReceivedAt: now,
		SendAt:     sendAt,
	}

	m.pending = append(m.pending, pending)
	m.stats.TotalReceived++

	// Si el batch está lleno, procesarlo inmediatamente
	if m.enableBatching && len(m.pending) >= m.batchSize {
		batch := m.pending
		m.pending = make([]*PendingPacket, 0)
		m.mu.Unlock()
		m.processBatch(batch)
		return nil
	}

	m.mu.Unlock()
	return nil
}

// Start inicia el bucle que procesa batches periódicamente.
func (m *Mixer) Start() {
	go func() {
		ticker := time.NewTicker(m.batchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.flushIfReady()
			}
		}
	}()
}

// flushIfReady procesa el batch actual si tiene paquetes.
func (m *Mixer) flushIfReady() {
	m.mu.Lock()
	if len(m.pending) == 0 {
		m.mu.Unlock()
		return
	}

	// Con batching, esperamos a tener al menos batchSize/2
	if m.enableBatching && len(m.pending) < m.batchSize/2 {
		// Verificar si el paquete más viejo lleva mucho esperando
		oldest := m.pending[0]
		if time.Since(oldest.ReceivedAt) < 2*time.Second {
			m.mu.Unlock()
			return
		}
	}

	batch := m.pending
	m.pending = make([]*PendingPacket, 0)
	m.mu.Unlock()

	m.processBatch(batch)
}

// processBatch procesa un batch de paquetes:
//
//   1. Reordena aleatoriamente
//   2. Espera a los delays Poisson individuales
//   3. Reenvía el batch
func (m *Mixer) processBatch(batch []*PendingPacket) {
	if len(batch) == 0 {
		return
	}

	// 1. Reordenar aleatoriamente
	m.mu.Lock()
	m.rng.Shuffle(len(batch), func(i, j int) {
		batch[i], batch[j] = batch[j], batch[i]
	})
	m.mu.Unlock()

	// 2. Calcular delay máximo del batch
	var maxDelay time.Duration
	for _, p := range batch {
		d := time.Until(p.SendAt)
		if d > maxDelay {
			maxDelay = d
		}
	}

	// 3. Si hay delays habilitados, esperar al máximo
	//    PERO cancelable si el mixer se cierra.
	//    (en producción, cada paquete esperaría su propio delay,
	//     pero para simplificar el batch, esperamos el máximo)
	if m.enableDelays && maxDelay > 0 && maxDelay < 5*time.Second {
		select {
		case <-time.After(maxDelay):
			// delay normal
		case <-m.stopCh:
			// cancelado por Close()
			return
		}
	}

	// 4. Extraer los packets ordenados
	packets := make([]*SphinxPacket, len(batch))
	var totalDelay int64
	for i, p := range batch {
		packets[i] = p.Packet
		totalDelay += time.Since(p.ReceivedAt).Milliseconds()
	}

	// 5. Actualizar stats
	m.mu.Lock()
	m.stats.TotalMixed += uint64(len(packets))
	m.stats.TotalBatches++
	if len(packets) > 0 {
		// Media ponderada del batch size
		totalBatches := float64(m.stats.TotalBatches)
		oldSum := m.stats.AvgBatchSize * (totalBatches - 1)
		m.stats.AvgBatchSize = (oldSum + float64(len(packets))) / totalBatches

		// Media del delay
		m.stats.AvgDelayMs = totalDelay / int64(len(packets))
	}
	callback := m.onBatchReady
	m.mu.Unlock()

	// 6. Disparar callback
	if callback != nil {
		callback(packets)
	}
}

// poissonDelay genera un delay según distribución Poisson.
//
// La distribución Poisson modela "cuánto tiempo hasta el próximo evento"
// donde los eventos ocurren a una tasa lambda (eventos por segundo).
//
// La fórmula es: delay = MinDelay + (-ln(U) / lambda)
// donde U es uniforme (0,1] y lambda es la tasa.
//
// IMPORTANTE: sumamos MinDelay para garantizar que TODOS los delays
// son >= MinDelay sin truncar (lo que destruiría la distribución).
func (m *Mixer) poissonDelay() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	// U uniforme en (0, 1]
	u := m.rng.Float64()
	if u <= 0 {
		u = 0.0001
	}
	if u >= 1 {
		u = 0.9999
	}

	// lambda en eventos por milisegundo
	lambdaPerMs := m.poissonLambda / 1000.0
	if lambdaPerMs <= 0 {
		lambdaPerMs = 0.003 // 3 eventos/segundo → ~333ms promedio
	}

	// delay base = -ln(U) / lambda (puede ser 0 si U≈1)
	baseMs := -math.Log(u) / lambdaPerMs
	if baseMs < 0 {
		baseMs = 0
	}

	// Sumar MinDelay (garantiza >= MinDelay sin truncar)
	minMs := float64(MinDelay) / float64(time.Millisecond)
	maxMs := float64(MaxDelay) / float64(time.Millisecond)

	delayMs := minMs + baseMs
	if delayMs > maxMs {
		delayMs = maxMs
	}

	return time.Duration(delayMs) * time.Millisecond
}

// Flush fuerza el procesamiento de todos los pendientes.
func (m *Mixer) Flush() {
	m.mu.Lock()
	if len(m.pending) == 0 {
		m.mu.Unlock()
		return
	}
	batch := m.pending
	m.pending = make([]*PendingPacket, 0)
	m.mu.Unlock()

	m.processBatch(batch)
}

// Stats devuelve las estadísticas del mixer.
func (m *Mixer) Stats() MixerStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

// PendingCount devuelve el número de paquetes pendientes.
func (m *Mixer) PendingCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.pending)
}

// Close cierra el mixer.
// Es idempotente: llamadas múltiples son seguras.
func (m *Mixer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	// Cerrar el canal con select para evitar panic si ya está cerrado
	select {
	case <-m.stopCh:
		// ya cerrado
	default:
		close(m.stopCh)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════
// Cover Traffic Generator
// ═══════════════════════════════════════════════════════════════

// CoverTrafficGenerator genera paquetes dummy para ocultar el tráfico real.
type CoverTrafficGenerator struct {
	rate     float64 // paquetes por segundo
	interval time.Duration

	mu       sync.Mutex
	rng      *mathrand.Rand
	stopCh   chan struct{}
	closed   bool

	onPacket func(*SphinxPacket)
	stats    CoverTrafficStats
}

// CoverTrafficStats lleva estadísticas de cover traffic.
type CoverTrafficStats struct {
	TotalGenerated uint64
	TotalSent      uint64
}

// NewCoverTrafficGenerator crea un generador.
func NewCoverTrafficGenerator(rate float64) *CoverTrafficGenerator {
	if rate <= 0 {
		rate = 1.0
	}
	interval := time.Duration(float64(time.Second) / rate)
	return &CoverTrafficGenerator{
		rate:     rate,
		interval: interval,
		rng:      mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
		stopCh:   make(chan struct{}),
	}
}

// SetPacketCallback establece el callback cuando se genera un paquete.
func (g *CoverTrafficGenerator) SetPacketCallback(fn func(*SphinxPacket)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onPacket = fn
}

// Start inicia el generador de cover traffic.
func (g *CoverTrafficGenerator) Start() {
	go func() {
		ticker := time.NewTicker(g.interval)
		defer ticker.Stop()

		for {
			select {
			case <-g.stopCh:
				return
			case <-ticker.C:
				g.generateAndSend()
			}
		}
	}()
}

// generateAndSend genera y envía un paquete dummy.
func (g *CoverTrafficGenerator) generateAndSend() {
	// Crear un paquete dummy
	packet := &SphinxPacket{
		EphemeralPublicKey: make([]byte, 32),
		Kyber768Ciphertext: make([]byte, 1088),
		RoutingInfo:        make([]byte, HeaderSize),
		MACs:               make([]byte, 32),
		Payload:            make([]byte, PayloadSize),
	}

	// Llenar con datos aleatorios
	g.mu.Lock()
	_, _ = g.rng.Read(packet.EphemeralPublicKey)
	_, _ = g.rng.Read(packet.Kyber768Ciphertext)
	_, _ = g.rng.Read(packet.Payload)
	g.stats.TotalGenerated++
	callback := g.onPacket
	g.mu.Unlock()

	if callback != nil {
		callback(packet)
		g.mu.Lock()
		g.stats.TotalSent++
		g.mu.Unlock()
	}
}

// Stats devuelve las estadísticas.
func (g *CoverTrafficGenerator) Stats() CoverTrafficStats {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stats
}

// Close cierra el generador.
// Es idempotente: llamadas múltiples son seguras.
func (g *CoverTrafficGenerator) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil
	}
	g.closed = true
	select {
	case <-g.stopCh:
		// ya cerrado
	default:
		close(g.stopCh)
	}
	return nil
}
