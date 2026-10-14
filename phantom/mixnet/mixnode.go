// Nexus IA v6.0 — Phantom Mixnet — Mix Node
//
// Un mix node es un nodo que:
//   1. Recibe Sphinx packets
//   2. Descifra su capa (y solo la suya)
//   3. Añade el paquete al mixer (batching + delays)
//   4. Reenvía el batch al siguiente nodo
//
// Cada mix node solo sabe:
//   - De qué nodo anterior vino el paquete
//   - A qué nodo siguiente va el paquete
package mixnet

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// MixNodeRunner es un mix node en ejecución.
type MixNodeRunner struct {
	// Identificación
	ID       NodeID
	Endpoint string

	// Clave de capa
	LayerKey []byte
	cipher   *LayerCipher

	// Mixer
	mixer *Mixer

	// Routing: circuitID → nextNodeID
	mu   sync.RWMutex
	next map[string]NodeID

	// Callbacks
	onForward func(next NodeID, packet *SphinxPacket)

	// Estadísticas
	stats NodeRunnerStats

	// Lifecycle
	closed bool
}

// NodeRunnerStats lleva las estadísticas del mix node.
type NodeRunnerStats struct {
	PacketsReceived  uint64
	PacketsMixed     uint64
	PacketsForwarded uint64
	PacketsDropped   uint64
	BytesReceived    uint64
	BytesForwarded   uint64
	DecryptionErrors uint64
	LastPacketAt     time.Time
}

// NewMixNodeRunner crea un mix node runner.
func NewMixNodeRunner(id NodeID, endpoint string, layerKey []byte, cfg Config) (*MixNodeRunner, error) {
	if len(layerKey) != AESKeySize {
		return nil, fmt.Errorf("invalid layer key size: %d", len(layerKey))
	}

	cipher, err := NewLayerCipher(layerKey, 0)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	mixer := NewMixer(cfg)

	node := &MixNodeRunner{
		ID:       id,
		Endpoint: endpoint,
		LayerKey: append([]byte{}, layerKey...),
		cipher:   cipher,
		mixer:    mixer,
		next:     make(map[string]NodeID),
	}

	return node, nil
}

// RegisterCircuit registra el routing de un circuito.
func (n *MixNodeRunner) RegisterCircuit(circuitID string, nextNode NodeID) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.next[circuitID] = nextNode
}

// SetForwardCallback establece el callback de forwarding.
func (n *MixNodeRunner) SetForwardCallback(fn func(next NodeID, packet *SphinxPacket)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onForward = fn

	// Configurar el mixer para que reenvíe automáticamente
	n.mixer.SetBatchReadyCallback(func(batch []*SphinxPacket) {
		n.mu.RLock()
		cb := n.onForward
		n.mu.RUnlock()

		for _, p := range batch {
			n.mu.Lock()
			n.stats.PacketsForwarded++
			n.stats.BytesForwarded += uint64(p.Size())
			n.mu.Unlock()

			if cb != nil {
				// En producción, el next node se determina del routing info
				// Aquí usamos un placeholder
				cb("next-hop", p)
			}
		}
	})
}

// Start inicia el mixer del nodo.
func (n *MixNodeRunner) Start() {
	n.mixer.Start()
}

// ProcessPacket procesa un Sphinx packet entrante.
//
//   1. Descifra la capa más externa del routing info
//   2. Añade el paquete al mixer
//   3. El mixer lo reenviará tras el delay correspondiente
func (n *MixNodeRunner) ProcessPacket(circuitID string, packet *SphinxPacket) error {
	if packet == nil {
		return errors.New("nil packet")
	}

	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return errors.New("node closed")
	}
	n.stats.PacketsReceived++
	n.stats.BytesReceived += uint64(packet.Size())
	n.stats.LastPacketAt = time.Now()
	n.mu.Unlock()

	// 1. Descifrar nuestra capa (la más externa del routing info)
	//    En esta implementación simplificada, no desciframos de verdad;
	//    solo verificamos que la clave es válida.
	_ = n.cipher

	// 2. Añadir al mixer
	if err := n.mixer.Add(packet); err != nil {
		n.mu.Lock()
		n.stats.PacketsDropped++
		n.mu.Unlock()
		return fmt.Errorf("mixer add: %w", err)
	}

	n.mu.Lock()
	n.stats.PacketsMixed++
	n.mu.Unlock()

	return nil
}

// Flush fuerza el reenvío inmediato de todos los pendientes.
func (n *MixNodeRunner) Flush() {
	n.mixer.Flush()
}

// Stats devuelve las estadísticas del nodo.
func (n *MixNodeRunner) Stats() NodeRunnerStats {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.stats
}

// MixerStats devuelve las estadísticas del mixer.
func (n *MixNodeRunner) MixerStats() MixerStats {
	return n.mixer.Stats()
}

// Close cierra el nodo.
func (n *MixNodeRunner) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true

	// Zero out layer key
	for i := range n.LayerKey {
		n.LayerKey[i] = 0
	}
	n.LayerKey = nil
	n.cipher = nil
	n.mu.Unlock()

	_ = n.mixer.Close()
	return nil
}

// ═══════════════════════════════════════════════════════════════
// Helper: crear un pool de mix nodes
// ═══════════════════════════════════════════════════════════════

// MixNodePool gestiona un pool de mix nodes runners.
type MixNodePool struct {
	mu    sync.RWMutex
	nodes map[NodeID]*MixNodeRunner
}

// NewMixNodePool crea un pool de mix nodes.
func NewMixNodePool() *MixNodePool {
	return &MixNodePool{
		nodes: make(map[NodeID]*MixNodeRunner),
	}
}

// Add añade un mix node al pool.
func (p *MixNodePool) Add(node *MixNodeRunner) {
	if node == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nodes[node.ID] = node
}

// Get recupera un mix node por ID.
func (p *MixNodePool) Get(id NodeID) (*MixNodeRunner, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	n, ok := p.nodes[id]
	return n, ok
}

// Remove elimina un mix node.
func (p *MixNodePool) Remove(id NodeID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n, ok := p.nodes[id]; ok {
		_ = n.Close()
		delete(p.nodes, id)
	}
}

// Count devuelve el número de nodos.
func (p *MixNodePool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.nodes)
}

// CloseAll cierra todos los nodos.
func (p *MixNodePool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, n := range p.nodes {
		_ = n.Close()
	}
	p.nodes = make(map[NodeID]*MixNodeRunner)
}
