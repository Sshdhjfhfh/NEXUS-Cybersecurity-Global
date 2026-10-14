// Nexus IA v6.0 — ANS Registry (directorio en memoria)
//
// El Registry mantiene el directorio de agentes en memoria con
// thread-safety. Es la fuente de verdad rápida para lookups.
//
// Se sincroniza con PostgreSQL para persistencia (store.go).
package ans

import (
	"fmt"
	"sync"
	"time"
)

// Registry mantiene el directorio de agentes en memoria.
type Registry struct {
	mu sync.RWMutex

	// byName mapea AgentName → *Entry
	byName map[AgentName]*Entry

	// byAgentID mapea AgentID → AgentName (índice inverso)
	byAgentID map[string]AgentName

	// byDID mapea DID → AgentName (índice inverso)
	byDID map[string]AgentName

	// stats de uso
	stats LookupStats
}

// NewRegistry crea un Registry vacío.
func NewRegistry() *Registry {
	return &Registry{
		byName:    make(map[AgentName]*Entry),
		byAgentID: make(map[string]AgentName),
		byDID:     make(map[string]AgentName),
	}
}

// Add añade o actualiza una entrada.
func (r *Registry) Add(entry *Entry) error {
	if entry == nil {
		return fmt.Errorf("nil entry")
	}
	if !entry.Name.IsValid() {
		return fmt.Errorf("invalid agent name: %q", entry.Name)
	}
	if entry.AgentID == "" {
		return fmt.Errorf("empty agent id")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Si ya existía con otro AgentID, limpiar índices viejos
	if old, ok := r.byName[entry.Name]; ok {
		delete(r.byAgentID, old.AgentID)
		delete(r.byDID, old.DID)
	}

	entry.UpdatedAt = time.Now()
	if entry.RegisteredAt.IsZero() {
		entry.RegisteredAt = entry.UpdatedAt
	}

	// Copia defensiva
	cpy := *entry
	r.byName[entry.Name] = &cpy
	r.byAgentID[entry.AgentID] = entry.Name
	if entry.DID != "" {
		r.byDID[entry.DID] = entry.Name
	}

	return nil
}

// Get recupera una entrada por AgentName.
func (r *Registry) Get(name AgentName) (*Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.byName[name]
	if !ok {
		return nil, false
	}
	cpy := *e
	return &cpy, true
}

// GetByAgentID recupera una entrada por AgentID.
func (r *Registry) GetByAgentID(agentID string) (*Entry, bool) {
	r.mu.RLock()
	name, ok := r.byAgentID[agentID]
	r.mu.RUnlock()

	if !ok {
		return nil, false
	}
	return r.Get(name)
}

// GetByDID recupera una entrada por DID.
func (r *Registry) GetByDID(did string) (*Entry, bool) {
	r.mu.RLock()
	name, ok := r.byDID[did]
	r.mu.RUnlock()

	if !ok {
		return nil, false
	}
	return r.Get(name)
}

// Remove elimina una entrada.
func (r *Registry) Remove(name AgentName) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.byName[name]
	if !ok {
		return false
	}
	delete(r.byName, name)
	delete(r.byAgentID, e.AgentID)
	if e.DID != "" {
		delete(r.byDID, e.DID)
	}
	return true
}

// UpdateStatus actualiza el estado de una entrada.
func (r *Registry) UpdateStatus(name AgentName, status AgentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.byName[name]
	if !ok {
		return fmt.Errorf("agent %q not found", name)
	}
	e.Status = status
	e.UpdatedAt = time.Now()
	return nil
}

// UpdateLastAction actualiza la última acción de una entrada.
func (r *Registry) UpdateLastAction(name AgentName, action string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.byName[name]
	if !ok {
		return fmt.Errorf("agent %q not found", name)
	}
	e.LastAction = action
	e.LastSeen = time.Now()
	e.UpdatedAt = time.Now()
	return nil
}

// List devuelve todas las entradas.
func (r *Registry) List() []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*Entry, 0, len(r.byName))
	for _, e := range r.byName {
		cpy := *e
		out = append(out, &cpy)
	}
	return out
}

// ListByStatus devuelve las entradas con un estado dado.
func (r *Registry) ListByStatus(status AgentStatus) []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*Entry
	for _, e := range r.byName {
		if e.Status == status {
			cpy := *e
			out = append(out, &cpy)
		}
	}
	return out
}

// ListByType devuelve las entradas de un AgentType.
func (r *Registry) ListByType(agentType string) []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []*Entry
	for _, e := range r.byName {
		if e.AgentType == agentType {
			cpy := *e
			out = append(out, &cpy)
		}
	}
	return out
}

// Count devuelve el número total de agentes registrados.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byName)
}

// CountByStatus devuelve el conteo por estado.
func (r *Registry) CountByStatus() map[AgentStatus]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[AgentStatus]int)
	for _, e := range r.byName {
		out[e.Status]++
	}
	return out
}

// PruneExpired elimina las entradas expiradas.
// Devuelve el número de entradas eliminadas.
func (r *Registry) PruneExpired() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	count := 0
	for name, e := range r.byName {
		if now.After(e.ExpiresAt) {
			delete(r.byName, name)
			delete(r.byAgentID, e.AgentID)
			if e.DID != "" {
				delete(r.byDID, e.DID)
			}
			count++
		}
	}
	return count
}

// RecordLookup registra una consulta en las estadísticas.
func (r *Registry) RecordLookup(found, denied bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats.TotalLookups++
	if found {
		r.stats.TotalFound++
	} else {
		r.stats.TotalNotFound++
	}
	if denied {
		r.stats.TotalDenied++
	}
}

// Stats devuelve las estadísticas de uso.
func (r *Registry) Stats() LookupStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}
