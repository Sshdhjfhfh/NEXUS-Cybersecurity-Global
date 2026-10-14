// Nexus IA v6.0 — Phantom Mixnet — Mixnet Sphinx post-cuántico
//
// Un mixnet es el sistema de anonimato más avanzado que existe.
// Mientras el onion router protege el CONTENIDO, el mixnet protege
// también los METADATOS (timing).
//
// Diferencias clave con onion:
//   - Onion: capas de cifrado, cada nodo ve su capa
//   - Mixnet: capas + batching + delays aleatorios → rompe correlación temporal
//
// Propiedades:
//   - 5 saltos de mezcla
//   - Sphinx packet format (tamaño fijo)
//   - Kyber768 + X25519 híbrido (post-cuántico)
//   - Mezcla + reordenación + retardo Poisson
//   - Destruye correlación de timing incluso con vista global de red
//
// Basado en Loopix (el estado del arte) y katzenpost.
//
// Nivel 1-2 — Phantom — Sub-módulo Mixnet
package mixnet

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// DefaultHops es el número por defecto de saltos de mezcla.
	DefaultHops = 5

	// MinHops es el mínimo de saltos para seguridad.
	MinHops = 3

	// MaxHops es el máximo razonable.
	MaxHops = 10

	// PacketSize es el tamaño FIJO de un Sphinx packet.
	// Todos los paquetes tienen el mismo tamaño (anti-fingerprinting).
	PacketSize = 32768 // 32 KB

	// HeaderSize es el tamaño del header Sphinx.
	// Incluye: EphemeralPublicKey(32) + Kyber CT(1088) + RoutingInfo + MACs
	HeaderSize = 4096

	// RoutingInfoSize es el tamaño útil de routing info (dentro del header).
	RoutingInfoSize = 2900

	// MACsSize es el tamaño de los MACs por hop.
	MACsSize = 32

	// PayloadSize es el tamaño del payload.
	PayloadSize = PacketSize - HeaderSize

	// BatchInterval es cada cuánto se procesa un batch.
	BatchInterval = 100 * time.Millisecond

	// MinDelay es el retardo mínimo por hop.
	MinDelay = 50 * time.Millisecond

	// MaxDelay es el retardo máximo por hop.
	MaxDelay = 500 * time.Millisecond

	// PoissonLambda es el parámetro lambda de la distribución Poisson.
	PoissonLambda = 150.0 // ms
)

// ═══════════════════════════════════════════════════════════════
// Tipos base
// ═══════════════════════════════════════════════════════════════

// NodeID identifica un mix node.
type NodeID string

// HopIndex es el índice de un salto (0 = entrada, N-1 = salida).
type HopIndex int

// MixNode describe un nodo de mezcla.
type MixNode struct {
	ID       NodeID
	Endpoint string
	Zone     string

	// Claves públicas del nodo
	X25519PublicKey   []byte // 32 bytes
	Kyber768PublicKey []byte // para KEM post-cuántico

	// Estado
	Active    bool
	LatencyMs int64
	Uptime    time.Duration

	// Métricas de mezcla
	BatchSize      int
	AvgDelayMs     int64
	PacketsMixed   uint64
	PacketsDropped uint64
}

// IsUsable devuelve true si el nodo se puede usar.
func (n *MixNode) IsUsable() bool {
	return n.Active
}

// ═══════════════════════════════════════════════════════════════
// Sphinx Packet
// ═══════════════════════════════════════════════════════════════

// SphinxPacket es un paquete Sphinx (tamaño FIJO).
//
// Estructura:
//   - Header (4096 bytes):
//     - Ephemeral public key (X25519)
//     - Routing info cifrado (con claves de capa)
//     - MAC por hop (integridad)
//   - Payload (28672 bytes): cifrado end-to-end
//
// El tamaño FIJO es lo que hace imposible distinguir paquetes por tamaño.
type SphinxPacket struct {
	// Header
	EphemeralPublicKey []byte // 32 bytes (X25519)
	Kyber768Ciphertext []byte // para KEM
	RoutingInfo        []byte // routing info enmascarada
	MACs               []byte // MAC por hop

	// Payload cifrado end-to-end
	Payload []byte // 28672 bytes

	// Metadatos (NO viajan por la red, solo para el cliente)
	CreatedAt time.Time
	PacketID  string
}

// Size devuelve el tamaño total del paquete.
func (p *SphinxPacket) Size() int {
	return len(p.EphemeralPublicKey) + len(p.Kyber768Ciphertext) +
		len(p.RoutingInfo) + len(p.MACs) + len(p.Payload)
}

// IsValidSize comprueba que el paquete tenga el tamaño correcto.
func (p *SphinxPacket) IsValidSize() bool {
	return p.Size() == PacketSize
}

// ═══════════════════════════════════════════════════════════════
// Mix Circuit
// ═══════════════════════════════════════════════════════════════

// MixCircuit es un circuito mixnet (camino de 5+ nodos).
type MixCircuit struct {
	// ID único
	ID string

	// Nodos en orden (entrada → salida)
	Path []NodeID

	// Claves de capa por hop
	LayerKeys [][]byte // una por hop

	// Timing
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt time.Time

	// Estadísticas
	PacketsSent     uint64
	PacketsReceived uint64
	BytesSent       uint64
	BytesReceived   uint64

	// Estado
	Closed bool
}

// IsExpired devuelve true si el circuito expiró.
func (c *MixCircuit) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// TTL devuelve el tiempo restante del circuito.
func (c *MixCircuit) TTL() time.Duration {
	if c.IsExpired() {
		return 0
	}
	return time.Until(c.ExpiresAt)
}

// HopCount devuelve el número de saltos.
func (c *MixCircuit) HopCount() int {
	return len(c.Path)
}

// ═══════════════════════════════════════════════════════════════
// Configuración
// ═══════════════════════════════════════════════════════════════

// Config configura el mixnet.
type Config struct {
	// Número de saltos
	Hops int

	// Directorio de mix nodes
	Directory *MixDirectory

	// TTL del circuito
	CircuitTTL time.Duration

	// ¿Activar batching?
	EnableBatching bool

	// Tamaño del batch
	BatchSize int

	// ¿Activar delays Poisson?
	EnablePoissonDelays bool

	// ¿Activar cover traffic (dummy packets)?
	EnableCoverTraffic bool

	// Rate de cover traffic (paquetes por segundo)
	CoverTrafficRate float64
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		Hops:                DefaultHops,
		CircuitTTL:          15 * time.Minute,
		EnableBatching:      true,
		BatchSize:           10,
		EnablePoissonDelays: true,
		EnableCoverTraffic:  true,
		CoverTrafficRate:    5.0,
	}
}

// ═══════════════════════════════════════════════════════════════
// MixDirectory
// ═══════════════════════════════════════════════════════════════

// MixDirectory mantiene el directorio de mix nodes.
type MixDirectory struct {
	nodes map[NodeID]*MixNode
}

// NewMixDirectory crea un directorio vacío.
func NewMixDirectory() *MixDirectory {
	return &MixDirectory{
		nodes: make(map[NodeID]*MixNode),
	}
}

// Add añade un nodo al directorio.
func (d *MixDirectory) Add(node *MixNode) {
	if node == nil || node.ID == "" {
		return
	}
	d.nodes[node.ID] = node
}

// Get recupera un nodo por ID.
func (d *MixDirectory) Get(id NodeID) (*MixNode, bool) {
	n, ok := d.nodes[id]
	return n, ok
}

// Remove elimina un nodo.
func (d *MixDirectory) Remove(id NodeID) {
	delete(d.nodes, id)
}

// List devuelve todos los nodos.
func (d *MixDirectory) List() []*MixNode {
	out := make([]*MixNode, 0, len(d.nodes))
	for _, n := range d.nodes {
		out = append(out, n)
	}
	return out
}

// ListActive devuelve los nodos activos.
func (d *MixDirectory) ListActive() []*MixNode {
	out := make([]*MixNode, 0)
	for _, n := range d.nodes {
		if n.IsUsable() {
			out = append(out, n)
		}
	}
	return out
}

// Count devuelve el número de nodos.
func (d *MixDirectory) Count() int {
	return len(d.nodes)
}

// CountActive devuelve el número de nodos activos.
func (d *MixDirectory) CountActive() int {
	return len(d.ListActive())
}

// ═══════════════════════════════════════════════════════════════
// API del Mixnet
// ═══════════════════════════════════════════════════════════════

// Mixnet es la interfaz del mixnet.
type Mixnet interface {
	// Send envía un payload a través del mixnet.
	Send(payload []byte) (*SphinxPacket, *MixCircuit, error)

	// Receive recibe un payload del mixnet.
	Receive(circuitID string, packet *SphinxPacket) ([]byte, error)

	// CircuitPool devuelve el pool de circuitos.
	CircuitPool() *MixCircuitPool

	// Directory devuelve el directorio de nodos.
	Directory() *MixDirectory

	// Stats devuelve estadísticas globales.
	Stats() MixnetStats

	// Close cierra el mixnet.
	Close() error
}

// MixnetStats lleva estadísticas del mixnet.
type MixnetStats struct {
	TotalCircuits     int
	TotalPacketsSent  uint64
	TotalPacketsRecv  uint64
	TotalCoverPackets uint64
	TotalDropped      uint64
	AvgDelayMs        int64
	ByNode            map[NodeID]*NodeStats
}

// NodeStats lleva estadísticas por nodo.
type NodeStats struct {
	NodeID       NodeID
	PacketsMixed uint64
	BytesMixed   uint64
	AvgDelayMs   int64
}

// ═══════════════════════════════════════════════════════════════
// Errores
// ═══════════════════════════════════════════════════════════════

var (
	ErrNoNodes         = "no mix nodes available in directory"
	ErrNotEnoughNodes  = "not enough mix nodes for requested hops"
	ErrInvalidHops     = "invalid number of hops"
	ErrCircuitExpired  = "circuit expired"
	ErrCircuitClosed   = "circuit closed"
	ErrInvalidPacket   = "invalid Sphinx packet"
	ErrDecryptionFailed = "packet decryption failed"
)
