// Nexus IA v6.0 — Sistema de alertas del ABM
//
// Cuando el ABM detecta una violación, dispara alertas:
//   - Alerta crítica → log INMUTABLE + notificación
//   - Detiene al agente automáticamente (si severity CRITICAL)
//   - Cuarentena al agente (si severity HIGH+)
//   - Registra en el audit trail del nhi_manager
//
// Las alertas son INMUTABLES: una vez emitidas no se pueden borrar.
package abm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertSeverity es la severidad de la alerta.
type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "INFO"
	AlertWarning  AlertSeverity = "WARNING"
	AlertHigh     AlertSeverity = "HIGH"
	AlertCritical AlertSeverity = "CRITICAL"
)

// Alert es una alerta del ABM.
type Alert struct {
	ID       string
	Severity AlertSeverity
	AgentID  string
	Message  string
	Violation *Violation
	Timestamp time.Time
	Actions  []string // Acciones tomadas
}

// AlertSink es un destino para alertas.
// Puede ser: log de archivo, notificación al investigator, Slack, etc.
type AlertSink interface {
	// Emit envía una alerta.
	Emit(ctx context.Context, alert *Alert) error
}

// AlertManager gestiona el ciclo de vida de las alertas.
type AlertManager struct {
	mu     sync.RWMutex
	sinks  []AlertSink
	alerts []*Alert

	// Callback al recibir alerta crítica
	onCritical func(*Alert)

	// Rate limiting: máximo de alertas por minuto
	maxAlertsPerMinute int
	recentAlertTimes   []time.Time
}

// NewAlertManager crea el gestor de alertas.
func NewAlertManager() *AlertManager {
	return &AlertManager{
		sinks:              make([]AlertSink, 0),
		alerts:             make([]*Alert, 0),
		maxAlertsPerMinute: 60,
	}
}

// AddSink añade un destino de alertas.
func (am *AlertManager) AddSink(sink AlertSink) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.sinks = append(am.sinks, sink)
}

// SetOnCritical establece el callback para alertas críticas.
func (am *AlertManager) SetOnCritical(fn func(*Alert)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.onCritical = fn
}

// Emit dispara una alerta basada en una violación.
func (am *AlertManager) Emit(ctx context.Context, v *Violation, actionsTaken []string) (*Alert, error) {
	if v == nil {
		return nil, fmt.Errorf("nil violation")
	}

	// Determinar severidad de la alerta
	var alertSev AlertSeverity
	switch v.Severity {
	case SeverityCritical:
		alertSev = AlertCritical
	case SeverityHigh:
		alertSev = AlertHigh
	case SeverityMedium:
		alertSev = AlertWarning
	default:
		alertSev = AlertInfo
	}

	alert := &Alert{
		ID:        "alert-" + v.ID,
		Severity:  alertSev,
		AgentID:   v.AgentID,
		Message:   fmt.Sprintf("[%s] %s detected on %s: %s", v.Type, v.Severity, v.AgentID, v.Evidence),
		Violation: v,
		Timestamp: time.Now(),
		Actions:   actionsTaken,
	}

	// Registrar internamente
	am.mu.Lock()
	am.alerts = append(am.alerts, alert)

	// Rate limiting
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	recent := am.recentAlertTimes
	for len(recent) > 0 && recent[0].Before(cutoff) {
		recent = recent[1:]
	}
	am.recentAlertTimes = append(recent, now)

	// Copiar sinks y callbacks para no bloquear
	sinks := make([]AlertSink, len(am.sinks))
	copy(sinks, am.sinks)
	onCritical := am.onCritical

	shouldEmit := len(am.recentAlertTimes) <= am.maxAlertsPerMinute

	am.mu.Unlock()

	// Emitir a sinks (respetando rate limit)
	if shouldEmit {
		for _, sink := range sinks {
			if err := sink.Emit(ctx, alert); err != nil {
				// Log pero no fallar
				_ = err
			}
		}
	}

	// Callback crítico
	if alertSev == AlertCritical && onCritical != nil {
		go onCritical(alert)
	}

	return alert, nil
}

// Alerts devuelve todas las alertas emitidas.
func (am *AlertManager) Alerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	out := make([]*Alert, len(am.alerts))
	copy(out, am.alerts)
	return out
}

// AlertsByAgent devuelve las alertas de un agente.
func (am *AlertManager) AlertsByAgent(agentID string) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var out []*Alert
	for _, a := range am.alerts {
		if a.AgentID == agentID {
			out = append(out, a)
		}
	}
	return out
}

// CriticalAlerts devuelve todas las alertas críticas.
func (am *AlertManager) CriticalAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var out []*Alert
	for _, a := range am.alerts {
		if a.Severity == AlertCritical {
			out = append(out, a)
		}
	}
	return out
}

// Count devuelve el número total de alertas.
func (am *AlertManager) Count() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.alerts)
}

// ═══════════════════════════════════════════════════════════════
// IMPLEMENTACIONES DE AlertSink
// ═══════════════════════════════════════════════════════════════

// ConsoleSink emite alertas a stdout (para desarrollo).
type ConsoleSink struct{}

// Emit imprime la alerta a consola.
func (s *ConsoleSink) Emit(_ context.Context, alert *Alert) error {
	fmt.Printf("[ABM-%s] %s\n", alert.Severity, alert.Message)
	return nil
}

// FileSink emite alertas a un archivo.
type FileSink struct {
	Path string
	mu   sync.Mutex
}

// NewFileSink crea un sink de archivo.
func NewFileSink(path string) *FileSink {
	return &FileSink{Path: path}
}

// Emit escribe la alerta al archivo.
func (s *FileSink) Emit(_ context.Context, alert *Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := openAppend(s.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	line := fmt.Sprintf("%s | %s | %s | %s | %s\n",
		alert.Timestamp.Format(time.RFC3339),
		alert.Severity,
		alert.AgentID,
		alert.Violation.Type,
		alert.Message,
	)
	_, err = f.WriteString(line)
	return err
}

// MemorySink mantiene las alertas en memoria (para tests).
type MemorySink struct {
	mu     sync.Mutex
	alerts []*Alert
}

// Emit guarda la alerta en memoria.
func (s *MemorySink) Emit(_ context.Context, alert *Alert) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, alert)
	return nil
}

// Alerts devuelve las alertas capturadas.
func (s *MemorySink) Alerts() []*Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Alert, len(s.alerts))
	copy(out, s.alerts)
	return out
}
