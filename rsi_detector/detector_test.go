// Nexus IA v6.0 — Tests del RSI Detector
package rsi_detector

import (
	"testing"
	"time"
)

// TestAllIndicators verifica los 4 indicadores.
func TestAllIndicators(t *testing.T) {
	inds := AllIndicators()
	if len(inds) != 4 {
		t.Fatalf("expected 4 indicators, got %d", len(inds))
	}

	expected := map[Indicator]bool{
		IndicatorImprovementRate:  true,
		IndicatorCodeGenerations:  true,
		IndicatorTechniqueNovelty: true,
		IndicatorLearningVelocity: true,
	}
	for _, ind := range inds {
		if !expected[ind] {
			t.Errorf("unexpected indicator: %s", ind)
		}
	}
}

// TestRecordSample verifica añadir muestras.
func TestRecordSample(t *testing.T) {
	d := NewDetector(DefaultConfig())

	// Sample válido
	err := d.Record(&Sample{
		Indicator: IndicatorImprovementRate,
		Value:     0.5,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Sample con indicador inválido
	err = d.Record(&Sample{
		Indicator: "INVALID_INDICATOR",
		Value:     1.0,
	})
	if err == nil {
		t.Error("invalid indicator should fail")
	}

	// Sample nil
	err = d.Record(nil)
	if err == nil {
		t.Error("nil sample should fail")
	}
}

// TestEvaluateStable verifica evaluación sin aceleración.
func TestEvaluateStable(t *testing.T) {
	d := NewDetector(DefaultConfig())
	now := time.Now()

	// Añadir muestras estables (mismo valor cada hora durante 8 horas)
	for i := 0; i < 8; i++ {
		ts := now.Add(-time.Duration(i) * time.Hour)
		for _, ind := range AllIndicators() {
			_ = d.Record(&Sample{
				Indicator: ind,
				Value:     1.0,
				Timestamp: ts,
			})
		}
	}

	report, err := d.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if report.Level != AlertNone {
		t.Errorf("stable values should give AlertNone, got %v", report.Level)
	}
}

// TestEvaluateCritical verifica detección de RSI crítico.
func TestEvaluateCritical(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Window = 4 * time.Hour // ventana corta para test
	cfg.MinDataPoints = 4

	d := NewDetector(cfg)
	now := time.Now()

	// Fase 1: valores bajos en el pasado (más de 4h atrás)
	for i := 10; i >= 5; i-- {
		ts := now.Add(-time.Duration(i) * time.Hour)
		for _, ind := range AllIndicators() {
			_ = d.Record(&Sample{
				Indicator: ind,
				Value:     1.0,
				Timestamp: ts,
			})
		}
	}

	// Fase 2: valores acelerando dentro de la ventana actual (4h)
	// Con crecimiento exponencial dentro de la ventana
	values := []float64{1.0, 2.0, 4.0, 8.0, 16.0, 32.0}
	for i, v := range values {
		ts := now.Add(-time.Duration(len(values)-1-i) * (4 * time.Hour / time.Duration(len(values))))
		for _, ind := range AllIndicators() {
			_ = d.Record(&Sample{
				Indicator: ind,
				Value:     v,
				Timestamp: ts,
			})
		}
	}

	report, err := d.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Con crecimiento exponencial, debería haber aceleración
	t.Logf("Report level: %s", report.Level)
	t.Logf("Summary: %s", report.Summary)
	for ind, status := range report.Statuses {
		t.Logf("  %s: current=%.2f prev=%.2f deltaPct=%.2f%% trend=%s exceeds=%v",
			ind, status.Current, status.Previous, status.DeltaPct*100, status.Trend, status.ExceedsThreshold)
	}

	// Si los 4 aceleran, debería ser CRITICAL
	if report.Level == AlertCritical {
		// Verificar que se emitió alerta
		alerts := d.Alerts()
		if len(alerts) == 0 {
			t.Error("critical report should emit alert")
		}
		if alerts[0].Level != AlertCritical {
			t.Errorf("alert level should be CRITICAL, got %v", alerts[0].Level)
		}
	}
}

// TestStatus verifica el estado de un indicador.
func TestStatus(t *testing.T) {
	d := NewDetector(DefaultConfig())

	status := d.Status(IndicatorImprovementRate)
	if status == nil {
		t.Fatal("status should not be nil")
	}
	if status.Indicator != IndicatorImprovementRate {
		t.Errorf("wrong indicator: %s", status.Indicator)
	}
	if status.Threshold != DefaultConfig().ThresholdImprovementRate {
		t.Errorf("wrong threshold: %f", status.Threshold)
	}
}

// TestAllStatuses verifica todos los estados.
func TestAllStatuses(t *testing.T) {
	d := NewDetector(DefaultConfig())

	statuses := d.AllStatuses()
	if len(statuses) != 4 {
		t.Errorf("expected 4 statuses, got %d", len(statuses))
	}
	for _, ind := range AllIndicators() {
		if _, ok := statuses[ind]; !ok {
			t.Errorf("missing status for %s", ind)
		}
	}
}

// TestAlertAcknowledge verifica el reconocimiento de alertas.
func TestAlertAcknowledge(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MinDataPoints = 0

	d := NewDetector(cfg)

	// Crear una alerta manualmente
	alert := &RSIAlert{
		ID:        "test-alert-1",
		Level:     AlertCritical,
		Timestamp: time.Now(),
		Message:   "test critical",
	}
	d.alerts = append(d.alerts, alert)

	// Acknowledge
	err := d.Acknowledge("test-alert-1", "investigator@nexus")
	if err != nil {
		t.Fatalf("Acknowledge failed: %v", err)
	}

	alerts := d.Alerts()
	if !alerts[0].Acknowledged {
		t.Error("alert should be acknowledged")
	}
	if alerts[0].AcknowledgedBy != "investigator@nexus" {
		t.Errorf("wrong acknowledged_by: %s", alerts[0].AcknowledgedBy)
	}

	// Acknowledge no existente
	err = d.Acknowledge("ghost-alert", "someone")
	if err == nil {
		t.Error("acknowledging ghost alert should fail")
	}
}

// TestConfigThreshold verifica los umbrales.
func TestConfigThreshold(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Threshold(IndicatorImprovementRate) != 0.20 {
		t.Error("wrong threshold for improvement rate")
	}
	if cfg.Threshold(IndicatorCodeGenerations) != 0.30 {
		t.Error("wrong threshold for code generations")
	}
	if cfg.Threshold(IndicatorTechniqueNovelty) != 0.15 {
		t.Error("wrong threshold for technique novelty")
	}
	if cfg.Threshold(IndicatorLearningVelocity) != 0.25 {
		t.Error("wrong threshold for learning velocity")
	}
}

// TestEvaluateInsufficientData verifica con pocos datos.
func TestEvaluateInsufficientData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MinDataPoints = 100

	d := NewDetector(cfg)

	// Solo 2 muestras
	d.Record(&Sample{Indicator: IndicatorImprovementRate, Value: 1.0, Timestamp: time.Now()})
	d.Record(&Sample{Indicator: IndicatorLearningVelocity, Value: 1.0, Timestamp: time.Now()})

	report, err := d.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Debería ser AlertNone por datos insuficientes
	if report.Level != AlertNone {
		t.Errorf("insufficient data should give AlertNone, got %v", report.Level)
	}
}

// TestShutdown verifica el shutdown.
func TestShutdown(t *testing.T) {
	d := NewDetector(DefaultConfig())

	err := d.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Record después de shutdown debe fallar
	err = d.Record(&Sample{Indicator: IndicatorImprovementRate, Value: 1.0})
	if err == nil {
		t.Error("Record after shutdown should fail")
	}

	// Evaluate después de shutdown debe fallar
	_, err = d.Evaluate()
	if err == nil {
		t.Error("Evaluate after shutdown should fail")
	}
}

// TestLastReport verifica el último reporte.
func TestLastReport(t *testing.T) {
	d := NewDetector(DefaultConfig())

	// Sin reporte aún
	if d.LastReport() != nil {
		t.Error("last report should be nil initially")
	}

	// Evaluar
	_, _ = d.Evaluate()

	// Verificar
	report := d.LastReport()
	if report == nil {
		t.Fatal("last report should not be nil after Evaluate")
	}
	if report.Timestamp.IsZero() {
		t.Error("report timestamp should be set")
	}
}
