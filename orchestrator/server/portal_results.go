// Client-facing assessment results + the operator "release" control. Clients read
// only their own org's released results, through curated DTOs (db/portal_results.go
// enforces the field whitelist, status filter and tenant scoping).
package server

import (
	"net/http"
	"strconv"
)

// GET /api/portal/requests/{id}/results — the caller org's results for a request.
func (s *Server) portalRequestResults(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reqID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	taskID, released, exists, err := pg.ResolveClientRequest(reqID, orgID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if !exists {
		writeErr(w, 404, "solicitud no encontrada")
		return
	}
	if taskID == 0 {
		writeJSON(w, 200, map[string]any{"state": "not_started", "findings": []any{}})
		return
	}
	if !released {
		writeJSON(w, 200, map[string]any{"state": "in_progress", "released": false, "findings": []any{}})
		return
	}
	findings, err := pg.ListClientFindings(taskID)
	if err != nil {
		writeErr(w, 500, "no se pudieron obtener los resultados")
		return
	}
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
	for _, f := range findings {
		counts[f.Severity]++
	}
	writeJSON(w, 200, map[string]any{
		"state":    "released",
		"released": true,
		"findings": findings,
		"summary":  counts,
		"total":    len(findings),
	})
}

// GET /api/portal/requests/{id}/findings/{fid} — one finding's detail (released,
// org-scoped, client-visible status only).
func (s *Server) portalRequestFinding(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := s.portalAuth(w, r)
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reqID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	findingID, err := strconv.ParseInt(r.PathValue("fid"), 10, 64)
	if err != nil || findingID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	taskID, released, exists, err := pg.ResolveClientRequest(reqID, orgID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if !exists || taskID == 0 || !released {
		writeErr(w, 404, "resultado no disponible")
		return
	}
	f, err := pg.GetClientFinding(taskID, findingID)
	if err != nil {
		writeErr(w, 500, "no se pudo obtener el hallazgo")
		return
	}
	if f == nil {
		writeErr(w, 404, "hallazgo no encontrado")
		return
	}
	writeJSON(w, 200, map[string]any{"finding": f})
}

// POST /api/admin/portal/requests/{id}/release {released} — operator releases (or
// retracts) the results of a request to the client. Console-only (requireAuth).
func (s *Server) adminReleaseResults(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reqID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	var body struct {
		Released bool `json:"released"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	pr, err := pg.GetPortalRequest(reqID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if pr == nil {
		writeErr(w, 404, "solicitud no encontrada")
		return
	}
	if body.Released && pr.EngineTaskID == nil {
		writeErr(w, 400, "la solicitud no tiene una tarea vinculada todavía")
		return
	}
	if err := pg.SetRequestResultsReleased(reqID, body.Released); err != nil {
		writeErr(w, 500, "no se pudo actualizar")
		return
	}
	action := "results.retract"
	if body.Released {
		action = "results.release"
	}
	s.audit(r, &pr.OrgID, "operator", "operator", action, "request:"+strconv.FormatInt(reqID, 10), "")
	if body.Released {
		s.portalNotify.resultsReleased(pr)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "released": body.Released})
}
