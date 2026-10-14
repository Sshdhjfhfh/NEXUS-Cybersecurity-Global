// Nexus IA v6.0 — Phantom Mixnet — Cliente
//
// El cliente construye circuitos mixnet y envía Sphinx packets.
// Cada paquete viaja por N saltos con delays aleatorios.
package mixnet

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Client es un cliente mixnet.
type Client struct {
	cfg     Config
	builder *MixCircuitBuilder
	pool    *MixCircuitPool
	mixer   *Mixer

	// Cover traffic
	coverGen *CoverTrafficGenerator

	mu     sync.RWMutex
	closed bool

	// Callbacks
	onPacketSent func(*SphinxPacket, *MixCircuit)
}

// NewClient crea un cliente mixnet.
func NewClient(cfg Config, directory *MixDirectory) *Client {
	builder := NewMixCircuitBuilder(cfg, directory)
	pool := NewMixCircuitPool(builder, 10)
	mixer := NewMixer(cfg)

	var coverGen *CoverTrafficGenerator
	if cfg.EnableCoverTraffic {
		coverGen = NewCoverTrafficGenerator(cfg.CoverTrafficRate)
	}

	return &Client{
		cfg:      cfg,
		builder:  builder,
		pool:     pool,
		mixer:    mixer,
		coverGen: coverGen,
	}
}

// SetPacketCallback establece el callback cuando se envía un paquete.
func (c *Client) SetPacketCallback(fn func(*SphinxPacket, *MixCircuit)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onPacketSent = fn
}

// Start inicia el mixer y el generador de cover traffic.
func (c *Client) Start() {
	c.mixer.Start()

	if c.coverGen != nil {
		c.coverGen.SetPacketCallback(func(p *SphinxPacket) {
			// Paquetes de cover traffic viajan por un circuito nuevo
			circuit, err := c.pool.GetOrCreate()
			if err == nil {
				c.notifySent(p, circuit)
			}
		})
		c.coverGen.Start()
	}
}

// Send envía un payload a través del mixnet.
func (c *Client) Send(payload []byte) (*SphinxPacket, *MixCircuit, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, nil, errors.New("client closed")
	}
	c.mu.RUnlock()

	// 1. Obtener circuito
	circuit, err := c.pool.GetOrCreate()
	if err != nil {
		return nil, nil, fmt.Errorf("get circuit: %w", err)
	}

	// 2. Construir Sphinx packet
	sb := NewSphinxBuilder(generateMasterKeyFromCircuit(circuit), len(circuit.Path))
	packet, err := sb.BuildPacket(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("build packet: %w", err)
	}

	// 3. Añadir al mixer (batching + delays)
	if err := c.mixer.Add(packet); err != nil {
		return nil, nil, fmt.Errorf("mixer add: %w", err)
	}

	// 4. Actualizar stats
	circuit.LastUsedAt = time.Now()
	circuit.PacketsSent++
	circuit.BytesSent += uint64(len(payload))

	// 5. Notificar
	c.notifySent(packet, circuit)

	return packet, circuit, nil
}

// notifySent dispara el callback onPacketSent.
func (c *Client) notifySent(packet *SphinxPacket, circuit *MixCircuit) {
	c.mu.RLock()
	cb := c.onPacketSent
	c.mu.RUnlock()
	if cb != nil {
		cb(packet, circuit)
	}
}

// Flush fuerza el envío inmediato de todos los paquetes pendientes.
func (c *Client) Flush() {
	c.mixer.Flush()
}

// CircuitPool devuelve el pool de circuitos.
func (c *Client) CircuitPool() *MixCircuitPool {
	return c.pool
}

// Mixer devuelve el mixer.
func (c *Client) Mixer() *Mixer {
	return c.mixer
}

// CoverGenerator devuelve el generador de cover traffic.
func (c *Client) CoverGenerator() *CoverTrafficGenerator {
	return c.coverGen
}

// Stats devuelve estadísticas del cliente.
func (c *Client) Stats() MixnetStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := MixnetStats{
		TotalCircuits: c.pool.Size(),
		ByNode:        make(map[NodeID]*NodeStats),
	}

	mixerStats := c.mixer.Stats()
	stats.TotalPacketsSent = mixerStats.TotalReceived
	stats.TotalDropped = mixerStats.TotalDropped
	stats.AvgDelayMs = mixerStats.AvgDelayMs

	if c.coverGen != nil {
		coverStats := c.coverGen.Stats()
		stats.TotalCoverPackets = coverStats.TotalGenerated
	}

	return stats
}

// Close cierra el cliente y todos los recursos.
// Es idempotente: llamadas múltiples son seguras.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// Cerrar cover traffic primero (para de generar paquetes)
	if c.coverGen != nil {
		_ = c.coverGen.Close()
	}

	// Cerrar mixer (cancela processBatch si está durmiendo)
	_ = c.mixer.Close()

	// Cerrar todos los circuitos
	c.pool.CloseAll()

	return nil
}

// generateMasterKeyFromCircuit deriva una master key del circuito.
func generateMasterKeyFromCircuit(c *MixCircuit) []byte {
	key := make([]byte, 32)
	// Concatenar los bytes de los primeros IDs del path
	for i, node := range c.Path {
		for j := 0; j < len(node) && i*32+j < 32; j++ {
			key[i*32+j] = node[j]
		}
	}
	// Rellenar con el ID si falta
	for i := 0; i < len(c.ID) && i < 32; i++ {
		key[i] ^= c.ID[i]
	}
	return key
}
