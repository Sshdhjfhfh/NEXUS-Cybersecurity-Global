// Nexus IA v6.0 — Phantom Timing
//
// Ofuscación temporal: destruye la correlación de timing entre
// entrada y salida de paquetes. Técnicas:
//
//   1. Jitter exponencial: delays aleatorios por paquete
//   2. Batching: agrupa paquetes antes de enviar
//   3. Reordenación: shuffle aleatorio antes de enviar
//
// El PDF lo describe así:
//   'Jitter exponencial + batching + reordenación aleatoria.
//    Destruye correlación temporal para adversario con vista global.'
package timing

import (
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
	// DefaultLambda es el parámetro de la distribución exponencial.
	// En ms. λ = 0.1 → 100ms promedio.
	DefaultLambda float64 = 0.1

	// DefaultMinDelay es el delay mínimo.
	DefaultMinDelay = 20 * time.Millisecond

	// DefaultMaxDelay es el delay máximo.
	DefaultMaxDelay = 2 * time.Second

	// DefaultBatchSize es el tamaño del batch.
	DefaultBatchSize = 20

	// DefaultBatchInterval es el tiempo máximo en el batch.
	DefaultBatchInterval = 200 * time.Millisecond
)

// ═══════════════════════════════════════════════════════════════
// Jitter
// ═══════════════════════════════════════════════════════════════

// Jitter aplica delays exponenciales a paquetes.
type Jitter struct {
	lambda   float64 // parámetro λ (mayor = delays más cortos)
	minDelay time.Duration
	maxDelay time.Duration

	mu  sync.Mutex
	rng *mathrand.Rand
}

// NewJitter crea un jitter con parámetros por defecto.
func NewJitter() *Jitter {
	return &Jitter{
		lambda:   DefaultLambda,
		minDelay: DefaultMinDelay,
		maxDelay: DefaultMaxDelay,
		rng:      mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

// NewJitterWithConfig crea un jitter con parámetros custom.
func NewJitterWithConfig(lambda float64, minDelay, maxDelay time.Duration) *Jitter {
	if lambda <= 0 {
		lambda = DefaultLambda
	}
	if minDelay <= 0 {
		minDelay = DefaultMinDelay
	}
	if maxDelay <= minDelay {
		maxDelay = DefaultMaxDelay
	}
	return &Jitter{
		lambda:   lambda,
		minDelay: minDelay,
		maxDelay: maxDelay,
		rng:      mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

// Delay genera un delay exponencial.
//
// Fórmula: delay = MinDelay + (-ln(U) / λ)
// donde U uniforme (0, 1].
func (j *Jitter) Delay() time.Duration {
	j.mu.Lock()
	defer j.mu.Unlock()

	u := j.rng.Float64()
	if u <= 0 {
		u = 0.0001
	}
	if u >= 1 {
		u = 0.9999
	}

	// delay base en ms
	baseMs := -math.Log(u) / j.lambda

	// Añadir MinDelay
	minMs := float64(j.minDelay) / float64(time.Millisecond)
	maxMs := float64(j.maxDelay) / float64(time.Millisecond)

	delayMs := minMs + baseMs
	if delayMs > maxMs {
		delayMs = maxMs
	}

	return time.Duration(delayMs) * time.Millisecond
}

// Apply aplica el delay (bloquea).
func (j *Jitter) Apply() {
	time.Sleep(j.Delay())
}

// SetLambda cambia el parámetro λ.
func (j *Jitter) SetLambda(lambda float64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if lambda > 0 {
		j.lambda = lambda
	}
}

// Lambda devuelve el parámetro actual.
func (j *Jitter) Lambda() float64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.lambda
}

// ═══════════════════════════════════════════════════════════════
// Batcher
// ═══════════════════════════════════════════════════════════════

// Batcher agrupa elementos antes de procesarlos.
type Batcher[T any] struct {
	batchSize     int
	batchInterval time.Duration

	mu       sync.Mutex
	pending  []T
	lastFlush time.Time
	flushCh  chan []T
	rng      *mathrand.Rand
}

// NewBatcher crea un batcher.
func NewBatcher[T any](batchSize int, batchInterval time.Duration) *Batcher[T] {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	if batchInterval <= 0 {
		batchInterval = DefaultBatchInterval
	}
	return &Batcher[T]{
		batchSize:     batchSize,
		batchInterval: batchInterval,
		pending:       make([]T, 0, batchSize),
		lastFlush:     time.Now(),
		flushCh:       make(chan []T, 16),
		rng:           mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

// Add añade un elemento al batch.
// Si el batch está lleno, se dispara el flush.
func (b *Batcher[T]) Add(item T) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pending = append(b.pending, item)

	// Si está lleno, flush
	if len(b.pending) >= b.batchSize {
		b.flushLocked()
	}
	return nil
}

// Flush fuerza el flush del batch actual.
func (b *Batcher[T]) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked()
}

// flushLocked hace flush (debe tener el lock).
func (b *Batcher[T]) flushLocked() {
	if len(b.pending) == 0 {
		return
	}
	// Reordenar aleatoriamente antes de enviar
	b.rng.Shuffle(len(b.pending), func(i, j int) {
		b.pending[i], b.pending[j] = b.pending[j], b.pending[i]
	})

	// Copia defensiva
	out := make([]T, len(b.pending))
	copy(out, b.pending)

	// Reset
	b.pending = b.pending[:0]
	b.lastFlush = time.Now()

	// Enviar al canal (no bloqueante)
	select {
	case b.flushCh <- out:
	default:
		// Canal lleno, descartar (o podríamos bloquear)
	}
}

// Batches devuelve el canal de batches listos.
func (b *Batcher[T]) Batches() <-chan []T {
	return b.flushCh
}

// PendingCount devuelve cuántos elementos hay pendientes.
func (b *Batcher[T]) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// CheckInterval fuerza un flush si ha pasado el intervalo.
func (b *Batcher[T]) CheckInterval() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Since(b.lastFlush) >= b.batchInterval && len(b.pending) > 0 {
		b.flushLocked()
	}
}

// ═══════════════════════════════════════════════════════════════
// Shuffler
// ═══════════════════════════════════════════════════════════════

// Shuffler reordena aleatoriamente slices.
type Shuffler struct {
	mu  sync.Mutex
	rng *mathrand.Rand
}

// NewShuffler crea un shuffler.
func NewShuffler() *Shuffler {
	return &Shuffler{
		rng: mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

// Shuffle reordena un slice de bytes.
func (s *Shuffler) Shuffle(b []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rng.Shuffle(len(b), func(i, j int) {
		b[i], b[j] = b[j], b[i]
	})
}

// ShuffleInts reordena un slice de ints.
func (s *Shuffler) ShuffleInts(b []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rng.Shuffle(len(b), func(i, j int) {
		b[i], b[j] = b[j], b[i]
	})
}

// ═══════════════════════════════════════════════════════════════
// TimingObfuscator
// ═══════════════════════════════════════════════════════════════

// TimingObfuscator combina jitter + batching + shuffling.
type TimingObfuscator struct {
	jitter  *Jitter
	batcher *Batcher[[]byte]
	shuf    *Shuffler

	stats TimingStats
	mu    sync.Mutex
}

// TimingStats lleva estadísticas.
type TimingStats struct {
	TotalProcessed uint64
	TotalDelays    time.Duration
	AvgDelayMs     int64
}

// NewTimingObfuscator crea el obfuscator completo.
func NewTimingObfuscator() *TimingObfuscator {
	return &TimingObfuscator{
		jitter:  NewJitter(),
		batcher: NewBatcher[[]byte](DefaultBatchSize, DefaultBatchInterval),
		shuf:    NewShuffler(),
	}
}

// Process procesa un payload con todas las técnicas.
//
//   1. Aplica jitter (delay aleatorio)
//   2. Añade al batcher
//   3. Cuando el batch esté listo, se reordena y se envía
func (t *TimingObfuscator) Process(payload []byte) error {
	if len(payload) == 0 {
		return errors.New("empty payload")
	}

	// 1. Aplicar jitter
	delay := t.jitter.Delay()
	t.mu.Lock()
	t.stats.TotalProcessed++
	t.stats.TotalDelays += delay
	if t.stats.TotalProcessed > 0 {
		t.stats.AvgDelayMs = int64(t.stats.TotalDelays/time.Millisecond) / int64(t.stats.TotalProcessed)
	}
	t.mu.Unlock()

	time.Sleep(delay)

	// 2. Añadir al batcher
	return t.batcher.Add(payload)
}

// Batches devuelve el canal de batches listos.
func (t *TimingObfuscator) Batches() <-chan [][]byte {
	return t.batcher.Batches()
}

// Flush fuerza el flush del batcher.
func (t *TimingObfuscator) Flush() {
	t.batcher.Flush()
}

// Stats devuelve las estadísticas.
func (t *TimingObfuscator) Stats() TimingStats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stats
}

// ═══════════════════════════════════════════════════════════════
// Manager
// ═══════════════════════════════════════════════════════════════

// Manager gestiona los componentes de timing.
type Manager struct {
	Jitter    *Jitter
	Batcher   *Batcher[[]byte]
	Shuffler  *Shuffler
	Full      *TimingObfuscator
}

// NewManager crea un manager completo.
func NewManager() *Manager {
	return &Manager{
		Jitter:   NewJitter(),
		Batcher:  NewBatcher[[]byte](DefaultBatchSize, DefaultBatchInterval),
		Shuffler: NewShuffler(),
		Full:     NewTimingObfuscator(),
	}
}

// ═══════════════════════════════════════════════════════════════
// Config
// ═══════════════════════════════════════════════════════════════

// Config configura el timing obfuscator.
type Config struct {
	Lambda        float64
	MinDelay      time.Duration
	MaxDelay      time.Duration
	BatchSize     int
	BatchInterval time.Duration
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		Lambda:        DefaultLambda,
		MinDelay:      DefaultMinDelay,
		MaxDelay:      DefaultMaxDelay,
		BatchSize:     DefaultBatchSize,
		BatchInterval: DefaultBatchInterval,
	}
}

// NewWithConfig crea un manager con config custom.
func NewWithConfig(cfg Config) *Manager {
	m := NewManager()
	m.Jitter = NewJitterWithConfig(cfg.Lambda, cfg.MinDelay, cfg.MaxDelay)
	m.Batcher = NewBatcher[[]byte](cfg.BatchSize, cfg.BatchInterval)
	return m
}
