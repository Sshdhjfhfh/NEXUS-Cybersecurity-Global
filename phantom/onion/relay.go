// Nexus IA v6.0 — Phantom Onion — Relay
//
// Un relay es un nodo que:
//   1. Recibe una celda del nodo anterior
//   2. Descifra SU capa (y solo la suya)
//   3. Envía la celda resultante al siguiente nodo
//
// Cada relay solo sabe:
//   - De qué nodo (anterior) vino la celda
//   - A qué nodo (siguiente) va la celda
//
// NO SABE:
//   - Quién es el cliente original
//   - Cuál es el destino final
//   - Qué contiene el payload (está cifrado con capas más internas)
package onion

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Relay es un nodo relay de la red onion.
type Relay struct {
	// Identificación
	ID       NodeID
	Endpoint string

	// Clave de capa (32 bytes AES-256)
	LayerKey []byte

	// Capa
	cipher *LayerCipher

	// Routing: mapa de circuitID → nextNodeID
	mu    sync.RWMutex
	next  map[string]NodeID
	prev  map[string]NodeID

	// Estadísticas
	stats RelayStats

	// Estado
	closed bool
}

// NewRelay crea un relay.
func NewRelay(id NodeID, endpoint string, layerKey []byte) (*Relay, error) {
	cipher, err := NewLayerCipherFromKey(layerKey, 0)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	return &Relay{
		ID:       id,
		Endpoint: endpoint,
		LayerKey: append([]byte{}, layerKey...),
		cipher:   cipher,
		next:     make(map[string]NodeID),
		prev:     make(map[string]NodeID),
	}, nil
}

// RegisterCircuit registra el routing de un circuito en este relay.
func (r *Relay) RegisterCircuit(circuitID string, nextNode NodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next[circuitID] = nextNode
}

// ProcessCell procesa una celda entrante.
//
//   1. Descifra SU capa
//   2. Determina el siguiente nodo
//   3. Devuelve el payload descifrado y el siguiente nodo
func (r *Relay) ProcessCell(circuitID string, encrypted []byte) ([]byte, NodeID, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, "", errors.New("relay closed")
	}
	nextNode, hasNext := r.next[circuitID]
	r.mu.RUnlock()

	// 1. Descifrar nuestra capa
	decrypted, err := r.cipher.Decrypt(encrypted)
	if err != nil {
		r.mu.Lock()
		r.stats.DecryptionErrors++
		r.mu.Unlock()
		return nil, "", fmt.Errorf("decrypt layer: %w", err)
	}

	// 2. Actualizar stats
	r.mu.Lock()
	r.stats.CellsProcessed++
	r.stats.BytesProcessed += uint64(len(encrypted))
	r.stats.LastProcessedAt = time.Now()
	r.mu.Unlock()

	// 3. Devolver el payload y el siguiente nodo
	// Si no hay siguiente, este relay es el último y el payload está en claro
	if !hasNext {
		return decrypted, "", nil
	}
	return decrypted, nextNode, nil
}

// EncryptReply cifra una respuesta que va de vuelta al cliente.
// El relay aplica SU capa de cifrado a la respuesta.
func (r *Relay) EncryptReply(reply []byte) ([]byte, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, errors.New("relay closed")
	}
	cipher := r.cipher
	r.mu.RUnlock()

	return cipher.Encrypt(reply)
}

// ID devuelve el ID del relay.
func (r *Relay) NodeID() NodeID {
	return r.ID
}

// Endpoint devuelve el endpoint del relay.
func (r *Relay) NodeEndpoint() string {
	return r.Endpoint
}

// Stats devuelve las estadísticas del relay.
func (r *Relay) Stats() RelayStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// Close cierra el relay.
func (r *Relay) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	// Zero out layer key
	for i := range r.LayerKey {
		r.LayerKey[i] = 0
	}
	r.LayerKey = nil
	r.cipher = nil
	r.closed = true
	return nil
}

// RelayStats lleva las estadísticas de un relay.
type RelayStats struct {
	CellsProcessed    uint64
	BytesProcessed    uint64
	DecryptionErrors  uint64
	LastProcessedAt   time.Time
}
