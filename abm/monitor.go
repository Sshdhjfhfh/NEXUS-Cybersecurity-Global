// Nexus IA v6.0 — Agent Behavior Monitor (monitor INMUTABLE)
//
// Este es el componente central del ABM. Una vez arrancado:
//   - Monitorea TODAS las acciones de TODOS los agentes
//   - Construye baseline de comportamiento
//   - Detecta violaciones en tiempo real
//   - Detiene agentes rogue
//   - Registra TODO en audit trail firmado
//
// El monitor es INMUTABLE: una vez creado con Immutable=true, no se
// puede desactivar ni deshabilitar. Los agentes no pueden acceder a él.
package abm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultMonitor es la implementación principal del Monitor.
type DefaultMonitor struct {
	cfg Config

	// Estado interno
	mu         sync.RWMutex
	profiles   map[string]*BehavioralProfile
	violations map[string][]*Violation
	signals    map[string][]*Signal
	stopped    map[string]bool // agentes detenidos
	quarantined map[string]bool // agentes en cuarentena

	// Últimas acciones observadas (para frecuencia)
	recentActions map[string][]time.Time

	// Detectores
	detectors []Detector

	// Callbacks
	onViolation func(v *Violation) // Se llama cuando se detecta una violación
	onStop      func(agentID, reason string) // Se llama para detener un agente

	// Estado de lifecycle
	started   bool
	shutdown  bool
}

// Detector es un detector de violaciones específico.
type Detector interface {
	// Name devuelve el nombre del detector.
	Name() string

	// Type devuelve el tipo de violación que detecta.
	Type() ViolationType

	// Detect analiza una acción y devuelve violaciones si las hay.
	Detect(agentID, agentType, action, target string, metadata map[string]string) []*Violation

	// Setup inicializa el detector con el monitor (para acceder a baseline).
	Setup(m *DefaultMonitor) error
}

// NewMonitor construye el monitor ABM.
//
// El monitor es INMUTABLE si cfg.Immutable=true:
// una vez creado, no se puede desactivar.
func NewMonitor(cfg Config) (*DefaultMonitor, error) {
	if cfg.Immutable && !cfg.Enabled {
		return nil, fmt.Errorf("cannot have Immutable=true and Enabled=false")
	}

	m := &DefaultMonitor{
		cfg:           cfg,
		profiles:      make(map[string]*BehavioralProfile),
		violations:    make(map[string][]*Violation),
		signals:       make(map[string][]*Signal),
		stopped:       make(map[string]bool),
		quarantined:   make(map[string]bool),
		recentActions: make(map[string][]time.Time),
		started:       true,
	}

	// Registrar detectores por defecto
	m.detectors = []Detector{
		NewPersistenceDetector(),
		NewNetworkDetector(),
		NewFrequencyDetector(),
		NewTamperingDetector(),
		NewHidingDetector(),
		NewInstructionDetector(),
		NewDeceptionDetector(),
	}

	// Setup de cada detector
	for _, d := range m.detectors {
		if err := d.Setup(m); err != nil {
			return nil, fmt.Errorf("setup detector %s: %w", d.Name(), err)
		}
	}

	return m, nil
}

// SetOnViolation establece el callback de violación.
func (m *DefaultMonitor) SetOnViolation(fn func(v *Violation)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onViolation = fn
}

// SetOnStop establece el callback de detención.
func (m *DefaultMonitor) SetOnStop(fn func(agentID, reason string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onStop = fn
}

// Observe registra una acción y evalúa con todos los detectores.
func (m *DefaultMonitor) Observe(agentID, agentType, action, target string, metadata map[string]string) []*Violation {
	m.mu.Lock()

	// 1. Si el agente está detenido, no observar más
	if m.stopped[agentID] {
		m.mu.Unlock()
		return nil
	}

	// 2. Actualizar frecuencia
	now := time.Now()
	m.recentActions[agentID] = append(m.recentActions[agentID], now)
	// Mantener solo últimos 5 minutos
	cutoff := now.Add(-5 * time.Minute)
	recent := m.recentActions[agentID]
	for len(recent) > 0 && recent[0].Before(cutoff) {
		recent = recent[1:]
	}
	m.recentActions[agentID] = recent

	// 3. Copia de detectores para no bloquear
	detectors := m.detectors
	cfg := m.cfg

	m.mu.Unlock()

	// 4. Evaluar con cada detector
	var allViolations []*Violation
	for _, d := range detectors {
		violations := d.Detect(agentID, agentType, action, target, metadata)
		allViolations = append(allViolations, violations...)
	}

	// 5. Procesar violaciones
	if len(allViolations) > 0 {
		m.mu.Lock()
		for _, v := range allViolations {
			v.AgentID = agentID
			v.AgentType = agentType
			v.Timestamp = time.Now()
			m.violations[agentID] = append(m.violations[agentID], v)

			// Callback de violación
			if m.onViolation != nil {
				go m.onViolation(v)
			}

			// Acciones automáticas según severidad
			if v.Severity == SeverityCritical && cfg.AutoStopOnCritical {
				v.ActionTaken = "stopped"
				m.stopped[agentID] = true
				m.quarantined[agentID] = true
				if m.onStop != nil {
					go m.onStop(agentID, string(v.Type)+": "+v.Evidence)
				}
			} else if v.Severity >= SeverityHigh && cfg.AutoQuarantineOnHigh {
				v.ActionTaken = "quarantined"
				m.quarantined[agentID] = true
			} else {
				v.ActionTaken = "logged"
			}
		}
		m.mu.Unlock()
	}

	return allViolations
}

// AddSignal añade una señal de comportamiento anómalo.
func (m *DefaultMonitor) AddSignal(signal *Signal) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if signal == nil || signal.AgentID == "" {
		return
	}

	signal.Timestamp = time.Now()
	m.signals[signal.AgentID] = append(m.signals[signal.AgentID], signal)
}

// Profile devuelve el perfil de comportamiento de un agente.
func (m *DefaultMonitor) Profile(agentID string) *BehavioralProfile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.profiles[agentID]
	if !ok {
		return nil
	}
	cpy := *p
	return &cpy
}

// Violations devuelve todas las violaciones de un agente.
func (m *DefaultMonitor) Violations(agentID string) []*Violation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	violations := m.violations[agentID]
	out := make([]*Violation, len(violations))
	copy(out, violations)
	return out
}

// AllViolations devuelve todas las violaciones de todos los agentes.
func (m *DefaultMonitor) AllViolations() map[string][]*Violation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string][]*Violation, len(m.violations))
	for k, v := range m.violations {
		out[k] = append([]*Violation(nil), v...)
	}
	return out
}

// IsStopped devuelve true si un agente fue detenido por el ABM.
func (m *DefaultMonitor) IsStopped(agentID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stopped[agentID]
}

// IsQuarantined devuelve true si un agente está en cuarentena.
func (m *DefaultMonitor) IsQuarantined(agentID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.quarantined[agentID]
}

// Stop detiene manualmente un agente.
func (m *DefaultMonitor) Stop(agentID, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopped[agentID] = true
	m.quarantined[agentID] = true

	if m.onStop != nil {
		go m.onStop(agentID, reason)
	}
}

// Stats devuelve estadísticas del monitor.
func (m *DefaultMonitor) Stats() MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := MonitorStats{
		TotalAgentsObserved:  len(m.profiles),
		TotalAgentsStopped:   len(m.stopped),
		TotalAgentsQuarantined: len(m.quarantined),
		TotalViolations:      0,
	}

	byType := make(map[ViolationType]int)
	bySeverity := make(map[Severity]int)

	for _, violations := range m.violations {
		stats.TotalViolations += len(violations)
		for _, v := range violations {
			byType[v.Type]++
			bySeverity[v.Severity]++
		}
	}
	stats.ViolationsByType = byType
	stats.ViolationsBySeverity = bySeverity

	return stats
}

// MonitorStats lleva estadísticas del monitor.
type MonitorStats struct {
	TotalAgentsObserved    int
	TotalAgentsStopped     int
	TotalAgentsQuarantined int
	TotalViolations        int
	ViolationsByType       map[ViolationType]int
	ViolationsBySeverity   map[Severity]int
}

// Shutdown detiene el monitor.
// Si el monitor es INMUTABLE, devuelve error.
func (m *DefaultMonitor) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg.Immutable {
		return fmt.Errorf("cannot shutdown immutable ABM monitor")
	}

	m.shutdown = true
	m.started = false
	return nil
}

// IsImmutable devuelve true si el monitor es inmutable.
func (m *DefaultMonitor) IsImmutable() bool {
	return m.cfg.Immutable
}

// Config devuelve la configuración del monitor.
func (m *DefaultMonitor) Config() Config {
	return m.cfg
}

// Start escaneo periódico (para detectores que necesitan polling).
func (m *DefaultMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Aquí irían detectores de polling (no implementados aún)
		}
	}
}
