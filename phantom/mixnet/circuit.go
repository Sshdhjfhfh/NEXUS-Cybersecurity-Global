// Nexus IA v6.0 — Phantom Mixnet — Circuitos
//
// Un circuito mixnet es un camino de N nodos de mezcla.
// Cada nodo solo descifra su capa y reenvía, con un retraso aleatorio.
//
// A diferencia del onion, el mixnet añade:
//   - Batching (agrupa paquetes)
//   - Delays Poisson (aleatorios)
//   - Reordenación (shuffle)
package mixnet

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand"
	"time"

	"golang.org/x/crypto/hkdf"
)

// ═══════════════════════════════════════════════════════════════
// MixCircuitBuilder
// ═══════════════════════════════════════════════════════════════

// MixCircuitBuilder construye circuitos mixnet.
type MixCircuitBuilder struct {
	cfg       Config
	directory *MixDirectory
	rng       *mathrand.Rand
}

// NewMixCircuitBuilder crea un constructor.
func NewMixCircuitBuilder(cfg Config, directory *MixDirectory) *MixCircuitBuilder {
	if cfg.Hops < MinHops {
		cfg.Hops = DefaultHops
	}
	if cfg.Hops > MaxHops {
		cfg.Hops = MaxHops
	}
	return &MixCircuitBuilder{
		cfg:       cfg,
		directory: directory,
		rng:       mathrand.New(mathrand.NewSource(time.Now().UnixNano())),
	}
}

// Build construye un circuito nuevo.
func (cb *MixCircuitBuilder) Build() (*MixCircuit, [][]byte, error) {
	if cb.directory == nil {
		return nil, nil, errors.New("no directory")
	}

	activeNodes := cb.directory.ListActive()
	if len(activeNodes) == 0 {
		return nil, nil, errors.New(ErrNoNodes)
	}
	if len(activeNodes) < cb.cfg.Hops {
		return nil, nil, fmt.Errorf("%s: need %d, have %d",
			ErrNotEnoughNodes, cb.cfg.Hops, len(activeNodes))
	}

	// 1. Seleccionar nodos sin repetición
	selected := cb.selectNodes(activeNodes, cb.cfg.Hops)

	// 2. Generar master key
	masterKey, err := generateMasterKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate master key: %w", err)
	}

	// 3. Generar claves de capa (una por hop)
	layerKeys := make([][]byte, cb.cfg.Hops)
	path := make([]NodeID, cb.cfg.Hops)
	for i, node := range selected {
		path[i] = node.ID
		info := []byte(fmt.Sprintf("mixnet-layer-%d", i))
		hkdfReader := hkdf.New(sha256.New, masterKey, nil, info)
		layerKey := make([]byte, 32) // AES-256
		if _, err := io.ReadFull(hkdfReader, layerKey); err != nil {
			return nil, nil, fmt.Errorf("derive layer key %d: %w", i, err)
		}
		layerKeys[i] = layerKey
	}

	// 4. Generar circuit ID
	circuitID, err := cb.generateCircuitID()
	if err != nil {
		return nil, nil, err
	}

	// 5. Crear circuito
	now := time.Now()
	circuit := &MixCircuit{
		ID:         circuitID,
		Path:       path,
		LayerKeys:  layerKeys,
		CreatedAt:  now,
		ExpiresAt:  now.Add(cb.cfg.CircuitTTL),
		LastUsedAt: now,
	}

	return circuit, layerKeys, nil
}

// selectNodes selecciona N nodos distintos.
func (cb *MixCircuitBuilder) selectNodes(nodes []*MixNode, count int) []*MixNode {
	shuffled := make([]*MixNode, len(nodes))
	copy(shuffled, nodes)
	cb.rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:count]
}

// generateCircuitID genera un ID único.
func (cb *MixCircuitBuilder) generateCircuitID() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(cb.rng, buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("mix-circuit-%x", buf), nil
}

// ═══════════════════════════════════════════════════════════════
// MixCircuitPool
// ═══════════════════════════════════════════════════════════════

// MixCircuitPool mantiene circuitos activos.
type MixCircuitPool struct {
	builder  *MixCircuitBuilder
	circuits map[string]*MixCircuit
	maxSize  int
}

// NewMixCircuitPool crea un pool de circuitos.
func NewMixCircuitPool(builder *MixCircuitBuilder, maxSize int) *MixCircuitPool {
	if maxSize <= 0 {
		maxSize = 10
	}
	return &MixCircuitPool{
		builder:  builder,
		circuits: make(map[string]*MixCircuit),
		maxSize:  maxSize,
	}
}

// GetOrCreate devuelve un circuito existente o crea uno nuevo.
func (cp *MixCircuitPool) GetOrCreate() (*MixCircuit, error) {
	// Buscar circuito válido
	for _, c := range cp.circuits {
		if !c.IsExpired() && !c.Closed && c.TTL() > 30*time.Second {
			return c, nil
		}
	}

	// Crear uno nuevo
	if len(cp.circuits) >= cp.maxSize {
		cp.evictOldest()
	}

	circuit, _, err := cp.builder.Build()
	if err != nil {
		return nil, err
	}

	cp.circuits[circuit.ID] = circuit
	return circuit, nil
}

// Get recupera un circuito por ID.
func (cp *MixCircuitPool) Get(id string) (*MixCircuit, bool) {
	c, ok := cp.circuits[id]
	return c, ok
}

// Close cierra un circuito.
func (cp *MixCircuitPool) Close(id string) {
	if c, ok := cp.circuits[id]; ok {
		c.Closed = true
		// Zero out layer keys
		for _, k := range c.LayerKeys {
			for i := range k {
				k[i] = 0
			}
		}
		delete(cp.circuits, id)
	}
}

// CloseAll cierra todos los circuitos.
func (cp *MixCircuitPool) CloseAll() {
	for id := range cp.circuits {
		cp.Close(id)
	}
}

// Size devuelve el número de circuitos.
func (cp *MixCircuitPool) Size() int {
	return len(cp.circuits)
}

// List devuelve todos los circuitos.
func (cp *MixCircuitPool) List() []*MixCircuit {
	out := make([]*MixCircuit, 0, len(cp.circuits))
	for _, c := range cp.circuits {
		out = append(out, c)
	}
	return out
}

// evictOldest elimina el circuito más viejo.
func (cp *MixCircuitPool) evictOldest() {
	var oldest *MixCircuit
	for _, c := range cp.circuits {
		if oldest == nil || c.CreatedAt.Before(oldest.CreatedAt) {
			oldest = c
		}
	}
	if oldest != nil {
		cp.Close(oldest.ID)
	}
}

// PruneExpired elimina circuitos expirados.
func (cp *MixCircuitPool) PruneExpired() int {
	count := 0
	for id, c := range cp.circuits {
		if c.IsExpired() || c.Closed {
			cp.Close(id)
			count++
		}
	}
	return count
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// generateMasterKey genera una master key de 32 bytes.
func generateMasterKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(cryptorand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// uint32ToBytes convierte uint32 a 4 bytes.
func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
