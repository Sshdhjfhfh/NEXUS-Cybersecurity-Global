// Nexus IA v6.0 — Principio 3: Micro-segmentación
//
// Cada agente corre en su propia red Docker aislada. No puede ver ni
// contactar a otros agentes excepto por canales explícitamente autorizados.
//
// Este principio se implementa como una matriz de conectividad:
//   - Cada AgentType tiene una "zona" (subred Docker)
//   - Cada zona tiene reglas de acceso a otras zonas
//   - Todo intento de contacto entre zonas se audita
//
// Zonas de Nexus v6:
//   - red_zone:    agentes RED (ofensivos)
//   - blue_zone:   agentes BLUE (defensivos)
//   - swarm_zone:  coordinador + agentes del swarm
//   - intel_zone:  prediction, cti_feed, knowledge_bus
//   - core_zone:   orquestador, db, llm_gateway, ethical_boundary
//   - lab_zone:    DVWA, Metasploitable, red virtual
//   - governance:  nhi_manager, abm, ans, rsi_detector, governance
package zero_trust

// Zone identifica una zona de micro-segmentación.
type Zone string

const (
	ZoneRed        Zone = "red_zone"
	ZoneBlue       Zone = "blue_zone"
	ZoneSwarm      Zone = "swarm_zone"
	ZoneIntel      Zone = "intel_zone"
	ZoneCore       Zone = "core_zone"
	ZoneLab        Zone = "lab_zone"
	ZoneGovernance Zone = "governance_zone"
	ZoneUnknown    Zone = "unknown"
)

// ZoneForAgentType mapea un AgentType a su zona.
func ZoneForAgentType(agentType string) Zone {
	switch agentType {
	case "RED", "EXPLOIT_NET", "EXPLOIT_WEB", "POST_EXPLOIT", "STEALTH":
		return ZoneRed
	case "BLUE":
		return ZoneBlue
	case "SWARM_COORDINATOR", "SCOUT", "ENUM":
		return ZoneSwarm
	case "PREDICTION":
		return ZoneIntel
	default:
		return ZoneUnknown
	}
}

// ConnectivityMap define qué zonas pueden contactar a qué zonas.
//
//	ConnectivityMap[from][to] = true  → permitido
//	ConnectivityMap[from][to] = false → denegado (por defecto)
var ConnectivityMap = map[Zone]map[Zone]bool{
	ZoneRed: {
		ZoneCore:       true,  // puede hablar con orchestrator
		ZoneLab:        true,  // puede atacar el lab
		ZoneGovernance: true,  // puede hablar con abm (monitoring)
		ZoneSwarm:      true,  // puede recibir órdenes del swarm
		ZoneBlue:       false, // NO puede tocar BLUE
		ZoneIntel:      false, // NO puede tocar prediction/cti
	},
	ZoneBlue: {
		ZoneCore:       true,
		ZoneGovernance: true,
		ZoneSwarm:      true,
		ZoneRed:        false,
		ZoneIntel:      true, // BLUE sí puede consultar intel
		ZoneLab:        false,
	},
	ZoneSwarm: {
		ZoneRed:        true,
		ZoneBlue:       true,
		ZoneIntel:      true,
		ZoneCore:       true,
		ZoneGovernance: true,
		ZoneLab:        false,
	},
	ZoneIntel: {
		ZoneCore:       true,
		ZoneGovernance: true,
		ZoneRed:        false,
		ZoneBlue:       false,
		ZoneLab:        false,
	},
	ZoneCore: {
		ZoneRed:        true,
		ZoneBlue:       true,
		ZoneSwarm:      true,
		ZoneIntel:      true,
		ZoneGovernance: true,
		ZoneLab:        true,
	},
	ZoneLab: {
		ZoneCore:  true, // solo el core puede gestionar el lab
		ZoneRed:   false,
		ZoneBlue:  false,
		ZoneSwarm: false,
	},
	ZoneGovernance: {
		// Governance puede observar TODO pero solo iniciar conexiones al core
		ZoneCore:  true,
		ZoneRed:   false,
		ZoneBlue:  false,
		ZoneSwarm: false,
		ZoneIntel: false,
		ZoneLab:   false,
	},
	ZoneUnknown: {
		// Unknown no puede nada
	},
}

// MicroSegmentationRule implementa el principio 3.
type MicroSegmentationRule struct {
	// Connectivity permite override del mapa por defecto.
	Connectivity map[Zone]map[Zone]bool

	// SourceZoneResolver resuelve la zona de un AgentID.
	// Se inyecta desde nhi_manager/ans.
	SourceZoneResolver func(agentID string) Zone
}

// NewMicroSegmentationRule crea la regla con el mapa por defecto.
func NewMicroSegmentationRule() *MicroSegmentationRule {
	return &MicroSegmentationRule{
		Connectivity: ConnectivityMap,
	}
}

// Name devuelve el nombre de la regla.
func (r *MicroSegmentationRule) Name() string {
	return "micro-segmentation"
}

// Principle devuelve el principio MAESTRO.
func (r *MicroSegmentationRule) Principle() Principle {
	return PrincipleMicroSegmentation
}

// Evaluate comprueba que el agente pueda contactar el target.
//
// El target se interpreta como "zona:recurso", ej:
//   - "core_zone:orchestrator"
//   - "lab_zone:dvwa"
//   - "governance_zone:abm"
func (r *MicroSegmentationRule) Evaluate(req *Request) Result {
	if req == nil {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "nil request",
		}
	}

	// Resolver zona del actor
	sourceZone := ZoneForAgentType(req.Action.AgentType)
	if r.SourceZoneResolver != nil {
		sourceZone = r.SourceZoneResolver(req.Action.ActorID)
	}

	// Resolver zona del target
	targetZone := zoneFromTarget(req.Action.Target)
	if targetZone == "" {
		// Target no es una zona conocida → permitir (no aplica)
		return Result{
			Decision:  Allow,
			Principle: r.Principle(),
			Reason:    "target not zone-segmented (not applicable)",
			Timestamp: req.Action.Timestamp,
		}
	}

	// Sin mapa de conectividad → denegar por defecto
	if r.Connectivity == nil {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "connectivity map not configured",
			Timestamp: req.Action.Timestamp,
		}
	}

	// Comprobar conectividad
	fromMap, ok := r.Connectivity[sourceZone]
	if !ok {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "unknown source zone: " + string(sourceZone),
			Timestamp: req.Action.Timestamp,
		}
	}

	allowed, ok := fromMap[targetZone]
	if !ok || !allowed {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "zone " + string(sourceZone) + " cannot reach zone " + string(targetZone),
			Timestamp: req.Action.Timestamp,
		}
	}

	return Result{
		Decision:  Allow,
		Principle: r.Principle(),
		Reason:    "zone " + string(sourceZone) + " → zone " + string(targetZone) + " permitted",
		Timestamp: req.Action.Timestamp,
	}
}

// zoneFromTarget extrae la zona de un target tipo "zone:recurso".
func zoneFromTarget(target string) Zone {
	for _, z := range []Zone{
		ZoneRed, ZoneBlue, ZoneSwarm, ZoneIntel,
		ZoneCore, ZoneLab, ZoneGovernance,
	} {
		prefix := string(z) + ":"
		if len(target) >= len(prefix) && target[:len(prefix)] == prefix {
			return z
		}
	}
	return ""
}
