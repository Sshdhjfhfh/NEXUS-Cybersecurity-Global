// Nexus IA v6.0 — Phantom VPN — Pool de nodos
//
// El pool de nodos son contenedores Docker propios que actúan
// como relays VPN. Cada nodo tiene:
//   - Una ubicación lógica (zone)
//   - Latencia medida
//   - Estado (activo, caído, mantenimiento)
//   - Clave pública X25519 + Kyber768
//
// La selección de nodo usa:
//   - Latencia (más bajo = mejor)
//   - Balanceo por zona geográfica
//   - Rotación periódica (evita correlación)
package vpn

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Tipos base
// ═══════════════════════════════════════════════════════════════

// NodeState es el estado de un nodo.
type NodeState string

const (
	NodeActive       NodeState = "ACTIVE"
	NodeDegraded     NodeState = "DEGRADED"
	NodeDown         NodeState = "DOWN"
	NodeMaintenance  NodeState = "MAINTENANCE"
)

// NodeZone es la zona geográfica lógica del nodo.
// NO es ubicación real (por privacidad).
type NodeZone string

const (
	ZoneA NodeZone = "zone-a"
	ZoneB NodeZone = "zone-b"
	ZoneC NodeZone = "zone-c"
	ZoneD NodeZone = "zone-d"
)

// Node es un nodo del pool VPN.
type Node struct {
	// Identificación
	ID       string
	Name     string
	Zone     NodeZone
	Endpoint string // host:port del contenedor Docker

	// Claves públicas (para handshake con el nodo)
	X25519PublicKey [X25519KeySize]byte
	Kyber768PublicKey []byte

	// Estado
	State           NodeState
	LastHealthCheck time.Time
	LastLatencyMs   int64

	// Estadísticas
	TotalConnections uint64
	FailedConnections uint64

	// Metadatos
	Metadata map[string]string
}

// IsUsable devuelve true si el nodo se puede usar para conexión.
func (n *Node) IsUsable() bool {
	return n.State == NodeActive || n.State == NodeDegraded
}

// ═══════════════════════════════════════════════════════════════
// Pool
// ═══════════════════════════════════════════════════════════════

// Pool gestiona el conjunto de nodos VPN.
type Pool struct {
	mu    sync.RWMutex
	nodes map[string]*Node

	// Configuración
	MaxLatencyMs   int64
	AllowDegraded  bool
	PreferLowLatency bool
}

// NewPool crea un pool vacío.
func NewPool() *Pool {
	return &Pool{
		nodes:            make(map[string]*Node),
		MaxLatencyMs:     500,
		AllowDegraded:    true,
		PreferLowLatency: true,
	}
}

// Add añade un nodo al pool.
func (p *Pool) Add(node *Node) error {
	if node == nil {
		return errors.New("nil node")
	}
	if node.ID == "" {
		return errors.New("empty node ID")
	}
	if node.Endpoint == "" {
		return errors.New("empty node endpoint")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if node.Metadata == nil {
		node.Metadata = make(map[string]string)
	}
	if node.LastHealthCheck.IsZero() {
		node.LastHealthCheck = time.Now()
	}
	p.nodes[node.ID] = node
	return nil
}

// Remove elimina un nodo del pool.
func (p *Pool) Remove(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.nodes[id]; !ok {
		return false
	}
	delete(p.nodes, id)
	return true
}

// Get recupera un nodo por ID.
func (p *Pool) Get(id string) (*Node, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	n, ok := p.nodes[id]
	if !ok {
		return nil, false
	}
	cpy := *n
	return &cpy, true
}

// List devuelve todos los nodos.
func (p *Pool) List() []*Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]*Node, 0, len(p.nodes))
	for _, n := range p.nodes {
		cpy := *n
		out = append(out, &cpy)
	}
	return out
}

// ListUsable devuelve los nodos usables (activos o degradados).
func (p *Pool) ListUsable() []*Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]*Node, 0)
	for _, n := range p.nodes {
		if !n.IsUsable() {
			continue
		}
		if n.State == NodeDegraded && !p.AllowDegraded {
			continue
		}
		if p.MaxLatencyMs > 0 && n.LastLatencyMs > p.MaxLatencyMs {
			continue
		}
		cpy := *n
		out = append(out, &cpy)
	}
	return out
}

// SelectBest devuelve el mejor nodo según los criterios del pool.
// Si no hay nodos usables, devuelve nil.
func (p *Pool) SelectBest() *Node {
	usable := p.ListUsable()
	if len(usable) == 0 {
		return nil
	}

	if !p.PreferLowLatency {
		// Sin preferencia, devolver el primero
		return usable[0]
	}

	// Ordenar por latencia (más bajo = mejor)
	best := usable[0]
	for _, n := range usable[1:] {
		if n.LastLatencyMs < best.LastLatencyMs {
			best = n
		}
	}
	return best
}

// SelectByZone devuelve el mejor nodo de una zona específica.
func (p *Pool) SelectByZone(zone NodeZone) *Node {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var best *Node
	for _, n := range p.nodes {
		if n.Zone != zone {
			continue
		}
		if !n.IsUsable() {
			continue
		}
		if best == nil || n.LastLatencyMs < best.LastLatencyMs {
			cpy := *n
			best = &cpy
		}
	}
	return best
}

// UpdateState actualiza el estado de un nodo.
func (p *Pool) UpdateState(id string, state NodeState) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	n, ok := p.nodes[id]
	if !ok {
		return fmt.Errorf("node not found: %s", id)
	}
	n.State = state
	n.LastHealthCheck = time.Now()
	return nil
}

// UpdateLatency actualiza la latencia de un nodo.
func (p *Pool) UpdateLatency(id string, latencyMs int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	n, ok := p.nodes[id]
	if !ok {
		return fmt.Errorf("node not found: %s", id)
	}
	n.LastLatencyMs = latencyMs
	n.LastHealthCheck = time.Now()
	return nil
}

// RecordConnection registra una conexión exitosa.
func (p *Pool) RecordConnection(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n, ok := p.nodes[id]; ok {
		n.TotalConnections++
	}
}

// RecordFailure registra una conexión fallida.
func (p *Pool) RecordFailure(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n, ok := p.nodes[id]; ok {
		n.FailedConnections++
		// Degradar si muchos fallos
		if n.FailedConnections > 10 && n.State == NodeActive {
			n.State = NodeDegraded
		}
	}
}

// Count devuelve el número de nodos en el pool.
func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.nodes)
}

// CountByState devuelve el conteo por estado.
func (p *Pool) CountByState() map[NodeState]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make(map[NodeState]int)
	for _, n := range p.nodes {
		out[n.State]++
	}
	return out
}

// CountByZone devuelve el conteo por zona.
func (p *Pool) CountByZone() map[NodeZone]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make(map[NodeZone]int)
	for _, n := range p.nodes {
		out[n.Zone]++
	}
	return out
}

// HealthyCount devuelve el número de nodos usables.
func (p *Pool) HealthyCount() int {
	return len(p.ListUsable())
}

// Stats devuelve estadísticas del pool.
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalNodes:  len(p.nodes),
		ByState:     make(map[NodeState]int),
		ByZone:      make(map[NodeZone]int),
	}

	var totalLatency int64
	var latencyCount int64

	for _, n := range p.nodes {
		stats.ByState[n.State]++
		stats.ByZone[n.Zone]++
		stats.TotalConnections += n.TotalConnections
		stats.TotalFailures += n.FailedConnections

		if n.LastLatencyMs > 0 {
			totalLatency += n.LastLatencyMs
			latencyCount++
		}
	}

	if latencyCount > 0 {
		stats.AvgLatencyMs = totalLatency / latencyCount
	}

	return stats
}

// PoolStats lleva estadísticas del pool.
type PoolStats struct {
	TotalNodes       int
	ByState          map[NodeState]int
	ByZone           map[NodeZone]int
	TotalConnections uint64
	TotalFailures    uint64
	AvgLatencyMs     int64
}
