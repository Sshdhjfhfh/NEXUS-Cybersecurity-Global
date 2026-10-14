// Nexus IA v6.0 — Principio 1: Mínimo Privilegio
//
// Cada agente solo hace lo que su rol define.
// Un agente RED no puede ejecutar acciones de BLUE, y viceversa.
// Un agente STEALTH no puede generar exploits.
// etc.
//
// Este principio se implementa como una tabla de permisos por AgentType.
package zero_trust

import (
	"time"
	"strings"
)

// PermissionMap define qué acciones puede ejecutar cada AgentType.
// Es la fuente de verdad del principio de mínimo privilegio.
var PermissionMap = map[string][]string{
	"RED": {
		"SCAN", "EXPLOIT", "DUMP_CREDENTIALS", "LATERAL_MOVE",
		"GENERATE_PAYLOAD", "QUERY_KNOWLEDGE",
	},
	"BLUE": {
		"SCAN", "ISOLATE_HOST", "DEPLOY_DECOY", "QUERY_KNOWLEDGE",
		"ROTATE_VPN",
	},
	"PREDICTION": {
		"QUERY_KNOWLEDGE",
	},
	"SWARM_COORDINATOR": {
		"SCAN", "EXPLOIT", "DUMP_CREDENTIALS", "LATERAL_MOVE",
		"GENERATE_PAYLOAD", "EVOLVE_MODULE", "QUERY_KNOWLEDGE",
		"ISOLATE_HOST", "DEPLOY_DECOY",
	},
	"SCOUT": {
		"SCAN", "QUERY_KNOWLEDGE",
	},
	"ENUM": {
		"SCAN", "QUERY_KNOWLEDGE",
	},
	"EXPLOIT_NET": {
		"SCAN", "EXPLOIT", "DUMP_CREDENTIALS", "LATERAL_MOVE",
		"QUERY_KNOWLEDGE",
	},
	"EXPLOIT_WEB": {
		"SCAN", "EXPLOIT", "GENERATE_PAYLOAD", "QUERY_KNOWLEDGE",
	},
	"POST_EXPLOIT": {
		"SCAN", "DUMP_CREDENTIALS", "LATERAL_MOVE", "QUERY_KNOWLEDGE",
	},
	"STEALTH": {
		"ROTATE_VPN", "QUERY_KNOWLEDGE",
	},
}

// MinimalPrivilegeRule implementa el principio 1.
type MinimalPrivilegeRule struct {
	// Permissions permite override de la tabla por defecto.
	Permissions map[string][]string
}

// NewMinimalPrivilegeRule crea la regla con la tabla por defecto.
func NewMinimalPrivilegeRule() *MinimalPrivilegeRule {
	return &MinimalPrivilegeRule{
		Permissions: PermissionMap,
	}
}

// Name devuelve el nombre de la regla.
func (r *MinimalPrivilegeRule) Name() string {
	return "minimal-privilege"
}

// Principle devuelve el principio MAESTRO que implementa.
func (r *MinimalPrivilegeRule) Principle() Principle {
	return PrincipleMinimalPrivilege
}

// Evaluate comprueba que la acción esté permitida para el AgentType.
func (r *MinimalPrivilegeRule) Evaluate(req *Request) Result {
	if req == nil || req.Action.ActorID == "" {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "invalid request: missing actor",
			Timestamp: nowOrZero(req),
		}
	}

	agentType := strings.ToUpper(req.Action.AgentType)
	actionType := strings.ToUpper(req.Action.Type)

	allowed, ok := r.Permissions[agentType]
	if !ok {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "unknown agent type: " + agentType,
			Timestamp: nowOrZero(req),
		}
	}

	for _, a := range allowed {
		if a == actionType {
			return Result{
				Decision:  Allow,
				Principle: r.Principle(),
				Reason:    "action permitted for role " + agentType,
				Timestamp: nowOrZero(req),
			}
		}
	}

	return Result{
		Decision:  Deny,
		Principle: r.Principle(),
		Reason:    "action " + actionType + " not permitted for role " + agentType,
		Timestamp: nowOrZero(req),
	}
}

// nowOrZero devuelve el timestamp del request o time.Time{} si es nil.
func nowOrZero(req *Request) (t time.Time) {
	if req != nil && !req.Action.Timestamp.IsZero() {
		return req.Action.Timestamp
	}
	return
}
