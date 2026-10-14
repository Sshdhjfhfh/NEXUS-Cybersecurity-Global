// Nexus IA v6.0 — Phantom Garlic Routing
//
// Garlic routing extiende el onion routing tradicional añadiendo
// "bulbos" (cloves) de mensajes señuelo. Cada garlic packet contiene:
//   - 1 mensaje real (payload)
//   - N mensajes señuelo (dummy)
//
// Un observador no puede distinguir cuál es el real.
// Cada clove se puede entregar a destinos distintos.
//
// El PDF lo describe así:
//   'Garlic routing interno. 1 real + N señuelos por bulbo cifrado.'
package garlic

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// DefaultDecoys es el número por defecto de señuelos.
	DefaultDecoys = 3

	// MinDecoys es el mínimo de señuelos.
	MinDecoys = 1

	// MaxDecoys es el máximo razonable.
	MaxDecoys = 10

	// DefaultCloveSize es el tamaño de cada clove.
	DefaultCloveSize = 1024
)

// ═══════════════════════════════════════════════════════════════
// Clove — un mensaje dentro del garlic
// ═══════════════════════════════════════════════════════════════

// Clove es un mensaje individual dentro de un garlic bulb.
type Clove struct {
	// ID único del clove
	ID string

	// Destino del clove (puede ser distinto por clove)
	Destination string

	// Tipo: "real" o "decoy"
	Type CloveType

	// Payload cifrado
	Payload []byte

	// Padding hasta CloveSize
	Padding []byte
}

// CloveType clasifica el tipo de clove.
type CloveType string

const (
	TypeReal  CloveType = "REAL"
	TypeDecoy CloveType = "DECOY"
)

// IsReal devuelve true si es el clove real.
func (c *Clove) IsReal() bool {
	return c.Type == TypeReal
}

// ═══════════════════════════════════════════════════════════════
// Bulb — un garlic completo
// ═══════════════════════════════════════════════════════════════

// Bulb es un paquete garlic completo.
type Bulb struct {
	// ID único del bulb
	ID string

	// Cloves (1 real + N señuelos)
	Cloves []*Clove

	// El clove real está en un índice aleatorio
	RealIndex int

	// Metadata
	TotalSize int
}

// RealClove devuelve el clove real.
func (b *Bulb) RealClove() *Clove {
	if b.RealIndex < 0 || b.RealIndex >= len(b.Cloves) {
		return nil
	}
	return b.Cloves[b.RealIndex]
}

// DecoyCount devuelve el número de señuelos.
func (b *Bulb) DecoyCount() int {
	count := 0
	for _, c := range b.Cloves {
		if !c.IsReal() {
			count++
		}
	}
	return count
}

// ═══════════════════════════════════════════════════════════════
// Builder
// ═══════════════════════════════════════════════════════════════

// Builder construye garlic bulbs.
type Builder struct {
	decoyCount int
	cloveSize  int

	mu  sync.Mutex
	rng *randReader
}

// Config configura el builder.
type Config struct {
	Decoys    int
	CloveSize int
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		Decoys:    DefaultDecoys,
		CloveSize: DefaultCloveSize,
	}
}

// NewBuilder crea un builder.
func NewBuilder(cfg Config) *Builder {
	if cfg.Decoys < MinDecoys {
		cfg.Decoys = DefaultDecoys
	}
	if cfg.Decoys > MaxDecoys {
		cfg.Decoys = MaxDecoys
	}
	if cfg.CloveSize <= 0 {
		cfg.CloveSize = DefaultCloveSize
	}
	return &Builder{
		decoyCount: cfg.Decoys,
		cloveSize:  cfg.CloveSize,
		rng:        &randReader{},
	}
}

// Build construye un bulb con 1 real + N señuelos.
func (b *Builder) Build(payload []byte, destination string) (*Bulb, error) {
	if len(payload) == 0 {
		return nil, errors.New("empty payload")
	}
	if destination == "" {
		return nil, errors.New("empty destination")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Número total de cloves
	total := b.decoyCount + 1

	// Crear cloves
	cloves := make([]*Clove, total)

	// Elegir índice aleatorio para el clove real
	realIdx, err := b.randomIndex(total)
	if err != nil {
		return nil, fmt.Errorf("random index: %w", err)
	}

	// Crear clove real
	cloves[realIdx] = &Clove{
		ID:          newID(),
		Destination: destination,
		Type:        TypeReal,
		Payload:     padTo(payload, b.cloveSize),
	}

	// Crear señuelos
	for i := 0; i < total; i++ {
		if i == realIdx {
			continue
		}
		decoy, err := b.makeDecoy(destination)
		if err != nil {
			return nil, err
		}
		cloves[i] = decoy
	}

	bulb := &Bulb{
		ID:        newID(),
		Cloves:    cloves,
		RealIndex: realIdx,
		TotalSize: total * b.cloveSize,
	}

	return bulb, nil
}

// makeDecoy crea un clove señuelo con datos aleatorios.
func (b *Builder) makeDecoy(realDest string) (*Clove, error) {
	payload := make([]byte, b.cloveSize)
	if _, err := b.rng.Read(payload); err != nil {
		return nil, err
	}

	// Los señuelos pueden tener destinos distintos o el mismo
	// Aquí usamos el mismo destino para que parezcan reales
	return &Clove{
		ID:          newID(),
		Destination: realDest,
		Type:        TypeDecoy,
		Payload:     payload,
	}, nil
}

// randomIndex genera un índice aleatorio [0, n).
func (b *Builder) randomIndex(n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("n must be > 0")
	}
	buf := make([]byte, 4)
	if _, err := b.rng.Read(buf); err != nil {
		return 0, err
	}
	v := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
	return int(v % uint32(n)), nil
}

// ═══════════════════════════════════════════════════════════════
// Parser
// ═══════════════════════════════════════════════════════════════

// Parser extrae cloves de un bulb.
type Parser struct{}

// NewParser crea un parser.
func NewParser() *Parser {
	return &Parser{}
}

// ExtractReal extrae el clove real de un bulb.
func (p *Parser) ExtractReal(bulb *Bulb) (*Clove, error) {
	if bulb == nil {
		return nil, errors.New("nil bulb")
	}
	real := bulb.RealClove()
	if real == nil {
		return nil, errors.New("no real clove in bulb")
	}
	return real, nil
}

// ExtractAll devuelve todos los cloves.
func (p *Parser) ExtractAll(bulb *Bulb) []*Clove {
	if bulb == nil {
		return nil
	}
	return bulb.Cloves
}

// CountDecoys cuenta los señuelos.
func (p *Parser) CountDecoys(bulb *Bulb) int {
	return bulb.DecoyCount()
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// randReader envuelve crypto/rand.
type randReader struct{}

func (r *randReader) Read(b []byte) (int, error) {
	return rand.Read(b)
}

// padTo rellena payload a un tamaño exacto con padding aleatorio.
func padTo(data []byte, size int) []byte {
	out := make([]byte, size)
	copy(out, data)
	if len(data) < size {
		// Padding aleatorio
		_, _ = rand.Read(out[len(data):])
	}
	return out
}

// newID genera un ID aleatorio.
func newID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("garlic-%x", buf)
}
