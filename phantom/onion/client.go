// Nexus IA v6.0 — Phantom Onion — Cliente
//
// El cliente construye circuitos y envía celdas cifradas por capas.
// Cada capa solo puede ser descifrada por el nodo correspondiente.
//
// Flujo de envío:
//   1. Cliente construye payload
//   2. Aplica N capas de cifrado (PeelOnion)
//   3. Envía la celda al primer nodo
//   4. Cada nodo descifra su capa y reenvía al siguiente
//   5. El último nodo ve el payload original
package onion

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Client es un cliente onion.
type Client struct {
	cfg    Config
	pool   *CircuitPool
	mu     sync.RWMutex
	closed bool
}

// NewClient crea un cliente onion con un directorio.
func NewClient(cfg Config, directory *Directory) *Client {
	builder := NewCircuitBuilder(cfg, directory)
	pool := NewCircuitPool(builder, 10)

	return &Client{
		cfg:  cfg,
		pool: pool,
	}
}

// Send envía un payload a través de un circuito onion.
//
//   1. Obtiene (o crea) un circuito
//   2. Aplica N capas de cifrado al payload
//   3. Devuelve el payload onion-ready
//
// En producción esto enviaría la celda al primer nodo por red.
// En esta implementación devuelve el payload para test.
func (c *Client) Send(payload []byte) ([]byte, *Circuit, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, nil, errors.New("client closed")
	}
	c.mu.RUnlock()

	// 1. Obtener circuito
	circuit, cipher, err := c.pool.GetOrCreate()
	if err != nil {
		return nil, nil, fmt.Errorf("get circuit: %w", err)
	}

	// 2. Cifrar capas
	onionPayload, err := cipher.PeelOnion(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("peel onion: %w", err)
	}

	// 3. Actualizar circuit stats
	circuit.LastUsedAt = time.Now()
	circuit.BytesSent += uint64(len(onionPayload))
	circuit.CellsSent++

	return onionPayload, circuit, nil
}

// DecryptReply descifra una respuesta que viene del circuito.
//
// Cuando un nodo responde, la respuesta se cifra con las mismas
// claves de capa pero en orden inverso. El cliente descifra todas
// las capas (porque las tiene todas).
func (c *Client) DecryptReply(circuitID string, data []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil, errors.New("client closed")
	}

	circuit, cipher, ok := c.pool.Get(circuitID)
	if !ok {
		return nil, errors.New("circuit not found")
	}
	if circuit.IsExpired() {
		return nil, errors.New(ErrCircuitExpired)
	}

	// Descifrar todas las capas (el cliente las conoce todas)
	plaintext, err := cipher.UnpeelAll(data)
	if err != nil {
		return nil, fmt.Errorf("unpeel all: %w", err)
	}

	// Actualizar stats
	circuit.BytesReceived += uint64(len(data))
	circuit.CellsReceived++

	return plaintext, nil
}

// Pool devuelve el pool de circuitos.
func (c *Client) Pool() *CircuitPool {
	return c.pool
}

// CircuitCount devuelve el número de circuitos activos.
func (c *Client) CircuitCount() int {
	return c.pool.Size()
}

// Close cierra el cliente y todos los circuitos.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.pool.CloseAll()
	c.closed = true
	return nil
}
