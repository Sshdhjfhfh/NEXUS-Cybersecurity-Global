// Nexus IA v6.0 — Principio 4: Assume Breach
//
// El sistema asume que cualquier agente puede comprometerse. No hay
// "zona segura". Cada acción se diseña asumiendo que el actor puede
// estar comprometido.
//
// Este principio se implementa como:
//   - Trust score dinámico por agente (baja con comportamiento anómalo)
//   - Circuit breaker por agente (si trust < threshold → cuarentena)
//   - Blast radius limiting (una brecha no puede tomar todo el sistema)
//   - Detección de señales de compromiso (comportamiento divergente)
package zero_trust

import (
	"sync"
	"time"
)

// TrustScore es el nivel de confianza de un agente (0-100).
// 100 = confianza total (recién verificado, comportamiento normal).
// 0   = cuarentena (brecha detectada).
type TrustScore int

const (
	TrustScoreMax        TrustScore = 100
	TrustScoreMin        TrustScore = 0
	TrustScoreQuarantine TrustScore = 20
	TrustScoreWarn       TrustScore = 50
)

// TrustRecord es el registro de confianza de un agente.
type TrustRecord struct {
	AgentID    string
	Score      TrustScore
	LastUpdate time.Time
	Signals    []TrustSignal // Señales que afectaron el score
	Reason     string
}

// TrustSignal es una señal que afecta el trust score.
type TrustSignal struct {
	Type      string    // "anomaly", "denied_action", "rate_limit", "rogue_behavior"
	Delta     int       // Cambio en el score (-100 a +10)
	Reason    string
	Timestamp time.Time
}

// TrustRegistry mantiene el trust score de todos los agentes.
// Es thread-safe.
type TrustRegistry struct {
	mu      sync.RWMutex
	records map[string]*TrustRecord

	// Parámetros del sistema
	quarantineThreshold TrustScore
	warnThreshold       TrustScore
}

// NewTrustRegistry crea un registro de confianza vacío.
func NewTrustRegistry() *TrustRegistry {
	return &TrustRegistry{
		records:             make(map[string]*TrustRecord),
		quarantineThreshold: TrustScoreQuarantine,
		warnThreshold:       TrustScoreWarn,
	}
}

// Register inicializa el score de un agente en el máximo.
func (tr *TrustRegistry) Register(agentID string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.records[agentID] = &TrustRecord{
		AgentID:    agentID,
		Score:      TrustScoreMax,
		LastUpdate: time.Now(),
		Signals:    nil,
		Reason:     "initial registration",
	}
}

// AdjustScore ajusta el score de un agente con una señal.
func (tr *TrustRegistry) AdjustScore(agentID string, delta int, signalType, reason string) TrustScore {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	rec, ok := tr.records[agentID]
	if !ok {
		rec = &TrustRecord{
			AgentID:    agentID,
			Score:      TrustScoreMax,
			LastUpdate: time.Now(),
			Reason:     "auto-created",
		}
		tr.records[agentID] = rec
	}

	newScore := int(rec.Score) + delta
	if newScore > int(TrustScoreMax) {
		newScore = int(TrustScoreMax)
	}
	if newScore < int(TrustScoreMin) {
		newScore = int(TrustScoreMin)
	}

	rec.Score = TrustScore(newScore)
	rec.LastUpdate = time.Now()
	rec.Reason = reason

	// Añadir señal (máximo 100 por agente)
	if len(rec.Signals) >= 100 {
		rec.Signals = rec.Signals[1:]
	}
	rec.Signals = append(rec.Signals, TrustSignal{
		Type:      signalType,
		Delta:     delta,
		Reason:    reason,
		Timestamp: time.Now(),
	})

	return rec.Score
}

// GetScore devuelve el score actual de un agente.
// Si no existe, devuelve 100 (confianza por defecto hasta que haya señal).
func (tr *TrustRegistry) GetScore(agentID string) TrustScore {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	rec, ok := tr.records[agentID]
	if !ok {
		return TrustScoreMax
	}
	return rec.Score
}

// GetRecord devuelve el registro completo de un agente.
func (tr *TrustRegistry) GetRecord(agentID string) *TrustRecord {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	rec, ok := tr.records[agentID]
	if !ok {
		return nil
	}
	// Copia defensiva
	cpy := *rec
	cpy.Signals = append([]TrustSignal(nil), rec.Signals...)
	return &cpy
}

// IsQuarantined devuelve true si el agente está en cuarentena.
func (tr *TrustRegistry) IsQuarantined(agentID string) bool {
	return tr.GetScore(agentID) <= tr.quarantineThreshold
}

// Reset devuelve el score de un agente al máximo (tras investigación).
func (tr *TrustRegistry) Reset(agentID string, reason string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	rec, ok := tr.records[agentID]
	if !ok {
		rec = &TrustRecord{
			AgentID:    agentID,
			LastUpdate: time.Now(),
		}
		tr.records[agentID] = rec
	}
	rec.Score = TrustScoreMax
	rec.Reason = "reset: " + reason
	rec.LastUpdate = time.Now()
}

// AssumeBreachRule implementa el principio 4.
type AssumeBreachRule struct {
	// Registry es el registro de trust scores.
	Registry *TrustRegistry

	// QuarantineThreshold override (0 = usar default 20).
	QuarantineThreshold TrustScore

	// MaxBlastRadius es el número máximo de agentes afectables por una brecha.
	// Si > 0, cuando N agentes están en cuarentena, se cuarentena TODOS.
	MaxBlastRadius int
}

// NewAssumeBreachRule crea la regla con un registro fresco.
func NewAssumeBreachRule() *AssumeBreachRule {
	return &AssumeBreachRule{
		Registry:            NewTrustRegistry(),
		QuarantineThreshold: TrustScoreQuarantine,
		MaxBlastRadius:      0,
	}
}

// Name devuelve el nombre de la regla.
func (r *AssumeBreachRule) Name() string {
	return "assume-breach"
}

// Principle devuelve el principio MAESTRO.
func (r *AssumeBreachRule) Principle() Principle {
	return PrincipleAssumeBreach
}

// Evaluate comprueba el trust score del agente.
func (r *AssumeBreachRule) Evaluate(req *Request) Result {
	if req == nil {
		return Result{
			Decision:  Deny,
			Principle: r.Principle(),
			Reason:    "nil request",
		}
	}

	if r.Registry == nil {
		// Sin registry → no aplica
		return Result{
			Decision:  Allow,
			Principle: r.Principle(),
			Reason:    "trust registry not configured (not applicable)",
			Timestamp: req.Action.Timestamp,
		}
	}

	score := r.Registry.GetScore(req.Action.ActorID)
	threshold := r.QuarantineThreshold
	if threshold == 0 {
		threshold = TrustScoreQuarantine
	}

	if score <= threshold {
		return Result{
			Decision:  Quarantine,
			Principle: r.Principle(),
			Reason:    "trust score " + scoreString(score) + " below quarantine threshold",
			Timestamp: req.Action.Timestamp,
		}
	}

	if score <= TrustScoreWarn {
		return Result{
			Decision:  Ask,
			Principle: r.Principle(),
			Reason:    "trust score " + scoreString(score) + " below warn threshold (human approval required)",
			RequiresHumanApproval: true,
			Timestamp: req.Action.Timestamp,
		}
	}

	return Result{
		Decision:  Allow,
		Principle: r.Principle(),
		Reason:    "trust score " + scoreString(score) + " OK",
		Timestamp: req.Action.Timestamp,
	}
}

// scoreString convierte un TrustScore a string.
func scoreString(s TrustScore) string {
	switch {
	case s >= 90:
		return "90+ (trusted)"
	case s >= 70:
		return "70-89 (normal)"
	case s >= 50:
		return "50-69 (watch)"
	case s >= 20:
		return "20-49 (warn)"
	default:
		return "<20 (quarantine)"
	}
}
