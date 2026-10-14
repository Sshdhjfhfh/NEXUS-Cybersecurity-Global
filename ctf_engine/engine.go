// Nexus IA v6.0 — CTF Engine (implementación principal)
//
// Mantiene el catálogo de retos, gestiona intentos, calcula métricas
// y rankings. Thread-safe.
package ctf_engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultEngine es la implementación principal del Engine.
type DefaultEngine struct {
	cfg Config

	mu sync.RWMutex

	// Catálogo de retos por ID
	challenges map[string]*Challenge

	// Intentos por ID
	attempts map[string]*Attempt

	// Índice: challengeID → []attemptIDs
	attemptsByChallenge map[string][]string

	// Índice: agentID → []attemptIDs
	attemptsByAgent map[string][]string

	// Métricas calculadas por challengeID
	metrics map[string]*SolveMetrics

	// Ranking por agentID
	rankings map[string]*AgentRanking

	// Lifecycle
	started  bool
	shutdown bool
}

// NewEngine construye el engine.
func NewEngine(cfg Config) *DefaultEngine {
	return &DefaultEngine{
		cfg:                 cfg,
		challenges:          make(map[string]*Challenge),
		attempts:            make(map[string]*Attempt),
		attemptsByChallenge: make(map[string][]string),
		attemptsByAgent:     make(map[string][]string),
		metrics:             make(map[string]*SolveMetrics),
		rankings:            make(map[string]*AgentRanking),
		started:             true,
	}
}

// LoadChallenge carga un reto al catálogo.
func (e *DefaultEngine) LoadChallenge(ch *Challenge) error {
	if ch == nil {
		return fmt.Errorf("nil challenge")
	}
	if ch.ID == "" {
		return fmt.Errorf("empty challenge ID")
	}
	if ch.Name == "" {
		return fmt.Errorf("empty challenge name")
	}

	// Verificar que la plataforma está activa
	enabled := false
	for _, p := range e.cfg.EnabledPlatforms {
		if p == ch.Platform {
			enabled = true
			break
		}
	}
	if !enabled {
		return fmt.Errorf("platform %s not enabled", ch.Platform)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.shutdown {
		return fmt.Errorf("engine shut down")
	}

	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = time.Now()
	}
	if ch.Metadata == nil {
		ch.Metadata = make(map[string]string)
	}

	// Copia defensiva
	cpy := *ch
	e.challenges[ch.ID] = &cpy

	return nil
}

// ListChallenges devuelve todos los retos.
func (e *DefaultEngine) ListChallenges() []*Challenge {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*Challenge, 0, len(e.challenges))
	for _, ch := range e.challenges {
		cpy := *ch
		out = append(out, &cpy)
	}
	return out
}

// ListUnsolvedChallenges devuelve los retos no resueltos.
func (e *DefaultEngine) ListUnsolvedChallenges() []*Challenge {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var out []*Challenge
	for _, ch := range e.challenges {
		if !ch.Solved {
			cpy := *ch
			out = append(out, &cpy)
		}
	}
	return out
}

// GetChallenge recupera un reto por ID.
func (e *DefaultEngine) GetChallenge(id string) (*Challenge, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ch, ok := e.challenges[id]
	if !ok {
		return nil, fmt.Errorf("challenge not found: %s", id)
	}
	cpy := *ch
	return &cpy, nil
}

// StartAttempt inicia un intento de resolución.
func (e *DefaultEngine) StartAttempt(challengeID, agentID string) (*Attempt, error) {
	if challengeID == "" || agentID == "" {
		return nil, fmt.Errorf("empty challenge ID or agent ID")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.shutdown {
		return nil, fmt.Errorf("engine shut down")
	}

	// Verificar que el reto existe
	ch, ok := e.challenges[challengeID]
	if !ok {
		return nil, fmt.Errorf("challenge not found: %s", challengeID)
	}
	if ch.Solved {
		return nil, fmt.Errorf("challenge already solved")
	}

	// Verificar paralelismo
	activeCount := 0
	for _, a := range e.attempts {
		if a.FinishedAt == nil && a.ChallengeID == challengeID {
			activeCount++
		}
	}
	if activeCount >= e.cfg.MaxParallelAttempts {
		return nil, fmt.Errorf("max parallel attempts (%d) reached for challenge", e.cfg.MaxParallelAttempts)
	}

	// Crear intento
	attemptID := fmt.Sprintf("attempt-%s-%d", challengeID, time.Now().UnixNano())
	attempt := &Attempt{
		ID:          attemptID,
		ChallengeID: challengeID,
		AgentID:     agentID,
		StartedAt:   time.Now(),
	}

	e.attempts[attemptID] = attempt
	e.attemptsByChallenge[challengeID] = append(e.attemptsByChallenge[challengeID], attemptID)
	e.attemptsByAgent[agentID] = append(e.attemptsByAgent[agentID], attemptID)

	return attempt, nil
}

// FinishAttempt cierra un intento.
func (e *DefaultEngine) FinishAttempt(attemptID string, solved bool, flag string, techniques []string, hintsUsed int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	attempt, ok := e.attempts[attemptID]
	if !ok {
		return fmt.Errorf("attempt not found: %s", attemptID)
	}
	if attempt.FinishedAt != nil {
		return fmt.Errorf("attempt already finished")
	}

	now := time.Now()
	attempt.FinishedAt = &now
	attempt.Solved = solved
	attempt.Flag = flag
	attempt.TimeSpent = now.Sub(attempt.StartedAt)
	attempt.Techniques = techniques
	attempt.HintsUsed = hintsUsed

	// Si resolvió, marcar el reto como resuelto
	if solved {
		ch, ok := e.challenges[attempt.ChallengeID]
		if ok && !ch.Solved {
			ch.Solved = true
			ch.SolvedBy = attempt.AgentID
			ch.SolvedAt = &now
			ch.Flag = flag
		}
	}

	// Actualizar métricas del reto
	e.updateMetricsLocked(attempt.ChallengeID)

	// Actualizar ranking del agente
	e.updateRankingLocked(attempt.AgentID)

	return nil
}

// ListAttempts devuelve los intentos de un reto.
func (e *DefaultEngine) ListAttempts(challengeID string) []*Attempt {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := e.attemptsByChallenge[challengeID]
	out := make([]*Attempt, 0, len(ids))
	for _, id := range ids {
		if a, ok := e.attempts[id]; ok {
			cpy := *a
			out = append(out, &cpy)
		}
	}
	return out
}

// ListAttemptsByAgent devuelve los intentos de un agente.
func (e *DefaultEngine) ListAttemptsByAgent(agentID string) []*Attempt {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := e.attemptsByAgent[agentID]
	out := make([]*Attempt, 0, len(ids))
	for _, id := range ids {
		if a, ok := e.attempts[id]; ok {
			cpy := *a
			out = append(out, &cpy)
		}
	}
	return out
}

// Metrics devuelve las métricas de un reto.
func (e *DefaultEngine) Metrics(challengeID string) *SolveMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	m, ok := e.metrics[challengeID]
	if !ok {
		return nil
	}
	cpy := *m
	return &cpy
}

// updateMetricsLocked recalcula las métricas de un reto.
func (e *DefaultEngine) updateMetricsLocked(challengeID string) {
	ch, ok := e.challenges[challengeID]
	if !ok || !ch.Solved {
		return
	}

	// Buscar el intento ganador
	var solveAttempt *Attempt
	for _, aid := range e.attemptsByChallenge[challengeID] {
		a := e.attempts[aid]
		if a != nil && a.Solved {
			solveAttempt = a
			break
		}
	}
	if solveAttempt == nil {
		return
	}

	metrics := &SolveMetrics{
		ChallengeID: challengeID,
		Platform:    ch.Platform,
		Category:    ch.Category,
		Difficulty:  ch.Difficulty,
		TimeSpent:   solveAttempt.TimeSpent,
		HintsUsed:   solveAttempt.HintsUsed,
		Techniques:  solveAttempt.Techniques,
	}

	// Estimar comparación con humanos (valores aproximados)
	metrics.HumanAvgTime = humanAvgTime(ch.Difficulty)
	metrics.HumanAvgHints = humanAvgHints(ch.Difficulty)
	metrics.Percentile = calculatePercentile(metrics.TimeSpent, metrics.HumanAvgTime)

	e.metrics[challengeID] = metrics
}

// updateRankingLocked recalcula el ranking de un agente.
func (e *DefaultEngine) updateRankingLocked(agentID string) {
	ranking, ok := e.rankings[agentID]
	if !ok {
		ranking = &AgentRanking{
			AgentID:    agentID,
			ByCategory: make(map[Category]int),
		}
		e.rankings[agentID] = ranking
	}

	// Recalcular desde cero
	ranking.TotalSolves = 0
	ranking.TotalAttempts = 0
	ranking.EasySolves = 0
	ranking.MediumSolves = 0
	ranking.HardSolves = 0
	ranking.InsaneSolves = 0
	ranking.ByCategory = make(map[Category]int)
	ranking.WeeklyProgress = nil

	var totalTime time.Duration
	var totalHints int
	var totalPercentile float64
	percentileCount := 0

	ids := e.attemptsByAgent[agentID]
	for _, aid := range ids {
		a := e.attempts[aid]
		if a == nil || a.FinishedAt == nil {
			continue
		}
		ranking.TotalAttempts++

		if !a.Solved {
			continue
		}

		ranking.TotalSolves++
		totalTime += a.TimeSpent
		totalHints += a.HintsUsed

		// Contar por dificultad
		ch, ok := e.challenges[a.ChallengeID]
		if ok {
			switch ch.Difficulty {
			case DifficultyEasy:
				ranking.EasySolves++
			case DifficultyMedium:
				ranking.MediumSolves++
			case DifficultyHard:
				ranking.HardSolves++
			case DifficultyInsane:
				ranking.InsaneSolves++
			}
			ranking.ByCategory[ch.Category]++
		}

		// Percentile
		if m, ok := e.metrics[a.ChallengeID]; ok {
			totalPercentile += m.Percentile
			percentileCount++
		}

		// Timestamps
		if ranking.FirstSolveAt == nil || a.FinishedAt.Before(*ranking.FirstSolveAt) {
			t := *a.FinishedAt
			ranking.FirstSolveAt = &t
		}
		if ranking.LastSolveAt == nil || a.FinishedAt.After(*ranking.LastSolveAt) {
			t := *a.FinishedAt
			ranking.LastSolveAt = &t
		}
	}

	// Promedios
	if ranking.TotalSolves > 0 {
		ranking.AvgTimeSpent = totalTime / time.Duration(ranking.TotalSolves)
		ranking.AvgHintsUsed = float64(totalHints) / float64(ranking.TotalSolves)
	}
	if ranking.TotalAttempts > 0 {
		ranking.SolveRate = float64(ranking.TotalSolves) / float64(ranking.TotalAttempts)
	}
	if percentileCount > 0 {
		ranking.AvgPercentile = totalPercentile / float64(percentileCount)
	}
}

// Ranking devuelve el ranking de un agente.
func (e *DefaultEngine) Ranking(agentID string) *AgentRanking {
	e.mu.RLock()
	defer e.mu.RUnlock()

	r, ok := e.rankings[agentID]
	if !ok {
		return nil
	}
	cpy := *r
	cpy.ByCategory = make(map[Category]int, len(r.ByCategory))
	for k, v := range r.ByCategory {
		cpy.ByCategory[k] = v
	}
	return &cpy
}

// ListRankings devuelve todos los rankings.
func (e *DefaultEngine) ListRankings() []*AgentRanking {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]*AgentRanking, 0, len(e.rankings))
	for _, r := range e.rankings {
		cpy := *r
		out = append(out, &cpy)
	}
	return out
}

// Stats devuelve estadísticas globales.
func (e *DefaultEngine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := EngineStats{
		TotalChallenges: len(e.challenges),
		TotalAttempts:   len(e.attempts),
		TotalAgents:     len(e.attemptsByAgent),
		ByPlatform:      make(map[Platform]int),
		ByCategory:      make(map[Category]int),
		ByDifficulty:    make(map[Difficulty]int),
	}

	for _, ch := range e.challenges {
		if ch.Solved {
			stats.TotalSolved++
		}
		stats.ByPlatform[ch.Platform]++
		stats.ByCategory[ch.Category]++
		stats.ByDifficulty[ch.Difficulty]++
	}

	return stats
}

// Shutdown detiene el engine.
func (e *DefaultEngine) Shutdown() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.shutdown = true
	e.started = false
	return nil
}

// Start inicia el motor (placeholder para integración futura).
func (e *DefaultEngine) Start(ctx context.Context) {
	// Aquí iría el bucle que ejecuta intentos en paralelo
	// con los agentes RED del swarm.
	<-ctx.Done()
}

// ═══════════════════════════════════════════════════════════════
// Helpers de comparación con humanos
// ═══════════════════════════════════════════════════════════════

// humanAvgTime devuelve el tiempo humano típico por dificultad.
func humanAvgTime(d Difficulty) time.Duration {
	switch d {
	case DifficultyEasy:
		return 20 * time.Minute
	case DifficultyMedium:
		return 1 * time.Hour
	case DifficultyHard:
		return 3 * time.Hour
	case DifficultyInsane:
		return 8 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// humanAvgHints devuelve el número de pistas típicas.
func humanAvgHints(d Difficulty) float64 {
	switch d {
	case DifficultyEasy:
		return 0.5
	case DifficultyMedium:
		return 1.5
	case DifficultyHard:
		return 2.5
	case DifficultyInsane:
		return 3.5
	default:
		return 1.0
	}
}

// calculatePercentile calcula el percentil del agente vs humanos.
//   - Si el agente resuelve más rápido que la media → percentil > 50
//   - Si resuelve más lento → percentil < 50
//
// Fórmula simple: percentile = 100 * (1 - (agentTime / (2 * humanAvg)))
//   - Si agentTime = humanAvg → 50
//   - Si agentTime = 0 → 100
//   - Si agentTime = 2 * humanAvg → 0
func calculatePercentile(agentTime, humanAvg time.Duration) float64 {
	if humanAvg <= 0 {
		return 50.0
	}
	ratio := float64(agentTime) / float64(humanAvg)
	p := 100.0 * (1.0 - ratio/2.0)
	if p > 100 {
		p = 100
	}
	if p < 0 {
		p = 0
	}
	return p
}
