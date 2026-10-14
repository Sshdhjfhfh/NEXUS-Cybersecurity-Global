// Nexus IA v6.0 — Renderizadores de reportes (JSON / STIX 2.1 / PDF)
//
// Convierte un Report a los 3 formatos requeridos por el PDF:
//   - JSON: para auditoría programática
//   - STIX 2.1: para interoperabilidad con threat intelligence
//   - PDF: para revisión humana
//
// El PDF se genera con texto plano por ahora (sin librerías externas).
package governance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// JSON
// ═══════════════════════════════════════════════════════════════

// RenderJSON serializa un reporte a JSON.
func RenderJSON(r *Report) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil report")
	}
	return json.MarshalIndent(r, "", "  ")
}

// SaveJSON guarda el reporte a un archivo JSON.
func SaveJSON(r *Report, dir string) (string, error) {
	data, err := RenderJSON(r)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	path := filepath.Join(dir, r.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write json: %w", err)
	}
	return path, nil
}

// ═══════════════════════════════════════════════════════════════
// STIX 2.1
// ═══════════════════════════════════════════════════════════════

// STIXBundle es un bundle STIX 2.1.
type STIXBundle struct {
	Type    string                   `json:"type"`
	ID      string                   `json:"id"`
	Objects []map[string]interface{} `json:"objects"`
}

// RenderSTIX renderiza un reporte en formato STIX 2.1.
func RenderSTIX(r *Report) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil report")
	}

	bundle := STIXBundle{
		Type:    "bundle",
		ID:      "bundle--" + r.ID,
		Objects: []map[string]interface{}{},
	}

	// Identity: Nexus IA
	bundle.Objects = append(bundle.Objects, map[string]interface{}{
		"type":         "identity",
		"spec_version": "2.1",
		"id":           "identity--nexus-ia",
		"created":      r.GeneratedAt.Format(time.RFC3339),
		"modified":     r.GeneratedAt.Format(time.RFC3339),
		"name":         "Nexus IA v6.0",
		"identity_class": "system",
	})

	// Report STIX
	reportID := "report--" + r.ID
	reportObj := map[string]interface{}{
		"type":         "report",
		"spec_version": "2.1",
		"id":           reportID,
		"created":      r.GeneratedAt.Format(time.RFC3339),
		"modified":     r.GeneratedAt.Format(time.RFC3339),
		"name":         fmt.Sprintf("Nexus IA Governance Report %s", r.ID),
		"description":  r.Summary,
		"published":    r.GeneratedAt.Format(time.RFC3339),
		"report_types": []string{"governance", "audit"},
		"object_refs":  []string{"identity--nexus-ia"},
	}

	// Añadir acciones como observables
	for i, action := range r.Actions {
		actionID := fmt.Sprintf("observed-data--%s-action-%d", r.ID, i)
		bundle.Objects = append(bundle.Objects, map[string]interface{}{
			"type":         "observed-data",
			"spec_version": "2.1",
			"id":           actionID,
			"created":      action.Timestamp.Format(time.RFC3339),
			"modified":     action.Timestamp.Format(time.RFC3339),
			"first_observed":   action.Timestamp.Format(time.RFC3339),
			"last_observed":    action.Timestamp.Format(time.RFC3339),
			"number_observed":  1,
			"objects": map[string]interface{}{
				"0": map[string]interface{}{
					"type":  "x-nexus-action",
					"agent": action.AgentID,
					"action": action.Action,
					"target": action.Target,
					"result": action.Result,
					"reason": action.Reason,
				},
			},
		})
		reportObj["object_refs"] = append(reportObj["object_refs"].([]string), actionID)
	}

	// Rogue incidents como indicators
	for i, inc := range r.RogueIncidents {
		indicatorID := fmt.Sprintf("indicator--%s-rogue-%d", r.ID, i)
		bundle.Objects = append(bundle.Objects, map[string]interface{}{
			"type":         "indicator",
			"spec_version": "2.1",
			"id":           indicatorID,
			"created":      inc.Timestamp.Format(time.RFC3339),
			"modified":     inc.Timestamp.Format(time.RFC3339),
			"name":         fmt.Sprintf("Rogue behavior: %s", inc.ViolationType),
			"description":  inc.Evidence,
			"pattern_type": "nexus-behavior",
			"pattern":      fmt.Sprintf("[agent:id = '%s'] AND [violation:type = '%s']", inc.AgentID, inc.ViolationType),
			"valid_from":   inc.Timestamp.Format(time.RFC3339),
			"labels":       []string{"rogue-agent", string(inc.Severity)},
		})
		reportObj["object_refs"] = append(reportObj["object_refs"].([]string), indicatorID)
	}

	// Añadir el reporte al final
	bundle.Objects = append(bundle.Objects, reportObj)

	return json.MarshalIndent(bundle, "", "  ")
}

// SaveSTIX guarda el reporte a un archivo STIX 2.1.
func SaveSTIX(r *Report, dir string) (string, error) {
	data, err := RenderSTIX(r)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	path := filepath.Join(dir, r.ID+".stix.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write stix: %w", err)
	}
	return path, nil
}

// ═══════════════════════════════════════════════════════════════
// PDF (texto plano por ahora — se puede mejorar con LaTeX/gofpdf)
// ═══════════════════════════════════════════════════════════════

// RenderPDF genera un reporte en formato PDF-like (texto plano).
// En una implementación futura, se puede usar una librería PDF real.
func RenderPDF(r *Report) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("nil report")
	}

	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString("  NEXUS IA v6.0 — GOVERNANCE REPORT\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	sb.WriteString(fmt.Sprintf("Report ID:    %s\n", r.ID))
	sb.WriteString(fmt.Sprintf("Period:       %s\n", r.Period))
	sb.WriteString(fmt.Sprintf("From:         %s\n", r.PeriodStart.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("To:           %s\n", r.PeriodEnd.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", r.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Generated by: %s\n", r.GeneratedBy))
	sb.WriteString(fmt.Sprintf("Risk Level:   %s\n\n", r.OverallRiskLevel))

	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("  SUMMARY\n")
	sb.WriteString("───────────────────────────────────────────────────────────────\n\n")
	sb.WriteString(r.Summary + "\n\n")

	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("  STATISTICS\n")
	sb.WriteString("───────────────────────────────────────────────────────────────\n\n")
	sb.WriteString(fmt.Sprintf("  Total actions:         %d\n", r.TotalActions))
	sb.WriteString(fmt.Sprintf("  Blocked actions:       %d\n", r.TotalBlockedActions))
	sb.WriteString(fmt.Sprintf("  Rogue incidents:       %d\n", r.TotalRogueIncidents))
	sb.WriteString(fmt.Sprintf("  Recommendations:       %d\n\n", len(r.Recommendations)))

	// RSI Evaluation
	if r.RSIEvaluation != nil {
		sb.WriteString("───────────────────────────────────────────────────────────────\n")
		sb.WriteString("  RSI EVALUATION\n")
		sb.WriteString("───────────────────────────────────────────────────────────────\n\n")
		sb.WriteString(fmt.Sprintf("  Level:             %s\n", r.RSIEvaluation.Level))
		sb.WriteString(fmt.Sprintf("  Improvement rate:  %.4f\n", r.RSIEvaluation.ImprovementRate))
		sb.WriteString(fmt.Sprintf("  Code generations:  %.4f\n", r.RSIEvaluation.CodeGenerations))
		sb.WriteString(fmt.Sprintf("  Technique novelty: %.4f\n", r.RSIEvaluation.TechniqueNovelty))
		sb.WriteString(fmt.Sprintf("  Learning velocity: %.4f\n", r.RSIEvaluation.LearningVelocity))
		sb.WriteString(fmt.Sprintf("\n  %s\n", r.RSIEvaluation.Summary))
		sb.WriteString(fmt.Sprintf("\n  RECOMMENDATION: %s\n\n", r.RSIEvaluation.Recommendation))
	}

	// Rogue Incidents
	if len(r.RogueIncidents) > 0 {
		sb.WriteString("───────────────────────────────────────────────────────────────\n")
		sb.WriteString("  ROGUE INCIDENTS\n")
		sb.WriteString("───────────────────────────────────────────────────────────────\n\n")
		for i, inc := range r.RogueIncidents {
			sb.WriteString(fmt.Sprintf("  [%d] %s | %s | %s\n",
				i+1, inc.Timestamp.Format(time.RFC3339), inc.AgentID, inc.ViolationType))
			sb.WriteString(fmt.Sprintf("      Severity: %s | Action: %s\n", inc.Severity, inc.ActionTaken))
			sb.WriteString(fmt.Sprintf("      Evidence: %s\n\n", inc.Evidence))
		}
	}

	// Recommendations
	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("  RECOMMENDATIONS\n")
	sb.WriteString("───────────────────────────────────────────────────────────────\n\n")
	for i, rec := range r.Recommendations {
		sb.WriteString(fmt.Sprintf("  [%d] [%s] %s\n", i+1, rec.Priority, rec.Title))
		sb.WriteString(fmt.Sprintf("      Category: %s\n", rec.Category))
		sb.WriteString(fmt.Sprintf("      %s\n", rec.Description))
		sb.WriteString(fmt.Sprintf("      Rationale: %s\n\n", rec.Rationale))
	}

	// Signature
	sb.WriteString("───────────────────────────────────────────────────────────────\n")
	sb.WriteString("  CRYPTOGRAPHIC SIGNATURE\n")
	sb.WriteString("───────────────────────────────────────────────────────────────\n\n")
	sb.WriteString(fmt.Sprintf("  Algorithm:  %s\n", r.SignatureAlg))
	sb.WriteString(fmt.Sprintf("  Key ID:     %s\n", r.SigningKeyID))
	sb.WriteString(fmt.Sprintf("  Content hash (SHA-256):\n    %s\n\n", r.ContentHash))
	sb.WriteString(fmt.Sprintf("  Dilithium3 signature:\n    %s\n\n", r.Signature))

	sb.WriteString("═══════════════════════════════════════════════════════════════\n")
	sb.WriteString("  END OF REPORT\n")
	sb.WriteString("═══════════════════════════════════════════════════════════════\n")

	return []byte(sb.String()), nil
}

// SavePDF guarda el reporte a un archivo PDF-like (texto plano).
func SavePDF(r *Report, dir string) (string, error) {
	data, err := RenderPDF(r)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	path := filepath.Join(dir, r.ID+".report.txt")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// ═══════════════════════════════════════════════════════════════
// SaveAll: guarda todos los formatos
// ═══════════════════════════════════════════════════════════════

// SaveAll guarda el reporte en todos los formatos configurados.
// Actualiza los campos JSONPath, PDFPath, STIXPath del reporte.
func SaveAll(r *Report, dir string) error {
	jsonPath, err := SaveJSON(r, dir)
	if err != nil {
		return fmt.Errorf("save json: %w", err)
	}
	r.JSONPath = jsonPath

	stixPath, err := SaveSTIX(r, dir)
	if err != nil {
		return fmt.Errorf("save stix: %w", err)
	}
	r.STIXPath = stixPath

	pdfPath, err := SavePDF(r, dir)
	if err != nil {
		return fmt.Errorf("save pdf: %w", err)
	}
	r.PDFPath = pdfPath

	return nil
}
