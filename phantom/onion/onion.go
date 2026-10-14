// Nexus IA v6.0 — Phantom Onion — Onion Router 3 saltos propio
//
// Sistema de enrutamiento en capas (onion) con 3 saltos.
// Cada capa cifra para un nodo específico, y solo ese nodo puede
// descifrar su capa. Ni el primer nodo sabe quién eres, ni el
// último sabe de dónde vienes.
//
// Características:
//   - 3 saltos (mínimo; configurable)
//   - AES-256-GCM por capa
//   - Circuito nuevo por sesión (claves efímeras)
//   - Cada nodo solo conoce al siguiente y al anterior
//   - No usa Tor (implementación propia)
//
// Nivel 1-2 — Phantom — Sub-módulo Onion
package onion

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════
// Constantes
// ═══════════════════════════════════════════════════════════════

const (
	// DefaultHops es el número por defecto de saltos.
	DefaultHops = 3

	// MinHops es el mínimo de saltos para seguridad.
	MinHops = 2

	// MaxHops es el máximo razonable (más = más lento).
	MaxHops = 7

	// CellSize es el tamaño fijo de una celda onion.
	// Todas las celdas tienen el mismo tamaño (anti-fingerprinting).
	CellSize = 512

	// CircuitTTL es la vida útil de un circuito.
	CircuitTTL = 10 * time.Minute

	// KeyRotateInterval es cada cuánto se rotan claves de capa.
	KeyRotateInterval = 5 * time.Minute
)

// ═══════════════════════════════════════════════════════════════
// Tipos
// ═══════════════════════════════════════════════════════════════

// NodeID identifica un nodo de la red onion.
type NodeID string

// HopIndex es el índice de un salto (0 = entrada, N-1 = salida).
type HopIndex int

// Layer es una capa de cifrado en el circuito.
type Layer struct {
	// Identificación
	Hop     HopIndex
	NodeID  NodeID

	// Claves (AES-256-GCM)
	EncryptionKey []byte // 32 bytes
	MACKey        []byte // 32 bytes

	// Nonces por capa
	Nonce []byte // 12 bytes
}

// Circuit es un circuito onion (camino de 3+ nodos).
type Circuit struct {
	// ID único
	ID string

	// Nodos en orden (entrada → salida)
	Path []NodeID

	// Capas (una por hop)
	Layers []*Layer

	// Timing
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt time.Time

	// Estadísticas
	BytesSent     uint64
	BytesReceived uint64
	CellsSent     uint64
	CellsReceived uint64

	// Estado
	Closed bool
}

// IsExpired devuelve true si el circuito expiró.
func (c *Circuit) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// TTL devuelve el tiempo restante del circuito.
func (c *Circuit) TTL() time.Duration {
	if c.IsExpired() {
		return 0
	}
	return time.Until(c.ExpiresAt)
}

// HopCount devuelve el número de saltos.
func (c *Circuit) HopCount() int {
	return len(c.Path)
}

// Cell es una unidad de transmisión en la red onion.
// Todas las celdas tienen el mismo tamaño (anti-fingerprinting).
type Cell struct {
	// Header
	CircuitID string
	Command   CellCommand

	// Payload cifrado
	Payload []byte // Tamaño variable, se rellena a CellSize

	// Padding (se rellena hasta CellSize)
	Padding []byte

	// Metadatos
	Timestamp time.Time
}

// CellCommand es el comando de una celda.
type CellCommand string

const (
	// CommandData lleva datos.
	CommandData CellCommand = "DATA"

	// CommandCreate crea una capa del circuito.
	CommandCreate CellCommand = "CREATE"

	// CommandCreated confirma la creación.
	CommandCreated CellCommand = "CREATED"

	// CommandExtend extiende el circuito.
	CommandExtend CellCommand = "EXTEND"

	// CommandExtended confirma la extensión.
	CommandExtended CellCommand = "EXTENDED"

	// CommandDestroy destruye el circuito.
	CommandDestroy CellCommand = "DESTROY"

	// CommandPadding es relleno (dummy).
	CommandPadding CellCommand = "PADDING"
)

// ═══════════════════════════════════════════════════════════════
// Configuración
// ═══════════════════════════════════════════════════════════════

// Config configura el onion router.
type Config struct {
	// Número de saltos por circuito
	Hops int

	// Directorio de nodos disponibles
	Directory *Directory

	// TTL del circuito
	CircuitTTL time.Duration

	// Rotación de claves
	KeyRotateInterval time.Duration

	// ¿Rellenar celdas al tamaño fijo?
	PadCells bool

	// Tamaño de la celda (si PadCells)
	CellSize int
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		Hops:              DefaultHops,
		CircuitTTL:        CircuitTTL,
		KeyRotateInterval: KeyRotateInterval,
		PadCells:          true,
		CellSize:          CellSize,
	}
}

// ═══════════════════════════════════════════════════════════════
// Directory
// ═══════════════════════════════════════════════════════════════

// NodeInfo describe un nodo disponible en la red onion.
type NodeInfo struct {
	ID       NodeID
	Endpoint string
	Zone     string

	// Claves públicas del nodo (para handshake con él)
	X25519PublicKey   []byte // 32 bytes
	Kyber768PublicKey []byte // para handshake post-cuántico

	// Estado
	Active bool
	LatencyMs int64

	// Metadatos
	Bandwidth int64 // bytes/segundo
	Uptime    time.Duration
}

// IsUsable devuelve true si el nodo se puede usar.
func (n *NodeInfo) IsUsable() bool {
	return n.Active
}

// Directory mantiene el directorio de nodos disponibles.
type Directory struct {
	nodes map[NodeID]*NodeInfo
}

// NewDirectory crea un directorio vacío.
func NewDirectory() *Directory {
	return &Directory{
		nodes: make(map[NodeID]*NodeInfo),
	}
}

// Add añade un nodo al directorio.
func (d *Directory) Add(node *NodeInfo) {
	if node == nil || node.ID == "" {
		return
	}
	d.nodes[node.ID] = node
}

// Get recupera un nodo por ID.
func (d *Directory) Get(id NodeID) (*NodeInfo, bool) {
	n, ok := d.nodes[id]
	return n, ok
}

// Remove elimina un nodo.
func (d *Directory) Remove(id NodeID) {
	delete(d.nodes, id)
}

// List devuelve todos los nodos.
func (d *Directory) List() []*NodeInfo {
	out := make([]*NodeInfo, 0, len(d.nodes))
	for _, n := range d.nodes {
		out = append(out, n)
	}
	return out
}

// ListActive devuelve los nodos activos.
func (d *Directory) ListActive() []*NodeInfo {
	out := make([]*NodeInfo, 0)
	for _, n := range d.nodes {
		if n.IsUsable() {
			out = append(out, n)
		}
	}
	return out
}

// Count devuelve el número de nodos.
func (d *Directory) Count() int {
	return len(d.nodes)
}

// CountActive devuelve el número de nodos activos.
func (d *Directory) CountActive() int {
	return len(d.ListActive())
}

// ═══════════════════════════════════════════════════════════════
// Errores
// ═══════════════════════════════════════════════════════════════

// Errores específicos del módulo.
var (
	ErrNoNodes           = "no nodes available in directory"
	ErrNotEnoughNodes    = "not enough nodes for requested hops"
	ErrInvalidHops       = "invalid number of hops"
	ErrCircuitExpired    = "circuit expired"
	ErrCircuitClosed     = "circuit closed"
	ErrInvalidLayer      = "invalid layer"
	ErrDecryptionFailed  = "layer decryption failed"
)
