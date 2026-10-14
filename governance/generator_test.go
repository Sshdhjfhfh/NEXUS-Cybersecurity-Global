// Nexus IA v6.0 — Tests del Governance
package governance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeDataSource implementa DataSource para tests.
type fakeDataSource struct {
	actions        []ActionRecord
	blockedActions []BlockedAction
	rogueIncidents []RogueIncident
	capability     *CapabilitySnapshot
	rsi            *RSIEvaluation
}

func (f *fakeDataSource) Actions(start, end time.Time) []ActionRecord {
	return f.actions
}
func (f *fakeDataSource) BlockedActions(start, end time.Time) []BlockedAction {
	return f.blockedActions
}
func (f *fakeDataSource) RogueIncidents(start, end time.Time) []RogueIncident {
	return f.rogueIncidents
}
func (f *fakeDataSource) CapabilitySnapshot() *CapabilitySnapshot {
	return f.capability
}
func (f *fakeDataSource) RSIEvaluation() *RSIEvaluation {
	return f.rsi
}

// TestNewSigner verifica la creación del signer.
func TestNewSigner(t *testing.T) {
	s, err := NewSigner("test-key-1")
	if err != nil {
		t.Fatalf("NewSigner failed: %v", err)
	}
	if s.KeyID() != "test-key-1" {
		t.Errorf("wrong key ID: %s", s.KeyID())
	}
	if len(s.PublicKeyBytes()) == 0 {
		t.Error("public key is empty")
	}
}

// TestSignVerifyReport verifica firma y verificación.
func TestSignVerifyReport(t *testing.T) {
	signer, err := NewSigner("test-key-2")
	if err != nil {
		t.Fatalf("NewSigner failed: %v", err)
	}

	now := time.Now()
	r := &Report{
		ID:          "test-report-1",
		Period:      PeriodWeekly,
		PeriodStart: now.Add(-7 * 24 * time.Hour),
		PeriodEnd:   now,
		GeneratedAt: now,
		GeneratedBy: "test",
		Summary:     "Test report summary",
	}

	// Firmar
	if err := signer.SignReport(r); err != nil {
		t.Fatalf("SignReport failed: %v", err)
	}
	if r.Signature == "" {
		t.Error("signature is empty")
	}
	if r.ContentHash == "" {
		t.Error("content hash is empty")
	}
	if r.SignatureAlg != "dilithium3" {
		t.Errorf("wrong alg: %s", r.SignatureAlg)
	}

	// Verificar
	ok, err := signer.VerifyReport(r)
	if err != nil {
		t.Fatalf("VerifyReport failed: %v", err)
	}
	if !ok {
		t.Fatal("VerifyReport should return true")
	}

	// Tamper: modificar summary
	r.Summary = "Modified summary after signing"
	ok, err = signer.VerifyReport(r)
	if err == nil && ok {
		t.Fatal("VerifyReport should fail after tampering")
	}
}

// TestGenerator verifica la generación de un reporte.
func TestGenerator(t *testing.T) {
	cfg := DefaultConfig()
	signer, _ := NewSigner("test-gen")
	g := NewGenerator(cfg, signer)

	ctx := context.Background()
	now := time.Now()
	report, err := g.Generate(ctx, PeriodWeekly, now.Add(-7*24*time.Hour), now)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if report == nil {
		t.Fatal("report is nil")
	}
	if report.ID == "" {
		t.Error("report ID is empty")
	}
	if report.Signature == "" {
		t.Error("report should be signed")
	}
	if report.Summary == "" {
		t.Error("report should have summary")
	}
}

// TestGeneratorWithDataSource verifica con fuente de datos.
func TestGeneratorWithDataSource(t *testing.T) {
	cfg := DefaultConfig()
	signer, _ := NewSigner("test-gen-2")
	g := NewGenerator(cfg, signer)

	source := &fakeDataSource{
		actions: []ActionRecord{
			{Timestamp: time.Now(), AgentID: "red-01", Action: "SCAN", Target: "x", Result: "allowed"},
			{Timestamp: time.Now(), AgentID: "red-01", Action: "EXPLOIT", Target: "y", Result: "allowed"},
		},
		blockedActions: []BlockedAction{
			{Timestamp: time.Now(), AgentID: "red-01", Action: "ISOLATE_HOST", Reason: "not permitted"},
		},
		rogueIncidents: []RogueIncident{
			{Timestamp: time.Now(), AgentID: "red-02", ViolationType: "DECEPTION", Severity: "CRITICAL", ActionTaken: "stopped"},
		},
		rsi: &RSIEvaluation{
			Level:          "WARNING",
			ImprovementRate: 0.35,
			Summary:        "3 indicators accelerating",
		},
	}
	g.SetDataSource(source)

	ctx := context.Background()
	now := time.Now()
	report, err := g.Generate(ctx, PeriodWeekly, now.Add(-7*24*time.Hour), now)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if report.TotalActions != 2 {
		t.Errorf("expected 2 actions, got %d", report.TotalActions)
	}
	if report.TotalBlockedActions != 1 {
		t.Errorf("expected 1 blocked, got %d", report.TotalBlockedActions)
	}
	if report.TotalRogueIncidents != 1 {
		t.Errorf("expected 1 rogue, got %d", report.TotalRogueIncidents)
	}
	if report.OverallRiskLevel != "HIGH" {
		t.Errorf("expected HIGH risk, got %s", report.OverallRiskLevel)
	}
	if report.RSIEvaluation == nil {
		t.Error("RSI evaluation should be set")
	}
}

// TestGenerateNow verifica el período por defecto.
func TestGenerateNow(t *testing.T) {
	cfg := DefaultConfig()
	signer, _ := NewSigner("test-now")
	g := NewGenerator(cfg, signer)

	report, err := g.GenerateNow(context.Background())
	if err != nil {
		t.Fatalf("GenerateNow failed: %v", err)
	}
	if report.Period != PeriodWeekly {
		t.Errorf("expected WEEKLY, got %s", report.Period)
	}
}

// TestRenderJSON verifica la serialización JSON.
func TestRenderJSON(t *testing.T) {
	signer, _ := NewSigner("test-json")
	g := NewGenerator(DefaultConfig(), signer)

	report, _ := g.GenerateNow(context.Background())
	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON is empty")
	}
	// Debe contener el ID del reporte
	if !contains(string(data), report.ID) {
		t.Error("JSON does not contain report ID")
	}
}

// TestRenderSTIX verifica la serialización STIX 2.1.
func TestRenderSTIX(t *testing.T) {
	signer, _ := NewSigner("test-stix")
	g := NewGenerator(DefaultConfig(), signer)

	report, _ := g.GenerateNow(context.Background())
	data, err := RenderSTIX(report)
	if err != nil {
		t.Fatalf("RenderSTIX failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("STIX is empty")
	}
	s := string(data)
	if !contains(s, "bundle") {
		t.Error("STIX should contain 'bundle'")
	}
	if !contains(s, "2.1") {
		t.Error("STIX should be spec_version 2.1")
	}
	if !contains(s, "identity--nexus-ia") {
		t.Error("STIX should contain nexus-ia identity")
	}
}

// TestRenderPDF verifica el renderizado PDF-like.
func TestRenderPDF(t *testing.T) {
	signer, _ := NewSigner("test-pdf")
	g := NewGenerator(DefaultConfig(), signer)

	report, _ := g.GenerateNow(context.Background())
	data, err := RenderPDF(report)
	if err != nil {
		t.Fatalf("RenderPDF failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("PDF is empty")
	}
	s := string(data)
	if !contains(s, "NEXUS IA v6.0") {
		t.Error("PDF should have title")
	}
	if !contains(s, "dilithium3") {
		t.Error("PDF should have signature algorithm")
	}
}

// TestSaveAll verifica el guardado en 3 formatos.
func TestSaveAll(t *testing.T) {
	signer, _ := NewSigner("test-save")
	g := NewGenerator(DefaultConfig(), signer)

	report, _ := g.GenerateNow(context.Background())

	dir, err := os.MkdirTemp("", "gov-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(dir)

	if err := SaveAll(report, dir); err != nil {
		t.Fatalf("SaveAll failed: %v", err)
	}

	// Verificar que los 3 archivos existen
	if _, err := os.Stat(filepath.Join(dir, report.ID+".json")); err != nil {
		t.Errorf("JSON file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, report.ID+".stix.json")); err != nil {
		t.Errorf("STIX file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, report.ID+".report.txt")); err != nil {
		t.Errorf("PDF file missing: %v", err)
	}

	// Verificar que los paths están actualizados
	if report.JSONPath == "" {
		t.Error("JSONPath should be set")
	}
	if report.STIXPath == "" {
		t.Error("STIXPath should be set")
	}
	if report.PDFPath == "" {
		t.Error("PDFPath should be set")
	}
}

// TestListReports verifica el listado de reportes.
func TestListReports(t *testing.T) {
	signer, _ := NewSigner("test-list")
	g := NewGenerator(DefaultConfig(), signer)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, _ = g.GenerateNow(ctx)
	}

	reports := g.ListReports()
	if len(reports) != 3 {
		t.Errorf("expected 3 reports, got %d", len(reports))
	}

	last := g.LastReport()
	if last == nil {
		t.Fatal("last report is nil")
	}
}

// TestGetReport verifica recuperar por ID.
func TestGetReport(t *testing.T) {
	signer, _ := NewSigner("test-get")
	g := NewGenerator(DefaultConfig(), signer)

	ctx := context.Background()
	report, _ := g.GenerateNow(ctx)

	got, err := g.GetReport(report.ID)
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}
	if got.ID != report.ID {
		t.Errorf("wrong report: %s", got.ID)
	}

	// Buscar inexistente
	_, err = g.GetReport("ghost-report")
	if err == nil {
		t.Error("GetReport should fail for unknown ID")
	}
}

// TestRiskAssessment verifica la evaluación de riesgo.
func TestRiskAssessment(t *testing.T) {
	tests := []struct {
		report  *Report
		expected string
	}{
		{&Report{TotalRogueIncidents: 1}, "HIGH"},
		{&Report{TotalBlockedActions: 15}, "MEDIUM"},
		{&Report{TotalActions: 100}, "LOW"},
	}
	for _, tc := range tests {
		got := assessRisk(tc.report)
		if got != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, got)
		}
	}
}

// TestRecommendations verifica la generación de recomendaciones.
func TestRecommendations(t *testing.T) {
	// Con rogue incidents → HIGH
	r1 := &Report{TotalRogueIncidents: 1}
	recs := generateRecommendations(r1)
	if len(recs) == 0 {
		t.Error("should have recommendations for rogue incident")
	}
	if recs[0].Priority != "HIGH" {
		t.Errorf("expected HIGH priority, got %s", recs[0].Priority)
	}

	// Con RSI CRITICAL
	r2 := &Report{
		RSIEvaluation: &RSIEvaluation{Level: "CRITICAL"},
	}
	recs = generateRecommendations(r2)
	found := false
	for _, rec := range recs {
		if rec.Priority == "CRITICAL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("should have CRITICAL recommendation for RSI CRITICAL")
	}
}

// TestShutdown verifica el shutdown.
func TestShutdown(t *testing.T) {
	signer, _ := NewSigner("test-shutdown")
	g := NewGenerator(DefaultConfig(), signer)

	if err := g.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	ctx := context.Background()
	_, err := g.GenerateNow(ctx)
	if err == nil {
		t.Error("GenerateNow after shutdown should fail")
	}
}

// TestInvalidPeriod verifica períodos inválidos.
func TestInvalidPeriod(t *testing.T) {
	signer, _ := NewSigner("test-invalid")
	g := NewGenerator(DefaultConfig(), signer)

	ctx := context.Background()

	// Sin fechas
	_, err := g.Generate(ctx, PeriodCustom, time.Time{}, time.Time{})
	if err == nil {
		t.Error("zero period should fail")
	}

	// End before start
	now := time.Now()
	_, err = g.Generate(ctx, PeriodCustom, now, now.Add(-time.Hour))
	if err == nil {
		t.Error("end before start should fail")
	}
}

// contains comprueba si s contiene substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && indexOf(s, substr) >= 0))
}

// indexOf devuelve el índice de substr en s (-1 si no).
func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
