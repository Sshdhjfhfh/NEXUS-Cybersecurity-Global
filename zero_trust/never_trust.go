// Nexus IA v6.0 — Principio 2: Never Trust, Always Verify
//
// Cada llamada entre módulos se autentica. No hay confianza implícita
// basada en red, ubicación o rol. CADA petición debe presentar un JWT
// válido firmado por nhi_manager.
//
// Este principio se implementa verificando el JWT en cada acción.
package zero_trust

import (
	"time"
)

// JWTVerifier es una función que verifica un JWT.
// Se inyecta desde nhi_manager.
type JWTVerifier func(token string) (*VerifiedClaims, error)

// VerifiedClaims son los claims extraídos de un JWT válido.
type VerifiedClaims struct {
	AgentID        string
	AgentType      string
	DID            string
	AllowedActions []string
	AllowedAssets  []string
	SessionID      string
	ExpiresAt      time.Time
}

// NeverTrustRule implementa el principio 2.
type NeverTrustRule struct {
	// VerifyJWT verifica un JWT. Si es nil, la regla siempre deniega.
	VerifyJWT JWTVerifier

	// RequireDID requiere que el JWT tenga un DID válido.
	RequireDID bool

	// CheckRevocation verifica si el agente fue revocado.
	// Se inyecta desde nhi_manager.
	CheckRevocation func(agentID string) bool
}

// NewNeverTrustRule crea la regla con un verificador de JWT.
func NewNeverTrustRule(verify JWTVerifier) *NeverTrustRule {
	return &NeverTrustRule{
		VerifyJWT:  verify,
		RequireDID: true,
	}
}

// Name devuelve el nombre de la regla.
func (r *NeverTrustRule) Name() string {
	return "never-trust-always-verify"
}

// Principle devuelve el principio MAESTRO.
func (r *NeverTrustRule) Principle() Principle {
	return PrincipleNeverTrust
}

// Evaluate verifica el JWT de la petición.
func (r *NeverTrustRule) Evaluate(req *Request) Result {
	if req == nil {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "nil request",
		}
	}

	// Sin verificador → denegar
	if r.VerifyJWT == nil {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "JWT verifier not configured",
			Timestamp: req.Action.Timestamp,
		}
	}

	// Sin JWT → denegar
	if req.JWT == "" {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "no JWT provided",
			Timestamp: req.Action.Timestamp,
		}
	}

	// Verificar JWT
	claims, err := r.VerifyJWT(req.JWT)
	if err != nil {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "JWT verification failed: " + err.Error(),
			Timestamp: req.Action.Timestamp,
		}
	}

	// Verificar DID si es requerido
	if r.RequireDID && claims.DID == "" {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "JWT missing DID",
			Timestamp: req.Action.Timestamp,
		}
	}

	// Verificar revocación
	if r.CheckRevocation != nil && r.CheckRevocation(claims.AgentID) {
		return Result{
			Decision:  Quarantine,
			Principle: r.Principle(),
			Reason:    "agent revoked: " + claims.AgentID,
			Timestamp: req.Action.Timestamp,
		}
	}

	// Verificar que el actor del request coincida con el JWT
	if req.Action.ActorID != claims.AgentID {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "actor mismatch: request=" + req.Action.ActorID + " jwt=" + claims.AgentID,
			Timestamp: req.Action.Timestamp,
		}
	}

	return Result{
		Decision:  Allow,
		Principle: r.Principle(),
		Reason:    "JWT valid for agent " + claims.AgentID,
		TTL:       time.Until(claims.ExpiresAt),
		Timestamp: req.Action.Timestamp,
	}
}
