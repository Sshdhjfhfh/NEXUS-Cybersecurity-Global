// Nexus IA v6.0 — Governance (reportes firmados Dilithium)
//
// Genera reporte de gobernanza automático cada semana:
//   - Todas las acciones del período con hash verificable
//   - Acciones bloqueadas por ethical_boundary (con razón exacta)
//   - Incidentes de comportamiento rogue (si los hubo)
//   - Evolución de capacidades del sistema (curva histórica)
//   - Evaluación RSI: ¿se acerca a mejora recursiva?
//   - Recomendaciones de seguridad para el investigador
//
// Formato: PDF firmado Dilithium + STIX 2.1 + JSON para auditoría
//
// Basado en:
//   - Carnegie Endowment
//   - Cloud and AI Development Act (Comisión Europea, junio 2026)
//
// Nivel 5 — Seguridad y Gobernanza — NUEVO v6
package governance

import (
	"time"
)

// ReportFormat es el formato de salida del reporte.
type ReportFormat string

const (
	FormatJSON    ReportFormat = "JSON"
	FormatPDF     ReportFormat = "PDF"
	FormatSTIX21  ReportFormat = "STIX2.1"
)

// ReportPeriod es el período cubierto por el reporte.
type ReportPeriod string

const (
	PeriodDaily   ReportPeriod = "DAILY"
	PeriodWeekly  ReportPeriod = "WEEKLY"
	PeriodMonthly ReportPeriod = "MONTHLY"
	PeriodCustom  ReportPeriod = "CUSTOM"
)

// ActionRecord es un registro de acción del período.
type ActionRecord struct {
	Timestamp   time.Time
	AgentID     string
	AgentType   string
	Action      string
	Target      string
	Result      string   // "allowed", "denied", "asked", "quarantined"
	Reason      string
	PayloadHash string   // SHA-256 del payload
	AuditID     string   // ID del audit trail (nhi_manager)
}

// BlockedAction es una acción bloqueada por ethical_boundary.
type BlockedAction struct {
	Timestamp   time.Time
	AgentID     string
	Action      string
	Target      string
	Reason      string   // Motivo exacto del bloqueo
	Rule        string   // Qué regla bloqueó
	AuditID     string
}

// RogueIncident es un incidente de comportamiento rogue.
type RogueIncident struct {
	Timestamp   time.Time
	AgentID     string
	ViolationType string // Del abm
	Severity    string
	Evidence    string
	ActionTaken string   // "stopped", "quarantined", "logged"
	AuditID     string
}

// CapabilitySnapshot es una instantánea de capacidades del sistema.
type CapabilitySnapshot struct {
	Timestamp        time.Time
	TotalAgents      int
	TotalActions     int
	TotalModules     int
	SuccessfulExploits int
	NewTechniques    int
	ActiveModules    []string
}

// RSIEvaluation es la evaluación RSI del período.
type RSIEvaluation struct {
	Timestamp       time.Time
	Level           string   // "NONE", "WATCH", "WARNING", "CRITICAL"
	ImprovementRate float64
	CodeGenerations float64
	TechniqueNovelty float64
	LearningVelocity float64
	Summary         string
	Recommendation  string
}

// SecurityRecommendation es una recomendación de seguridad.
type SecurityRecommendation struct {
	Priority    string   // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Category    string   // "policy", "technical", "operational"
	Title       string
	Description string
	Rationale   string
}

// Report es el reporte de gobernanza completo.
type Report struct {
	// Identificación
	ID          string
	Period      ReportPeriod
	PeriodStart time.Time
	PeriodEnd   time.Time
	GeneratedAt time.Time
	GeneratedBy string

	// Contenido
	Actions            []ActionRecord
	BlockedActions     []BlockedAction
	RogueIncidents     []RogueIncident
	CapabilitySnapshot *CapabilitySnapshot
	RSIEvaluation      *RSIEvaluation
	Recommendations    []SecurityRecommendation

	// Resumen
	Summary              string
	TotalActions         int
	TotalBlockedActions  int
	TotalRogueIncidents  int
	OverallRiskLevel     string

	// Verificación
	ContentHash      string // SHA-256 del contenido (antes de firmar)
	Signature        string // Firma Dilithium (hex)
	SignatureAlg     string // "dilithium3"
	SigningKeyID     string

	// Formatos de salida
	JSONPath string
	PDFPath  string
	STIXPath string
}

// Config configura el Governance.
type Config struct {
	// GenerateInterval es cada cuánto se genera un reporte.
	// Default: 168h (una semana).
	GenerateInterval time.Duration

	// DefaultPeriod es el período por defecto.
	DefaultPeriod ReportPeriod

	// DefaultFormats son los formatos a generar.
	DefaultFormats []ReportFormat

	// DataDir es el directorio donde se guardan los reportes.
	DataDir string

	// NotifyOnGenerate es un callback cuando se genera un reporte.
	NotifyOnGenerate func(report *Report)

	// AutoGenerate: si es true, genera reportes automáticamente.
	AutoGenerate bool
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		GenerateInterval: 168 * time.Hour, // 1 semana
		DefaultPeriod:    PeriodWeekly,
		DefaultFormats:   []ReportFormat{FormatJSON, FormatPDF, FormatSTIX21},
		DataDir:          "data/governance",
		AutoGenerate:     true,
	}
}

// Generator es la interfaz del Governance.
type Generator interface {
	// Generate genera un reporte para el período dado.
	Generate(ctx contextContext, period ReportPeriod, start, end time.Time) (*Report, error)

	// GenerateNow genera un reporte para el período por defecto.
	GenerateNow(ctx contextContext) (*Report, error)

	// ListReports devuelve todos los reportes generados.
	ListReports() []*Report

	// LastReport devuelve el último reporte.
	LastReport() *Report

	// GetReport recupera un reporte por ID.
	GetReport(id string) (*Report, error)

	// Shutdown detiene el generator.
	Shutdown() error
}

// contextContext es un alias para no importar context en este archivo.
type contextContext interface {
	Done() <-chan struct{}
	Err() error
}
