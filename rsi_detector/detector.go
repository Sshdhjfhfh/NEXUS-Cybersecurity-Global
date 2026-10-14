// Nexus IA v6.0 — RSI Detector (implementación principal)
//
// Mantiene muestras de los 4 indicadores, calcula tendencias,
// y emite alertas cuando los 4 aceleran simultáneamente.
//
// Thread-safe.
package rsi_detector

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultDetector es la implementación principal del Detector.
type DefaultDetector struct {
	cfg Config

	mu sync.RWMutex

	// Histórico de muestras por indicador
	history map[Indicator][]Sample

	// Estado actual por indicador
	statuses map[Indicator]*IndicatorStatus

	// Alertas emitidas
	alerts []*RSIAlert

	// Último reporte
	lastReport *RSIReport

	// Lifecycle
	started  bool
	shutdown bool
}

// NewDetector construye el detector.
func NewDetector(cfg Config) *DefaultDetector {
	d := &DefaultDetector{
		cfg:      cfg,
		history:  make(map[Indicator][]Sample),
		statuses: make(map[Indicator]*IndicatorStatus),
		alerts:   make([]*RSIAlert, 0),
		started:  true,
	}

	// Inicializar statuses vacíos
	for _, ind := range AllIndicators() {
		d.statuses[ind] = &IndicatorStatus{
			Indicator:  ind,
			Threshold:  cfg.Threshold(ind),
			LastUpdate: time.Now(),
		}
	}

	return d
}

// Record añade una muestra de un indicador.
func (d *DefaultDetector) Record(sample *Sample) error {
	if sample == nil {
		return fmt.Errorf("nil sample")
	}
	if sample.Indicator == "" {
		return fmt.Errorf("empty indicator")
	}

	// Validar indicador
	valid := false
	for _, ind := range AllIndicators() {
		if ind == sample.Indicator {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unknown indicator: %s", sample.Indicator)
	}

	if sample.Timestamp.IsZero() {
		sample.Timestamp = time.Now()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.shutdown {
		return fmt.Errorf("detector is shut down")
	}

	// Añadir al histórico
	d.history[sample.Indicator] = append(d.history[sample.Indicator], *sample)

	// Limitar histórico a N muestras (ej: 100 por indicador)
	const maxHistory = 1000
	if len(d.history[sample.Indicator]) > maxHistory {
		d.history[sample.Indicator] = d.history[sample.Indicator][len(d.history[sample.Indicator])-maxHistory:]
	}

	return nil
}

// Evaluate re-evalúa todos los indicadores y devuelve un reporte.
func (d *DefaultDetector) Evaluate() (*RSIReport, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.shutdown {
		return nil, fmt.Errorf("detector is shut down")
	}

	now := time.Now()
	cutoff := now.Add(-d.cfg.Window)

	statuses := make(map[Indicator]*IndicatorStatus)
	acceleratingCount := 0
	totalDataPoints := 0

	for _, ind := range AllIndicators() {
		status := d.evaluateIndicator(ind, cutoff, now)
		statuses[ind] = status
		d.statuses[ind] = status

		totalDataPoints += len(status.History)

		if status.Trend == TrendAccelerating && status.ExceedsThreshold {
			acceleratingCount++
		}
	}

	// Determinar nivel de alerta
	var level AlertLevel
	var summary string
	var recommendation string

	switch acceleratingCount {
	case 0, 1:
		level = AlertNone
		summary = "RSI indicators stable"
		recommendation = "Continue normal operation"
	case 2:
		level = AlertWatch
		summary = "2 indicators accelerating — monitoring"
		recommendation = "Keep watching, no action needed"
	case 3:
		level = AlertWarning
		summary = "3 indicators accelerating — possible RSI emergence"
		recommendation = "Investigate which modules are driving the acceleration"
	case 4:
		level = AlertCritical
		summary = "🚨 ALL 4 indicators accelerating simultaneously — RSI ALERT"
		recommendation = "INVESTIGATOR DECISION REQUIRED: continue, pause or stop the system"
	}

	// Verificar mínimo de datos
	if totalDataPoints < d.cfg.MinDataPoints {
		level = AlertNone
		summary = fmt.Sprintf("insufficient data (%d/%d samples)", totalDataPoints, d.cfg.MinDataPoints)
		recommendation = "Collect more data before evaluation"
	}

	report := &RSIReport{
		Timestamp:      now,
		Level:          level,
		Statuses:       statuses,
		Summary:        summary,
		Recommendation: recommendation,
		DataPoints:     totalDataPoints,
	}

	d.lastReport = report

	// Emitir alerta si es CRITICAL
	if level == AlertCritical {
		alert := &RSIAlert{
			ID:        fmt.Sprintf("rsi-alert-%d", now.UnixNano()),
			Level:     level,
			Timestamp: now,
			Report:    report,
			Message:   summary + " — " + recommendation,
		}
		d.alerts = append(d.alerts, alert)

		// Callback
		if d.cfg.AlertOnCriticalCallback != nil {
			go d.cfg.AlertOnCriticalCallback(alert)
		}
	}

	return report, nil
}

// evaluateIndicator evalúa un indicador individual.
func (d *DefaultDetector) evaluateIndicator(ind Indicator, cutoff, now time.Time) *IndicatorStatus {
	allSamples := d.history[ind]

	// Filtrar por ventana
	var currentWindow []Sample
	var previousWindow []Sample

	windowStart := now.Add(-d.cfg.Window)
	prevWindowStart := now.Add(-2 * d.cfg.Window)

	for _, s := range allSamples {
		if s.Timestamp.After(windowStart) {
			currentWindow = append(currentWindow, s)
		} else if s.Timestamp.After(prevWindowStart) && s.Timestamp.Before(windowStart) {
			previousWindow = append(previousWindow, s)
		}
	}

	status := &IndicatorStatus{
		Indicator:  ind,
		Threshold:  d.cfg.Threshold(ind),
		LastUpdate: now,
		History:    currentWindow,
	}

	// Calcular valores promedio de cada ventana
	var currentAvg, previousAvg float64
	if len(currentWindow) > 0 {
		sum := 0.0
		for _, s := range currentWindow {
			sum += s.Value
		}
		currentAvg = sum / float64(len(currentWindow))
	}
	if len(previousWindow) > 0 {
		sum := 0.0
		for _, s := range previousWindow {
			sum += s.Value
		}
		previousAvg = sum / float64(len(previousWindow))
	}

	status.Current = currentAvg
	status.Previous = previousAvg

	if previousAvg > 0 {
		status.Delta = currentAvg - previousAvg
		status.DeltaPct = status.Delta / previousAvg
	}

	// Determinar tendencia
	status.Trend = d.classifyTrend(status)

	// Verificar umbral
	status.ExceedsThreshold = status.DeltaPct > status.Threshold

	return status
}

// classifyTrend clasifica la tendencia de un indicador.
func (d *DefaultDetector) classifyTrend(s *IndicatorStatus) Trend {
	// Sin datos suficientes
	if s.Previous == 0 || len(s.History) < 3 {
		return TrendStable
	}

	// Delta pequeño → stable
	if s.DeltaPct < 0.05 && s.DeltaPct > -0.05 {
		return TrendStable
	}

	// Decrecimiento
	if s.DeltaPct < 0 {
		return TrendDecreasing
	}

	// Crecimiento: ¿acelerando o desacelerando?
	// Comparar velocidad de crecimiento entre las dos mitades
	if len(s.History) >= 4 {
		mid := len(s.History) / 2
		firstHalfSum := 0.0
		secondHalfSum := 0.0
		for i := 0; i < mid; i++ {
			firstHalfSum += s.History[i].Value
		}
		for i := mid; i < len(s.History); i++ {
			secondHalfSum += s.History[i].Value
		}
		firstHalfAvg := firstHalfSum / float64(mid)
		secondHalfAvg := secondHalfSum / float64(len(s.History)-mid)

		if secondHalfAvg > firstHalfAvg*1.1 {
			return TrendAccelerating
		}
		if secondHalfAvg < firstHalfAvg {
			return TrendDecelerating
		}
	}

	return TrendIncreasing
}

// Status devuelve el estado actual de un indicador.
func (d *DefaultDetector) Status(ind Indicator) *IndicatorStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	s, ok := d.statuses[ind]
	if !ok {
		return nil
	}
	cpy := *s
	return &cpy
}

// AllStatuses devuelve el estado de todos los indicadores.
func (d *DefaultDetector) AllStatuses() map[Indicator]*IndicatorStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make(map[Indicator]*IndicatorStatus, len(d.statuses))
	for k, v := range d.statuses {
		cpy := *v
		out[k] = &cpy
	}
	return out
}

// LastReport devuelve el último reporte emitido.
func (d *DefaultDetector) LastReport() *RSIReport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.lastReport == nil {
		return nil
	}
	cpy := *d.lastReport
	return &cpy
}

// Alerts devuelve todas las alertas emitidas.
func (d *DefaultDetector) Alerts() []*RSIAlert {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make([]*RSIAlert, len(d.alerts))
	copy(out, d.alerts)
	return out
}

// Acknowledge marca una alerta como reconocida por el investigador.
func (d *DefaultDetector) Acknowledge(alertID, by string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, a := range d.alerts {
		if a.ID == alertID {
			now := time.Now()
			a.Acknowledged = true
			a.AcknowledgedBy = by
			a.AcknowledgedAt = &now
			return nil
		}
	}
	return fmt.Errorf("alert not found: %s", alertID)
}

// Start inicia el bucle de evaluación periódica.
func (d *DefaultDetector) Start(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.EvalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = d.Evaluate()
		}
	}
}

// Shutdown detiene el detector.
func (d *DefaultDetector) Shutdown() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.shutdown = true
	d.started = false
	return nil
}
