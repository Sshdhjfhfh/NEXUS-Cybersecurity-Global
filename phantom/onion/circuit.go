// Nexus IA v6.0 — Phantom Onion — Construcción de circuitos
//
// Un circuito onion es un camino de N nodos. Cada nodo solo conoce:
//   - Al nodo anterior (de dónde vino la celda)
//   - Al nodo siguiente (a dónde va la celda)
//
// La construcción del circuito es incremental:
//   1. Cliente → Nodo1: CREATE (negocia clave con Nodo1)
//   2. Cliente → Nodo1 → Nodo2: EXTEND (a través de Nodo1)
//   3. Cliente → Nodo1 → Nodo2 → Nodo3: EXTEND
//   ...
//
// Cada nodo solo ve su capa del mensaje. Ninguno ve el camino completo.
package onion

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"

	"golang.org/x/crypto/hkdf"
)

// ═══════════════════════════════════════════════════════════════
// CircuitBuilder
// ═══════════════════════════════════════════════════════════════

// CircuitBuilder construye circuitos onion.
type CircuitBuilder struct {
	cfg       Config
	directory *Directory
	rng       *rand.Rand
}

// NewCircuitBuilder crea un constructor de circuitos.
func NewCircuitBuilder(cfg Config, directory *Directory) *CircuitBuilder {
	if cfg.Hops < MinHops {
		cfg.Hops = DefaultHops
	}
	if cfg.Hops > MaxHops {
		cfg.Hops = MaxHops
	}
	return &CircuitBuilder{
		cfg:       cfg,
		directory: directory,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Build construye un circuito nuevo.
func (cb *CircuitBuilder) Build() (*Circuit, *MultiLayerCipher, error) {
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
	masterKey, err := GenerateMasterKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate master key: %w", err)
	}

	// 3. Construir multi-layer cipher
	mlc, err := NewMultiLayerCipher(masterKey, cb.cfg.Hops)
	if err != nil {
		return nil, nil, fmt.Errorf("create multi layer: %w", err)
	}

	// 4. Construir capas
	layers := make([]*Layer, cb.cfg.Hops)
	path := make([]NodeID, cb.cfg.Hops)
	for i, node := range selected {
		path[i] = node.ID

		// Derivar claves de capa desde master
		info := []byte(fmt.Sprintf("onion-layer-%d", i))
		hkdfReader := hkdf.New(sha256.New, masterKey, nil, info)

		encKey := make([]byte, AESKeySize)
		macKey := make([]byte, AESKeySize)
		nonce := make([]byte, GCMNonceSize)

		if _, err := io.ReadFull(hkdfReader, encKey); err != nil {
			return nil, nil, fmt.Errorf("derive enc key: %w", err)
		}
		if _, err := io.ReadFull(hkdfReader, macKey); err != nil {
			return nil, nil, fmt.Errorf("derive mac key: %w", err)
		}
		if _, err := io.ReadFull(hkdfReader, nonce); err != nil {
			return nil, nil, fmt.Errorf("derive nonce: %w", err)
		}

		layers[i] = &Layer{
			Hop:           HopIndex(i),
			NodeID:        node.ID,
			EncryptionKey: encKey,
			MACKey:        macKey,
			Nonce:         nonce,
		}
	}

	// 5. Generar circuit ID
	circuitID, err := cb.generateCircuitID()
	if err != nil {
		return nil, nil, fmt.Errorf("generate circuit ID: %w", err)
	}

	// 6. Crear circuit
	now := time.Now()
	circuit := &Circuit{
		ID:         circuitID,
		Path:       path,
		Layers:     layers,
		CreatedAt:  now,
		ExpiresAt:  now.Add(cb.cfg.CircuitTTL),
		LastUsedAt: now,
	}

	return circuit, mlc, nil
}

// selectNodes selecciona N nodos distintos al azar.
func (cb *CircuitBuilder) selectNodes(nodes []*NodeInfo, count int) []*NodeInfo {
	// Shuffle
	shuffled := make([]*NodeInfo, len(nodes))
	copy(shuffled, nodes)
	cb.rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Tomar los primeros N
	return shuffled[:count]
}

// generateCircuitID genera un ID único de circuito.
func (cb *CircuitBuilder) generateCircuitID() (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(cb.rng, buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("onion-circuit-%x", buf), nil
}

// ═══════════════════════════════════════════════════════════════
// CircuitPool
// ═══════════════════════════════════════════════════════════════

// CircuitPool mantiene circuitos activos.
type CircuitPool struct {
	builder  *CircuitBuilder
	circuits map[string]*Circuit
	ciphers  map[string]*MultiLayerCipher
	maxSize  int
}

// NewCircuitPool crea un pool de circuitos.
func NewCircuitPool(builder *CircuitBuilder, maxSize int) *CircuitPool {
	if maxSize <= 0 {
		maxSize = 10
	}
	return &CircuitPool{
		builder:  builder,
		circuits: make(map[string]*Circuit),
		ciphers:  make(map[string]*MultiLayerCipher),
		maxSize:  maxSize,
	}
}

// GetOrCreate devuelve un circuito existente o crea uno nuevo.
func (cp *CircuitPool) GetOrCreate() (*Circuit, *MultiLayerCipher, error) {
	// Buscar circuito válido
	for id, c := range cp.circuits {
		if !c.IsExpired() && !c.Closed && c.TTL() > 30*time.Second {
			return c, cp.ciphers[id], nil
		}
	}

	// Crear uno nuevo
	if len(cp.circuits) >= cp.maxSize {
		// Eliminar el más viejo
		cp.evictOldest()
	}

	circuit, cipher, err := cp.builder.Build()
	if err != nil {
		return nil, nil, err
	}

	cp.circuits[circuit.ID] = circuit
	cp.ciphers[circuit.ID] = cipher
	return circuit, cipher, nil
}

// Get recupera un circuito por ID.
func (cp *CircuitPool) Get(id string) (*Circuit, *MultiLayerCipher, bool) {
	c, ok := cp.circuits[id]
	if !ok {
		return nil, nil, false
	}
	return c, cp.ciphers[id], true
}

// Close cierra un circuito.
func (cp *CircuitPool) Close(id string) {
	if c, ok := cp.circuits[id]; ok {
		c.Closed = true
		delete(cp.circuits, id)
		delete(cp.ciphers, id)
	}
}

// CloseAll cierra todos los circuitos.
func (cp *CircuitPool) CloseAll() {
	for id, c := range cp.circuits {
		c.Closed = true
		delete(cp.circuits, id)
		delete(cp.ciphers, id)
	}
}

// Size devuelve el número de circuitos en el pool.
func (cp *CircuitPool) Size() int {
	return len(cp.circuits)
}

// List devuelve todos los circuitos.
func (cp *CircuitPool) List() []*Circuit {
	out := make([]*Circuit, 0, len(cp.circuits))
	for _, c := range cp.circuits {
		out = append(out, c)
	}
	return out
}

// evictOldest elimina el circuito más viejo.
func (cp *CircuitPool) evictOldest() {
	var oldest *Circuit
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
// Devuelve el número eliminado.
func (cp *CircuitPool) PruneExpired() int {
	count := 0
	for id, c := range cp.circuits {
		if c.IsExpired() || c.Closed {
			delete(cp.circuits, id)
			delete(cp.ciphers, id)
			count++
		}
	}
	return count
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// NewCircuitFromPath crea un circuito con un path específico (para tests).
func NewCircuitFromPath(path []NodeID, masterKey []byte, ttl time.Duration) (*Circuit, *MultiLayerCipher, error) {
	hops := len(path)
	if hops < MinHops {
		return nil, nil, fmt.Errorf("%s: need at least %d", ErrInvalidHops, MinHops)
	}

	mlc, err := NewMultiLayerCipher(masterKey, hops)
	if err != nil {
		return nil, nil, err
	}

	layers := make([]*Layer, hops)
	for i := 0; i < hops; i++ {
		info := []byte(fmt.Sprintf("onion-layer-%d", i))
		hkdfReader := hkdf.New(sha256.New, masterKey, nil, info)

		encKey := make([]byte, AESKeySize)
		macKey := make([]byte, AESKeySize)
		nonce := make([]byte, GCMNonceSize)

		if _, err := io.ReadFull(hkdfReader, encKey); err != nil {
			return nil, nil, err
		}
		if _, err := io.ReadFull(hkdfReader, macKey); err != nil {
			return nil, nil, err
		}
		if _, err := io.ReadFull(hkdfReader, nonce); err != nil {
			return nil, nil, err
		}

		layers[i] = &Layer{
			Hop:           HopIndex(i),
			NodeID:        path[i],
			EncryptionKey: encKey,
			MACKey:        macKey,
			Nonce:         nonce,
		}
	}

	now := time.Now()
	circuit := &Circuit{
		ID:         fmt.Sprintf("test-circuit-%d", time.Now().UnixNano()),
		Path:       path,
		Layers:     layers,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		LastUsedAt: now,
	}

	return circuit, mlc, nil
}

// uint32ToBytes convierte un uint32 a 4 bytes.
func uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
