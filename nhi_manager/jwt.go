// Nexus IA v6.0 — JWT con claims de permisos exactos
//
// Cada agente recibe un JWT firmado por Nexus que encapsula:
//   - AgentID + AgentType
//   - DID único
//   - Permisos por acción (AllowedActions)
//   - Assets autorizados (AllowedAssets)
//   - Timestamps de emisión/expiración
//
// Los JWT son de vida corta (1h por defecto) y se verifican en cada
// acción del agente. Si el agente es revocado, su JWT deja de valer
// aunque no haya expirado.
package nhi_manager

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DefaultJWTTTL es la vida por defecto de un JWT.
const DefaultJWTTTL = 1 * time.Hour

// JWTSigningKey es la clave HMAC para firmar los JWT.
// En producción debe cargarse de un secret seguro.
type JWTSigningKey []byte

// ClaimsNexus son los claims personalizados del JWT de Nexus.
type ClaimsNexus struct {
	jwt.RegisteredClaims

	// Identidad
	AgentID   string    `json:"agent_id"`
	AgentType AgentType `json:"agent_type"`
	DID       string    `json:"did"`

	// Permisos
	AllowedActions []string `json:"allowed_actions"`
	AllowedAssets  []string `json:"allowed_assets"`

	// Ciclo de vida
	SessionID string `json:"session_id,omitempty"`
}

// IssueJWT emite un JWT para un agente.
// Incluye todos los permisos del NHI y firma HMAC-SHA256.
func IssueJWT(key JWTSigningKey, nhi *NHI, sessionID string, ttl time.Duration) (string, error) {
	if key == nil || len(key) < 32 {
		return "", errors.New("jwt signing key too short (min 32 bytes)")
	}

	if nhi == nil {
		return "", errors.New("nhi cannot be nil")
	}

	if ttl == 0 {
		ttl = DefaultJWTTTL
	}

	now := time.Now()

	// Convertir permisos a strings para el JWT
	actions := make([]string, len(nhi.AllowedActions))
	for i, a := range nhi.AllowedActions {
		actions[i] = string(a)
	}

	claims := ClaimsNexus{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "nexus-ia",
			Subject:   nhi.AgentID,
			Audience:  jwt.ClaimStrings{"nexus-ia"},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Minute)),
			ID:        nhi.DID.String(),
		},
		AgentID:        nhi.AgentID,
		AgentType:      nhi.AgentType,
		DID:            nhi.DID.String(),
		AllowedActions: actions,
		AllowedAssets:  nhi.AllowedAssets,
		SessionID:      sessionID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(key))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return signed, nil
}

// VerifyJWT verifica y parsea un JWT.
// Devuelve los claims si la firma es válida y el token no ha expirado.
func VerifyJWT(key JWTSigningKey, tokenString string) (*ClaimsNexus, error) {
	if key == nil || len(key) < 32 {
		return nil, errors.New("jwt signing key too short (min 32 bytes)")
	}

	token, err := jwt.ParseWithClaims(tokenString, &ClaimsNexus{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(key), nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := token.Claims.(*ClaimsNexus)
	if !ok {
		return nil, errors.New("invalid claims type")
	}

	if !token.Valid {
		return nil, errors.New("jwt not valid")
	}

	return claims, nil
}

// HasPermission verifica si los claims permiten la acción.
func (c *ClaimsNexus) HasPermission(action ActionType) bool {
	for _, a := range c.AllowedActions {
		if a == string(action) {
			return true
		}
	}
	return false
}

// TTL devuelve el tiempo restante de vida del JWT.
func (c *ClaimsNexus) TTL() time.Duration {
	if c.ExpiresAt == nil {
		return 0
	}
	return time.Until(c.ExpiresAt.Time)
}

// GenerateSigningKey genera una clave de firma aleatoria de 32 bytes.
func GenerateSigningKey() (JWTSigningKey, error) {
	key := make([]byte, 32)
	if _, err := readRandom(key); err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}
	return JWTSigningKey(key), nil
}
