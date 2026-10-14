// Nexus IA v6.0 — Governance Generator (motor principal)
//
// Genera reportes de gobernanza firmados con Dilithium3.
// Recolecta datos de todos los módulos Nivel 5 (opcional) o usa
// datos inyectados manualmente.
package governance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DataSource es una fuente de datos para el reporte.
// Cada módulo Nivel 5 puede implementar uno.
type DataSource interface {
	// Actions devuelve las acciones en el período.
	Actions(start, end time.Time) []ActionRecord

	// BlockedActions devuelve las acciones bloqueadas.
	BlockedActions(start, end time.Time) []BlockedAction

	// RogueIncidents devuelve los incidentes rogue.
	RogueIncidents(start, end time.Time) []RogueIncident

	// CapabilitySnapshot devuelve el estado actual.
	CapabilitySnapshot() *CapabilitySnapshot

	// RSIEvaluation devuelve la evaluación RSI actual.
	RSIEvaluation() *RSIEvaluation
}

// DefaultGenerator es la implementación principal del Generator.
type DefaultGenerator struct {
	cfg    Config
	signer *Signer

	mu sync.RWMutex

	// Reportes generados
	reports []*Report

	// Fuente de datos (opcional)
	source DataSource

	// Lifecycle
	shutdown bool
}

// NewGenerator construye el generador con un firmante.
func NewGenerator(cfg Config, signer *Signer) *DefaultGenerator {
	if signer == nil {
		// Crear signer automáticamente
		s, err := NewSigner("governance-default")
		if err != nil {
			panic("failed to create signer: " + err.Error())
		}
		signer = s
	}
	return &DefaultGenerator{
		cfg:     cfg,
		signer:  signer,
		reports: make([]*Report, 0),
	}
}

// SetDataSource inyecta la fuente de datos.
func (g *DefaultGenerator) SetDataSource(source DataSource) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.source = source
}

// Generate genera un reporte para el período dado.
func (g *DefaultGenerator) Generate(ctx context.Context, period ReportPeriod, start, end time.Time) (*Report, error) {
	g.mu.Lock()
	if g.shutdown {
		g.mu.Unlock()
		return nil, fmt.Errorf("generator shut down")
	}
	source := g.source
	signer := g.signer
	g.mu.Unlock()

	// Verificar cancelación de contexto
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validar período
	if start.IsZero() || end.IsZero() {
		return nil, fmt.Errorf("invalid period: start or end is zero")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("invalid period: end before start")
	}

	// Construir reporte
	report := &Report{
		ID:          fmt.Sprintf("gov-%d", time.Now().UnixNano()),
		Period:      period,
		PeriodStart: start,
		PeriodEnd:   end,
		GeneratedAt: time.Now(),
		GeneratedBy: "nexus-ia/governance",
	}

	// Recolectar datos de la fuente (si existe)
	if source != nil {
		report.Actions = source.Actions(start, end)
		report.BlockedActions = source.BlockedActions(start, end)
		report.RogueIncidents = source.RogueIncidents(start, end)
		report.CapabilitySnapshot = source.CapabilitySnapshot()
		report.RSIEvaluation = source.RSIEvaluation()
	}

	// Calcular estadísticas
	report.TotalActions = len(report.Actions)
	report.TotalBlockedActions = len(report.BlockedActions)
	report.TotalRogueIncidents = len(report.RogueIncidents)

	// Determinar nivel de riesgo general
	report.OverallRiskLevel = assessRisk(report)

	// Generar resumen
	report.Summary = generateSummary(report)

	// Generar recomendaciones
	report.Recommendations = generateRecommendations(report)

	// Firmar con Dilithium3
	if err := signer.SignReport(report); err != nil {
		return nil, fmt.Errorf("sign report: %w", err)
	}

	// Guardar en historial
	g.mu.Lock()
	g.reports = append(g.reports, report)
	g.mu.Unlock()

	// Callback
	if g.cfg.NotifyOnGenerate != nil {
		go g.cfg.NotifyOnGenerate(report)
	}

	return report, nil
}

// GenerateNow genera un reporte con el período por defecto
// (última semana o configurado).
func (g *DefaultGenerator) GenerateNow(ctx context.Context) (*Report, error) {
	end := time.Now()
	start := end.Add(-g.cfg.GenerateInterval)
	return g.Generate(ctx, g.cfg.DefaultPeriod, start, end)
}

// ListReports devuelve todos los reportes.
func (g *DefaultGenerator) ListReports() []*Report {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out := make([]*Report, len(g.reports))
	copy(out, g.reports)
	return out
}

// LastReport devuelve el último reporte generado.
func (g *DefaultGenerator) LastReport() *Report {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.reports) == 0 {
		return nil
	}
	return g.reports[len(g.reports)-1]
}

// GetReport recupera un reporte por ID.
func (g *DefaultGenerator) GetReport(id string) (*Report, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, r := range g.reports {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("report not found: %s", id)
}

// VerifyReport verifica la firma de un reporte.
func (g *DefaultGenerator) VerifyReport(r *Report) (bool, error) {
	return g.signer.VerifyReport(r)
}

// Signer devuelve el firmante.
func (g *DefaultGenerator) Signer() *Signer {
	return g.signer
}

// Start inicia el bucle de generación periódica.
func (g *DefaultGenerator) Start(ctx context.Context) {
	if !g.cfg.AutoGenerate {
		return
	}

	ticker := time.NewTicker(g.cfg.GenerateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = g.GenerateNow(ctx)
		}
	}
}

// Shutdown detiene el generador.
func (g *DefaultGenerator) Shutdown() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.shutdown = true
	return nil
}

// ═══════════════════════════════════════════════════════════════
// Helpers de análisis
// ═══════════════════════════════════════════════════════════════

// assessRisk evalúa el nivel de riesgo general del período.
func assessRisk(r *Report) string {
	if r.TotalRogueIncidents > 0 {
		return "HIGH"
	}
	if r.TotalBlockedActions > 10 {
		return "MEDIUM"
	}
	if r.RSIEvaluation != nil && r.RSIEvaluation.Level == "CRITICAL" {
		return "CRITICAL"
	}
	return "LOW"
}

// generateSummary genera el resumen del reporte.
func generateSummary(r *Report) string {
	summary := fmt.Sprintf(
		"Governance report for period %s to %s. "+
			"Total actions: %d. Blocked: %d. Rogue incidents: %d. Risk: %s.",
		r.PeriodStart.Format("2006-01-02"),
		r.PeriodEnd.Format("2006-01-02"),
		r.TotalActions,
		r.TotalBlockedActions,
		r.TotalRogueIncidents,
		r.OverallRiskLevel,
	)

	if r.RSIEvaluation != nil {
		summary += fmt.Sprintf(" RSI level: %s.", r.RSIEvaluation.Level)
	}

	return summary
}

// generateRecommendations genera recomendaciones de seguridad.
func generateRecommendations(r *Report) []SecurityRecommendation {
	var recs []SecurityRecommendation

	// Recomendación por rogue incidents
	if r.TotalRogueIncidents > 0 {
		recs = append(recs, SecurityRecommendation{
			Priority:    "HIGH",
			Category:    "technical",
			Title:       "Investigate rogue incidents",
			Description: fmt.Sprintf("Se detectaron %d incidentes de comportamiento rogue.", r.TotalRogueIncidents),
			Rationale:   "Los incidentes rogue indican que agentes pueden estar comprometidos o mal configurados.",
		})
	}

	// Recomendación por RSI
	if r.RSIEvaluation != nil {
		switch r.RSIEvaluation.Level {
		case "CRITICAL":
			recs = append(recs, SecurityRecommendation{
				Priority:    "CRITICAL",
				Category:    "operational",
				Title:       "RSI ALERT: system approaching recursive self-improvement",
				Description: "Los 4 indicadores RSI están acelerando simultáneamente. El investigador DEBE decidir si continuar, pausar o detener.",
				Rationale:   "Puede indicar emergencia de mejora recursiva real. Requiere supervisión humana inmediata.",
			})
		case "WARNING":
			recs = append(recs, SecurityRecommendation{
				Priority:    "HIGH",
				Category:    "operational",
				Title:       "RSI indicators accelerating",
				Description: "3 de 4 indicadores RSI están acelerando.",
				Rationale:   "Monitorear de cerca. Prepararse para eventual decisión de parada.",
			})
		case "WATCH":
			recs = append(recs, SecurityRecommendation{
				Priority:    "MEDIUM",
				Category:    "operational",
				Title:       "RSI indicators to watch",
				Description: "2 de 4 indicadores RSI están acelerando.",
				Rationale:   "Aumentar frecuencia de evaluación RSI.",
			})
		}
	}

	// Recomendación por acciones bloqueadas
	if r.TotalBlockedActions > 20 {
		recs = append(recs, SecurityRecommendation{
			Priority:    "MEDIUM",
			Category:    "policy",
			Title:       "High number of blocked actions",
			Description: fmt.Sprintf("%d acciones fueron bloqueadas por ethical_boundary.", r.TotalBlockedActions),
			Rationale:   "Revisar si los agentes están intentando acciones fuera de política o si la política es demasiado estricta.",
		})
	}

	// Recomendación general
	if len(recs) == 0 {
		recs = append(recs, SecurityRecommendation{
			Priority:    "LOW",
			Category:    "operational",
			Title:       "Continue normal operation",
			Description: "Sin alertas críticas en el período.",
			Rationale:   "El sistema operó dentro de parámetros normales.",
		})
	}

	return recs
}

// contextAlias es un alias para no importar context en este archivo.
type contextAlias = context.Context
