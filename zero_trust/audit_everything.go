// Nexus IA v6.0 — Principio 5: Audit Everything
//
// CADA acción queda registrada con:
//   - Hash SHA-256 del payload
//   - Firma Dilithium3 (post-cuántica)
//   - Timestamp firmado
//   - Actor + target + decisión
//
// El audit trail es INMUTABLE. Se escribe una vez y no se puede modificar.
// Esto permite publicar investigación verificable y reproducible.
package zero_trust

import (
	"context"
	"time"
)

// AuditSink es cualquier destino que reciba entradas de audit.
// Se inyecta desde nhi_manager (que implementa el audit trail firmado).
type AuditSink interface {
	// RecordAudit registra una acción en el audit trail.
	// El sink se encarga de firmar con Dilithium3 y persistir.
	RecordAudit(ctx context.Context, entry *AuditRecord) error
}

// AuditRecord es una entrada de audit en el framework Zero-Trust.
// Se convierte a nhi_manager.AuditEntry en el sink real.
type AuditRecord struct {
	// Identidad del actor
	ActorID   string
	ActorDID  string
	AgentType string

	// Acción
	Action string
	Target string
	Result string // "allow", "deny", "ask", "quarantine"
	Reason string

	// Contexto
	SessionID string
	TraceID   string
	Timestamp time.Time

	// Decisión Zero-Trust
	Decisions []DecisionRecord
}

// DecisionRecord es la decisión individual de una regla.
type DecisionRecord struct {
	Rule      string
	Principle Principle
	Decision  Decision
	Reason    string
}

// AuditEverythingRule implementa el principio 5.
type AuditEverythingRule struct {
	// Sink es el destino del audit. Se inyecta desde nhi_manager.
	Sink AuditSink

	// AsyncAudit si es true, el audit se hace en una goroutine.
	// Más rápido, pero puede perder entradas si el proceso muere.
	AsyncAudit bool

	// AuditBufferSize es el tamaño del buffer para audit asíncrono.
	AuditBufferSize int
}

// NewAuditEverythingRule crea la regla con un sink dado.
func NewAuditEverythingRule(sink AuditSink) *AuditEverythingRule {
	return &AuditEverythingRule{
		Sink:            sink,
		AsyncAudit:      false,
		AuditBufferSize: 1000,
	}
}

// Name devuelve el nombre de la regla.
func (r *AuditEverythingRule) Name() string {
	return "audit-everything"
}

// Principle devuelve el principio MAESTRO.
func (r *AuditEverythingRule) Principle() Principle {
	return PrincipleAuditEverything
}

// Evaluate NUNCA deniega; siempre permite. Su trabajo es AUDITAR.
// El audit real se hace en RecordEvaluation, que el framework llama
// después de consolidar todas las decisiones.
func (r *AuditEverythingRule) Evaluate(req *Request) Result {
	return Result{
		Decision:  Allow,
		Principle: r.Principle(),
		Reason:    "audit-everything never denies (observability principle)",
		Timestamp: nowOrZero(req),
	}
}

// RecordEvaluation graba una evaluación completa en el audit trail.
// Se llama desde el framework tras evaluar todas las reglas.
func (r *AuditEverythingRule) RecordEvaluation(ctx context.Context, req *Request, result *Result, decisions []DecisionRecord) error {
	if r.Sink == nil {
		return nil
	}

	record := &AuditRecord{
		ActorID:   req.Action.ActorID,
		ActorDID:  req.Action.ActorDID,
		AgentType: req.Action.AgentType,
		Action:    req.Action.Type,
		Target:    req.Action.Target,
		Result:    result.Decision.String(),
		Reason:    result.Reason,
		SessionID: req.Action.SessionID,
		TraceID:   req.Action.TraceID,
		Timestamp: req.Action.Timestamp,
		Decisions: decisions,
	}

	return r.Sink.RecordAudit(ctx, record)
}
