// Nexus IA v6.0 — Tests del ANS
package ans

import (
	"context"
	"testing"
	"time"
)

// TestAgentName verifica la construcción de nombres.
func TestAgentName(t *testing.T) {
	name := NewAgentName("red-agent-01")
	if string(name) != "nexus.local/agents/red-agent-01" {
		t.Errorf("unexpected full name: %s", name)
	}
	if name.Short() != "red-agent-01" {
		t.Errorf("unexpected short name: %s", name.Short())
	}
	if !name.IsValid() {
		t.Error("name should be valid")
	}

	// Con slash inicial
	name2 := NewAgentName("/scout-01")
	if name2.Short() != "scout-01" {
		t.Errorf("unexpected short: %s", name2.Short())
	}

	// Inválido
	bad := AgentName("bad-format")
	if bad.IsValid() {
		t.Error("bad name should be invalid")
	}
}

// TestRegistryAdd verifica añadir al registry.
func TestRegistryAdd(t *testing.T) {
	r := NewRegistry()

	entry := &Entry{
		Name:      NewAgentName("test-01"),
		AgentID:   "test-01",
		AgentType: "RED",
		DID:       "did:nexus:test-01",
		Status:    StatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if err := r.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if r.Count() != 1 {
		t.Errorf("count = %d, want 1", r.Count())
	}

	got, ok := r.Get(entry.Name)
	if !ok {
		t.Fatal("Get failed")
	}
	if got.AgentID != "test-01" {
		t.Errorf("AgentID = %s, want test-01", got.AgentID)
	}
}

// TestRegistryGetByAgentID verifica el índice inverso.
func TestRegistryGetByAgentID(t *testing.T) {
	r := NewRegistry()

	entry := &Entry{
		Name:      NewAgentName("test-02"),
		AgentID:   "test-02",
		AgentType: "BLUE",
		DID:       "did:nexus:test-02",
		Status:    StatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = r.Add(entry)

	got, ok := r.GetByAgentID("test-02")
	if !ok {
		t.Fatal("GetByAgentID failed")
	}
	if got.Name != entry.Name {
		t.Errorf("wrong entry: %s", got.Name)
	}
}

// TestRegistryRemove verifica eliminar del registry.
func TestRegistryRemove(t *testing.T) {
	r := NewRegistry()
	entry := &Entry{
		Name:      NewAgentName("test-03"),
		AgentID:   "test-03",
		AgentType: "RED",
		Status:    StatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = r.Add(entry)

	if !r.Remove(entry.Name) {
		t.Error("Remove failed")
	}
	if r.Count() != 0 {
		t.Errorf("count = %d, want 0", r.Count())
	}
}

// TestServiceRegister verifica el registro completo.
func TestServiceRegister(t *testing.T) {
	svc := NewService(DefaultConfig(), nil, nil)
	ctx := context.Background()

	entry, err := svc.Register(ctx, RegisterRequest{
		Name:           "red-agent-01",
		AgentID:        "red-agent-01",
		AgentType:      "RED",
		DID:            "did:nexus:red-agent-01",
		CertX509PEM:    []byte("fake-cert-pem"),
		AllowedActions: []string{"SCAN", "EXPLOIT"},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if entry.Status != StatusActive {
		t.Errorf("status = %s, want ACTIVE", entry.Status)
	}
	if svc.Count() != 1 {
		t.Errorf("count = %d, want 1", svc.Count())
	}
}

// TestServiceRegisterRequiresCert verifica la validación.
func TestServiceRegisterRequiresCert(t *testing.T) {
	svc := NewService(DefaultConfig(), nil, nil)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		Name:      "no-cert-agent",
		AgentID:   "no-cert-agent",
		AgentType: "RED",
		// Sin CertX509PEM
	})
	if err == nil {
		t.Error("Register should fail without certificate")
	}
}

// TestServiceLookup verifica el lookup.
func TestServiceLookup(t *testing.T) {
	svc := NewService(DefaultConfig(), nil, nil)
	ctx := context.Background()

	// Registrar
	_, _ = svc.Register(ctx, RegisterRequest{
		Name:           "red-agent-01",
		AgentID:        "red-agent-01",
		AgentType:      "RED",
		DID:            "did:nexus:red-agent-01",
		CertX509PEM:    []byte("fake-cert-pem"),
		AllowedActions: []string{"SCAN"},
	})

	// Lookup válido
	result, err := svc.Lookup(ctx, NewAgentName("red-agent-01"))
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !result.Found {
		t.Error("agent should be found")
	}
	if result.Entry.AgentID != "red-agent-01" {
		t.Errorf("wrong agent: %s", result.Entry.AgentID)
	}

	// Lookup no existente
	result, err = svc.Lookup(ctx, NewAgentName("ghost-agent"))
	if err != nil {
		t.Fatalf("Lookup ghost failed: %v", err)
	}
	if result.Found {
		t.Error("ghost agent should NOT be found (Zero Trust)")
	}
}

// TestServiceCanAct verifica la regla Zero Trust.
func TestServiceCanAct(t *testing.T) {
	svc := NewService(DefaultConfig(), nil, nil)
	ctx := context.Background()

	_, _ = svc.Register(ctx, RegisterRequest{
		Name:           "red-agent-01",
		AgentID:        "red-agent-01",
		AgentType:      "RED",
		DID:            "did:nexus:red-agent-01",
		CertX509PEM:    []byte("fake-cert-pem"),
		AllowedActions: []string{"SCAN", "EXPLOIT"},
	})

	// Acción permitida
	ok, reason := svc.CanAct(ctx, NewAgentName("red-agent-01"), "SCAN")
	if !ok {
		t.Errorf("SCAN should be allowed, reason: %s", reason)
	}

	// Acción no permitida
	ok, reason = svc.CanAct(ctx, NewAgentName("red-agent-01"), "ISOLATE_HOST")
	if ok {
		t.Errorf("ISOLATE_HOST should be denied")
	}
	if reason == "" {
		t.Error("reason should not be empty")
	}

	// Agente no registrado → NO puede actuar
	ok, reason = svc.CanAct(ctx, NewAgentName("ghost-agent"), "SCAN")
	if ok {
		t.Error("ghost agent should NOT be able to act (Zero Trust)")
	}
	if reason != "agent not registered in ANS" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

// TestServiceUpdateStatus verifica el cambio de estado.
func TestServiceUpdateStatus(t *testing.T) {
	svc := NewService(DefaultConfig(), nil, nil)
	ctx := context.Background()

	_, _ = svc.Register(ctx, RegisterRequest{
		Name:        "red-agent-01",
		AgentID:     "red-agent-01",
		AgentType:   "RED",
		DID:         "did:nexus:red-agent-01",
		CertX509PEM: []byte("fake-cert-pem"),
	})

	// Actualizar a quarantined
	if err := svc.UpdateStatus(ctx, NewAgentName("red-agent-01"), StatusQuarantined); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verificar que CanAct deniega
	ok, _ := svc.CanAct(ctx, NewAgentName("red-agent-01"), "SCAN")
	if ok {
		t.Error("quarantined agent should not be able to act")
	}
}

// TestEntryIsValid verifica la validez de entradas.
func TestEntryIsValid(t *testing.T) {
	now := time.Now()

	valid := &Entry{
		Status:      StatusActive,
		CertX509PEM: []byte("cert"),
		ExpiresAt:   now.Add(time.Hour),
	}
	if !valid.IsValid() {
		t.Error("active + cert + not expired should be valid")
	}

	// Sin cert
	noCert := &Entry{
		Status:    StatusActive,
		ExpiresAt: now.Add(time.Hour),
	}
	if noCert.IsValid() {
		t.Error("without cert should be invalid")
	}

	// Expirado
	expired := &Entry{
		Status:      StatusActive,
		CertX509PEM: []byte("cert"),
		ExpiresAt:   now.Add(-time.Hour),
	}
	if expired.IsValid() {
		t.Error("expired should be invalid")
	}

	// Quarantined
	quarantined := &Entry{
		Status:      StatusQuarantined,
		CertX509PEM: []byte("cert"),
		ExpiresAt:   now.Add(time.Hour),
	}
	if quarantined.IsValid() {
		t.Error("quarantined should be invalid")
	}

	// Revoked
	revoked := &Entry{
		Status:      StatusRevoked,
		CertX509PEM: []byte("cert"),
		ExpiresAt:   now.Add(time.Hour),
	}
	if revoked.IsValid() {
		t.Error("revoked should be invalid")
	}
}

// TestServiceStats verifica las estadísticas.
func TestServiceStats(t *testing.T) {
	svc := NewService(DefaultConfig(), nil, nil)
	ctx := context.Background()

	_, _ = svc.Register(ctx, RegisterRequest{
		Name:        "red-agent-01",
		AgentID:     "red-agent-01",
		AgentType:   "RED",
		CertX509PEM: []byte("cert"),
	})

	// 3 lookups: 2 found, 1 not found
	_, _ = svc.Lookup(ctx, NewAgentName("red-agent-01"))
	_, _ = svc.Lookup(ctx, NewAgentName("red-agent-01"))
	_, _ = svc.Lookup(ctx, NewAgentName("ghost"))

	stats := svc.Stats()
	if stats.TotalLookups != 3 {
		t.Errorf("TotalLookups = %d, want 3", stats.TotalLookups)
	}
	if stats.TotalFound != 2 {
		t.Errorf("TotalFound = %d, want 2", stats.TotalFound)
	}
	if stats.TotalNotFound != 1 {
		t.Errorf("TotalNotFound = %d, want 1", stats.TotalNotFound)
	}
}

// TestRegistryPruneExpired verifica la limpieza.
func TestRegistryPruneExpired(t *testing.T) {
	r := NewRegistry()

	// Añadir una expirada
	expired := &Entry{
		Name:      NewAgentName("old-agent"),
		AgentID:   "old-agent",
		AgentType: "RED",
		Status:    StatusActive,
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	_ = r.Add(expired)

	// Añadir una válida
	valid := &Entry{
		Name:      NewAgentName("new-agent"),
		AgentID:   "new-agent",
		AgentType: "RED",
		Status:    StatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = r.Add(valid)

	// Prune
	removed := r.PruneExpired()
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if r.Count() != 1 {
		t.Errorf("count = %d, want 1", r.Count())
	}
}
