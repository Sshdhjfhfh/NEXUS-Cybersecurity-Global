// Nexus IA v6.0 — ANS Service (API pública)
//
// Une el Registry (memoria) con el Store (PostgreSQL) y expone la API
// pública del ANS:
//
//   - Register(ctx, req) (*Entry, error)
//   - Lookup(ctx, name) (*LookupResult, error)
//   - LookupByAgentID(ctx, agentID) (*LookupResult, error)
//   - LookupByDID(ctx, did) (*LookupResult, error)
//   - UpdateStatus(ctx, name, status) error
//   - UpdateLastAction(ctx, name, action) error
//   - Deregister(ctx, name) error
//
// REGLA ZERO TRUST:
//   Si un agente NO está en el ANS → NO puede actuar.
//
// El Service es thread-safe (delegado al Registry).
package ans

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Service es la API pública del Agent Name Service.
type Service struct {
	cfg      Config
	registry *Registry
	store    *Store
	mu       sync.RWMutex
}

// NewService construye el servicio ANS.
func NewService(cfg Config, registry *Registry, store *Store) *Service {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Service{
		cfg:      cfg,
		registry: registry,
		store:    store,
	}
}

// Register registra un agente en el ANS.
//
// Reglas:
//   - Si RequireCert y no hay cert X.509 → error
//   - Si el AgentID ya está registrado con otro nombre → error
//   - La entrada se persiste en DB y en el Registry
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*Entry, error) {
	// Validaciones
	if req.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id cannot be empty")
	}
	if req.AgentType == "" {
		return nil, fmt.Errorf("agent_type cannot be empty")
	}
	if s.cfg.RequireCert && len(req.CertX509PEM) == 0 {
		return nil, fmt.Errorf("certificate X.509 required but not provided")
	}

	name := NewAgentName(req.Name)
	if !name.IsValid() {
		return nil, fmt.Errorf("invalid agent name: %q", req.Name)
	}

	now := time.Now()
	entry := &Entry{
		Name:           name,
		AgentID:        req.AgentID,
		AgentType:      req.AgentType,
		DID:            req.DID,
		CertX509PEM:    req.CertX509PEM,
		PublicKey:      req.PublicKey,
		AllowedActions: req.AllowedActions,
		AllowedAssets:  req.AllowedAssets,
		Status:         StatusActive,
		LastSeen:       now,
		RegisteredAt:   now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(s.cfg.EntryTTL),
	}

	// Añadir al Registry
	if err := s.registry.Add(entry); err != nil {
		return nil, fmt.Errorf("registry add: %w", err)
	}

	// Persistir en DB (si hay store)
	if s.store != nil {
		if err := s.store.SaveEntry(ctx, entry); err != nil {
			// Revertir registry
			s.registry.Remove(name)
			return nil, fmt.Errorf("store save: %w", err)
		}
	}

	return entry, nil
}

// Lookup busca un agente por nombre.
//
// Devuelve LookupResult con Found=false si no existe.
// Registra la consulta en stats.
func (s *Service) Lookup(ctx context.Context, name AgentName) (*LookupResult, error) {
	entry, ok := s.registry.Get(name)

	if !ok {
		// Fallback a DB
		if s.store != nil {
			var err error
			entry, err = s.store.GetEntry(ctx, name)
			if err != nil {
				return nil, err
			}
			if entry != nil {
				// Re-hidratar el Registry
				_ = s.registry.Add(entry)
				ok = true
			}
		}
	}

	if !ok || entry == nil {
		s.registry.RecordLookup(false, false)
		return &LookupResult{
			Found:  false,
			Reason: "agent not registered",
		}, nil
	}

	// Verificar validez
	if !entry.IsValid() {
		s.registry.RecordLookup(true, true)
		return &LookupResult{
			Found:  true,
			Entry:  entry,
			Reason: "agent found but not valid (status=" + string(entry.Status) + ")",
		}, nil
	}

	s.registry.RecordLookup(true, false)
	return &LookupResult{
		Found: true,
		Entry: entry,
	}, nil
}

// LookupByAgentID busca por AgentID.
func (s *Service) LookupByAgentID(ctx context.Context, agentID string) (*LookupResult, error) {
	entry, ok := s.registry.GetByAgentID(agentID)
	if !ok {
		return &LookupResult{Found: false, Reason: "agent_id not registered"}, nil
	}
	return s.Lookup(ctx, entry.Name)
}

// LookupByDID busca por DID.
func (s *Service) LookupByDID(ctx context.Context, did string) (*LookupResult, error) {
	entry, ok := s.registry.GetByDID(did)
	if !ok {
		return &LookupResult{Found: false, Reason: "DID not registered"}, nil
	}
	return s.Lookup(ctx, entry.Name)
}

// CanAct comprueba si un agente puede ejecutar una acción.
//
// Implementa la regla Zero Trust:
//   Si un agente NO está en el ANS → NO puede actuar.
//
// Devuelve:
//   - true si el agente está registrado, activo y tiene el permiso
//   - false en cualquier otro caso
func (s *Service) CanAct(ctx context.Context, name AgentName, action string) (bool, string) {
	result, err := s.Lookup(ctx, name)
	if err != nil {
		return false, "lookup error: " + err.Error()
	}
	if !result.Found {
		return false, "agent not registered in ANS"
	}
	if !result.Entry.IsValid() {
		return false, "agent not valid: " + result.Reason
	}
	if !result.Entry.CanAct(action) {
		return false, "action not permitted: " + action
	}
	return true, "OK"
}

// UpdateStatus actualiza el estado de un agente.
func (s *Service) UpdateStatus(ctx context.Context, name AgentName, status AgentStatus) error {
	if err := s.registry.UpdateStatus(name, status); err != nil {
		return err
	}

	entry, ok := s.registry.Get(name)
	if !ok {
		return fmt.Errorf("entry vanished after update")
	}

	if s.store != nil {
		return s.store.SaveEntry(ctx, entry)
	}
	return nil
}

// UpdateLastAction actualiza la última acción de un agente.
func (s *Service) UpdateLastAction(ctx context.Context, name AgentName, action string) error {
	if err := s.registry.UpdateLastAction(name, action); err != nil {
		return err
	}

	entry, ok := s.registry.Get(name)
	if !ok {
		return fmt.Errorf("entry vanished after update")
	}

	if s.store != nil {
		return s.store.SaveEntry(ctx, entry)
	}
	return nil
}

// Deregister elimina un agente del ANS.
func (s *Service) Deregister(ctx context.Context, name AgentName) error {
	if !s.registry.Remove(name) {
		return fmt.Errorf("agent %q not found", name)
	}
	if s.store != nil {
		return s.store.DeleteEntry(ctx, name)
	}
	return nil
}

// List devuelve todas las entradas.
func (s *Service) List() []*Entry {
	return s.registry.List()
}

// ListByStatus devuelve las entradas con un estado.
func (s *Service) ListByStatus(status AgentStatus) []*Entry {
	return s.registry.ListByStatus(status)
}

// ListByType devuelve las entradas de un AgentType.
func (s *Service) ListByType(agentType string) []*Entry {
	return s.registry.ListByType(agentType)
}

// Count devuelve el número de agentes registrados.
func (s *Service) Count() int {
	return s.registry.Count()
}

// CountByStatus devuelve el conteo por estado.
func (s *Service) CountByStatus() map[AgentStatus]int {
	return s.registry.CountByStatus()
}

// Stats devuelve las estadísticas de uso.
func (s *Service) Stats() LookupStats {
	return s.registry.Stats()
}

// Registry devuelve el Registry subyacente.
func (s *Service) Registry() *Registry {
	return s.registry
}
