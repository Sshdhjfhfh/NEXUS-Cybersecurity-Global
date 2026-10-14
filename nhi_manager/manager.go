// Nexus IA v6.0 — NHI Manager (orquestador principal)
//
// Une todos los componentes:
//   - CertAuthority (X.509)
//   - DilithiumKeyPair (firma post-cuántica)
//   - JWTSigningKey (JWT)
//   - RevocationRegistry (revocación)
//   - Store (persistencia PostgreSQL)
//   - Audit trail firmado
//
// API principal:
//   - Register(ctx, agentID, agentType, allowedActions, allowedAssets)
//   - VerifyJWT(ctx, token)
//   - Audit(ctx, nhi, action, target, payload, result, reason)
//   - Revoke(ctx, agentID, reason, revokedBy)
package nhi_manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config configura el NHI Manager.
type Config struct {
	// TTL por defecto de las identidades
	DefaultCertTTL time.Duration
	DefaultJWTTTL  time.Duration

	// CA interna
	RequireX509 bool // Si es true, cada identidad recibe certificado X.509
}

// DefaultConfig devuelve una configuración sensata.
func DefaultConfig() Config {
	return Config{
		DefaultCertTTL: DefaultCertTTL,
		DefaultJWTTTL:  DefaultJWTTTL,
		RequireX509:    true,
	}
}

// Manager es el orquestador principal del NHI Manager.
type Manager struct {
	cfg Config

	// CA interna
	ca *CertAuthority

	// Registro de revocaciones
	revocations *RevocationRegistry

	// Persistencia
	store *Store

	// Clave de firma JWT
	jwtKey JWTSigningKey

	// Clave de firma del audit trail (por sistema)
	systemKey     *DilithiumKeyPair
	systemKeyID   string

	// Pool de PostgreSQL
	pool *pgxpool.Pool

	// Cache en memoria de identidades activas
	mu    sync.RWMutex
	cache map[string]*NHI
}

// NewManager construye un Manager desde un pool de PostgreSQL.
func NewManager(ctx context.Context, pool *pgxpool.Pool, cfg Config) (*Manager, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool cannot be nil")
	}

	// Generar CA interna
	ca, err := NewCertAuthority()
	if err != nil {
		return nil, fmt.Errorf("init CA: %w", err)
	}

	// Generar clave JWT
	jwtKey, err := GenerateSigningKey()
	if err != nil {
		return nil, fmt.Errorf("init JWT key: %w", err)
	}

	// Generar clave de firma del audit trail
	systemKey, err := GenerateDilithiumKeyPair()
	if err != nil {
		return nil, fmt.Errorf("init system key: %w", err)
	}

	systemKeyID := "nexus-audit-v1"

	// Crear Store
	store := NewStore(pool)

	// Migrar schema
	if err := store.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate store: %w", err)
	}

	// Registrar clave pública del sistema
	if err := store.SaveSigningKey(ctx, systemKeyID, systemKey.PublicKeyBytes()); err != nil {
		return nil, fmt.Errorf("save system key: %w", err)
	}

	return &Manager{
		cfg:         cfg,
		ca:          ca,
		revocations: NewRevocationRegistry(),
		store:       store,
		jwtKey:      jwtKey,
		systemKey:   systemKey,
		systemKeyID: systemKeyID,
		pool:        pool,
		cache:       make(map[string]*NHI),
	}, nil
}

// RegisterRequest encapsula los parámetros de registro.
type RegisterRequest struct {
	AgentID        string
	AgentType      AgentType
	AllowedActions []ActionType
	AllowedAssets  []string
}

// Register registra un nuevo agente y le asigna identidad criptográfica.
// Devuelve la NHI lista para usar.
func (m *Manager) Register(ctx context.Context, req RegisterRequest) (*NHI, error) {
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id cannot be empty")
	}
	if req.AgentType == "" {
		return nil, fmt.Errorf("agent_type cannot be empty")
	}

	// Generar DID único
	did := NewDID()

	// Generar par de claves Dilithium
	dilithiumKP, err := GenerateDilithiumKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate dilithium key: %w", err)
	}

	// Emitir certificado X.509
	var certPEM []byte
	if m.cfg.RequireX509 {
		certPEM, err = m.ca.IssueCert(req.AgentID, req.AgentType, did, m.cfg.DefaultCertTTL)
		if err != nil {
			return nil, fmt.Errorf("issue x509: %w", err)
		}
	}

	// Construir NHI
	now := time.Now()
	nhi := &NHI{
		AgentID:        req.AgentID,
		AgentType:      req.AgentType,
		DID:            did,
		CertX509PEM:    certPEM,
		PublicKey:      dilithiumKP.PublicKeyBytes(),
		AllowedActions: req.AllowedActions,
		AllowedAssets:  req.AllowedAssets,
		IssuedAt:       now,
		ExpiresAt:      now.Add(m.cfg.DefaultCertTTL),
	}

	// Emitir JWT
	token, err := IssueJWT(m.jwtKey, nhi, "", m.cfg.DefaultJWTTTL)
	if err != nil {
		return nil, fmt.Errorf("issue jwt: %w", err)
	}
	nhi.JWTToken = token

	// Persistir
	if err := m.store.SaveIdentity(ctx, nhi); err != nil {
		return nil, fmt.Errorf("save nhi: %w", err)
	}

	// Guardar en cache
	m.mu.Lock()
	m.cache[req.AgentID] = nhi
	m.mu.Unlock()

	return nhi, nil
}

// RefreshIdentity renueva el JWT de una identidad existente.
// No cambia el certificado X.509 (que se renueva por separado cada 24h).
func (m *Manager) RefreshIdentity(ctx context.Context, agentID string) (*NHI, error) {
	nhi, err := m.GetIdentity(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if nhi == nil {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	if nhi.IsRevoked() {
		return nil, fmt.Errorf("agent %s is revoked", agentID)
	}

	// Renovar JWT
	token, err := IssueJWT(m.jwtKey, nhi, "", m.cfg.DefaultJWTTTL)
	if err != nil {
		return nil, fmt.Errorf("issue jwt: %w", err)
	}
	nhi.JWTToken = token

	// Persistir cambio
	if err := m.store.SaveIdentity(ctx, nhi); err != nil {
		return nil, fmt.Errorf("save nhi: %w", err)
	}

	return nhi, nil
}

// GetIdentity recupera una identidad por AgentID.
// Primero busca en cache, luego en DB.
func (m *Manager) GetIdentity(ctx context.Context, agentID string) (*NHI, error) {
	m.mu.RLock()
	nhi := m.cache[agentID]
	m.mu.RUnlock()

	if nhi != nil {
		return nhi, nil
	}

	// Buscar en DB
	nhi, err := m.store.GetIdentity(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if nhi == nil {
		return nil, nil
	}

	// Guardar en cache
	m.mu.Lock()
	m.cache[agentID] = nhi
	m.mu.Unlock()

	return nhi, nil
}

// VerifyJWT verifica un token JWT.
// Devuelve los claims si el token es válido y el agente no está revocado.
func (m *Manager) VerifyJWT(ctx context.Context, token string) (*ClaimsNexus, error) {
	claims, err := VerifyJWT(m.jwtKey, token)
	if err != nil {
		return nil, err
	}

	// Verificar que no esté revocado
	if err := m.revocations.VerifyNotRevoked(claims); err != nil {
		return nil, err
	}

	// Verificar en DB (por si fue revocado en otra instancia)
	nhi, err := m.store.GetIdentity(ctx, claims.AgentID)
	if err != nil {
		return nil, err
	}
	if nhi == nil {
		return nil, fmt.Errorf("agent %s not found in store", claims.AgentID)
	}
	if nhi.IsRevoked() {
		return nil, fmt.Errorf("agent %s revoked in store", claims.AgentID)
	}

	return claims, nil
}

// Audit registra una acción firmada en el audit trail.
// El audit trail es INMUTABLE.
func (m *Manager) Audit(ctx context.Context, nhi *NHI, action ActionType, target string, payload []byte, result, reason string) (*AuditEntry, error) {
	if nhi == nil {
		return nil, fmt.Errorf("nhi cannot be nil")
	}

	entry := NewAuditEntry(nhi, action, target, payload, result, reason)

	// Firmar con la clave del sistema
	if err := entry.Sign(m.systemKey, m.systemKeyID); err != nil {
		return nil, fmt.Errorf("sign audit entry: %w", err)
	}

	// Persistir
	if err := m.store.SaveAudit(ctx, entry); err != nil {
		return nil, fmt.Errorf("save audit: %w", err)
	}

	return entry, nil
}

// Revoke revoca una identidad.
func (m *Manager) Revoke(ctx context.Context, agentID string, reason RevocationReason, message, revokedBy string) error {
	nhi, err := m.GetIdentity(ctx, agentID)
	if err != nil {
		return err
	}
	if nhi == nil {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Revocar en el registro
	record := m.revocations.Revoke(nhi, reason, message, revokedBy)

	// Persistir
	if err := m.store.SaveIdentity(ctx, nhi); err != nil {
		return fmt.Errorf("save revoked nhi: %w", err)
	}

	// Registrar en audit
	_, auditErr := m.Audit(ctx, nhi, "REVOKE" /* pseudo-action */, agentID, nil, "ok", string(reason))
	if auditErr != nil {
		// No fallar si el audit falla; la revocación ya está hecha
		_ = auditErr
	}

	// Invalidar cache
	m.mu.Lock()
	delete(m.cache, agentID)
	m.mu.Unlock()

	// Invalidar cache del registro
	_ = record

	return nil
}

// ListRevoked devuelve todas las identidades revocadas.
func (m *Manager) ListRevoked() []*RevocationRecord {
	return m.revocations.List()
}

// CountRevoked devuelve cuántas identidades están revocadas.
func (m *Manager) CountRevoked() int {
	return m.revocations.Count()
}

// CACertPEM devuelve el certificado de la CA interna.
func (m *Manager) CACertPEM() []byte {
	return m.ca.CACertPEM()
}

// Store devuelve el Store subyacente.
func (m *Manager) Store() *Store {
	return m.store
}

// Revocations devuelve el registro de revocaciones.
func (m *Manager) Revocations() *RevocationRegistry {
	return m.revocations
}

// SystemKeyID devuelve el ID de la clave de firma del sistema.
func (m *Manager) SystemKeyID() string {
	return m.systemKeyID
}
