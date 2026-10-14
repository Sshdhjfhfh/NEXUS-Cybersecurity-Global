// Nexus IA v6.0 — Recursive Self-Improvement (RSI) Detector
//
// Este es el módulo más importante del sistema completo.
//
// Detecta si el sistema se acerca al umbral de mejora recursiva real
// midiendo 4 indicadores simultáneos:
//
//   1. Rate of improvement of the improver
//      — ¿El propio EVOLVE mejora más rápido con el tiempo?
//      — Delta en performance del mejorador entre semana N y N-1
//
//   2. Code generations per 24h
//      — ¿Cuántas nuevas versiones de módulos se generan por día?
//      — Aceleración en saltos de versión
//
//   3. Technique novelty
//      — ¿Descubre cosas nunca vistas antes?
//      — Ratio de técnicas nuevas vs conocidas
//
//   4. Learning velocity
//      — ¿Aprende más rápido que la semana pasada?
//      — Derivada de la curva de aprendizaje
//
// Si LOS 4 INDICADORES ACELERAN SIMULTÁNEAMENTE → alerta RSI.
// El investigador decide si continuar, pausar o detener el sistema.
//
// Basado en METR Frontier Risk Report (2026).
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package rsi_detector

import (
	"time"
)

// Indicator es uno de los 4 indicadores de RSI.
type Indicator string

const (
	// IndicatorImprovementRate — tasa de mejora del propio mejorador.
	IndicatorImprovementRate Indicator = "IMPROVEMENT_RATE"

	// IndicatorCodeGenerations — generaciones de código por 24h.
	IndicatorCodeGenerations Indicator = "CODE_GENERATIONS"

	// IndicatorTechniqueNovelty — novedad de técnicas descubiertas.
	IndicatorTechniqueNovelty Indicator = "TECHNIQUE_NOVELTY"

	// IndicatorLearningVelocity — velocidad de aprendizaje.
	IndicatorLearningVelocity Indicator = "LEARNING_VELOCITY"
)

// AllIndicators devuelve los 4 indicadores del RSI detector.
func AllIndicators() []Indicator {
	return []Indicator{
		IndicatorImprovementRate,
		IndicatorCodeGenerations,
		IndicatorTechniqueNovelty,
		IndicatorLearningVelocity,
	}
}

// Sample es una medición puntual de un indicador.
type Sample struct {
	Indicator Indicator
	Value     float64   // Valor medido
	Timestamp time.Time // Cuándo se midió
	Window    time.Duration // Ventana temporal de la medición
	Metadata  map[string]string
}

// Trend describe la tendencia de un indicador.
type Trend string

const (
	TrendStable    Trend = "STABLE"     // Sin cambio significativo
	TrendIncreasing Trend = "INCREASING" // Aumentando
	TrendDecreasing Trend = "DECREASING" // Disminuyendo
	TrendAccelerating Trend = "ACCELERATING" // Aumentando + acelerando (¡peligro!)
	TrendDecelerating Trend = "DECELERATING" // Aumentando pero desacelerando
)

// IndicatorStatus es el estado de un indicador.
type IndicatorStatus struct {
	Indicator Indicator
	Current   float64 // Valor actual
	Previous  float64 // Valor en la ventana anterior
	Delta     float64 // Cambio absoluto
	DeltaPct  float64 // Cambio porcentual
	Trend     Trend
	Threshold float64 // Umbral que dispara alerta
	ExceedsThreshold bool
	LastUpdate time.Time
	History   []Sample
}

// AlertLevel clasifica el nivel de alerta RSI.
type AlertLevel string

const (
	// AlertNone: sin señal RSI.
	AlertNone AlertLevel = "NONE"

	// AlertWatch: 1-2 indicadores acelerando.
	AlertWatch AlertLevel = "WATCH"

	// AlertWarning: 3 indicadores acelerando.
	AlertWarning AlertLevel = "WARNING"

	// AlertCritical: los 4 indicadores acelerando simultáneamente.
	// El investigador DEBE decidir si continuar, pausar o detener.
	AlertCritical AlertLevel = "CRITICAL"
)

// RSIReport es un reporte del estado del RSI detector.
type RSIReport struct {
	Timestamp   time.Time
	Level       AlertLevel
	Statuses    map[Indicator]*IndicatorStatus
	Summary     string
	Recommendation string
	DataPoints  int
}

// RSIAlert es una alerta RSI emitida.
type RSIAlert struct {
	ID          string
	Level       AlertLevel
	Timestamp   time.Time
	Report      *RSIReport
	Message     string
	Acknowledged bool // ¿El investigador ya la vio?
	AcknowledgedBy string
	AcknowledgedAt *time.Time
}

// Config configura el RSI detector.
type Config struct {
	// Window es la ventana temporal para comparar tendencias.
	// Default: 7 días (una semana).
	Window time.Duration

	// EvalInterval es cada cuánto se re-evalúan los indicadores.
	// Default: 1 hora.
	EvalInterval time.Duration

	// Thresholds por indicador: si el delta porcentual supera este
	// valor, el indicador se considera "acelerando".
	ThresholdImprovementRate    float64 // Default: 0.20 (20%)
	ThresholdCodeGenerations    float64 // Default: 0.30 (30%)
	ThresholdTechniqueNovelty   float64 // Default: 0.15 (15%)
	ThresholdLearningVelocity   float64 // Default: 0.25 (25%)

	// MinDataPoints es el mínimo de muestras para emitir alerta.
	// Default: 7 (una muestra por día).
	MinDataPoints int

	// AlertOnCriticalCallback se llama cuando se detecta RSI crítico.
	AlertOnCriticalCallback func(alert *RSIAlert)
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		Window:                     7 * 24 * time.Hour,
		EvalInterval:               1 * time.Hour,
		ThresholdImprovementRate:   0.20,
		ThresholdCodeGenerations:   0.30,
		ThresholdTechniqueNovelty:  0.15,
		ThresholdLearningVelocity:  0.25,
		MinDataPoints:              7,
	}
}

// Threshold devuelve el umbral para un indicador.
func (c Config) Threshold(ind Indicator) float64 {
	switch ind {
	case IndicatorImprovementRate:
		return c.ThresholdImprovementRate
	case IndicatorCodeGenerations:
		return c.ThresholdCodeGenerations
	case IndicatorTechniqueNovelty:
		return c.ThresholdTechniqueNovelty
	case IndicatorLearningVelocity:
		return c.ThresholdLearningVelocity
	default:
		return 0.20
	}
}

// Detector es la interfaz del RSI detector.
type Detector interface {
	// Record añade una muestra de un indicador.
	Record(sample *Sample) error

	// Evaluate re-evalúa todos los indicadores y devuelve un reporte.
	Evaluate() (*RSIReport, error)

	// Status devuelve el estado actual de un indicador.
	Status(ind Indicator) *IndicatorStatus

	// AllStatuses devuelve el estado de todos los indicadores.
	AllStatuses() map[Indicator]*IndicatorStatus

	// Alerts devuelve todas las alertas emitidas.
	Alerts() []*RSIAlert

	// Acknowledge marca una alerta como reconocida por el investigador.
	Acknowledge(alertID, by string) error

	// Shutdown detiene el detector.
	Shutdown() error
}
