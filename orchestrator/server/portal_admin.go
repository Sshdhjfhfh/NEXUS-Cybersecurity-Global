// Operator-facing bridge between the client portal and the assessment engine.
// These routes live under /api/admin/ (NOT /api/portal/), so requireAuth gates
// them to the single console operator — a self-service portal client can never
// reach them. Promotion is deliberately human-in-the-loop: a client request only
// becomes an engine task when an operator reviews it and promotes it, and the
// task is created WITHOUT auto-launching the autonomous engine, so the operator
// still confirms scope/authorization and starts the run with the usual controls.
package server

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/nexus-ia/nexus/db"
)

// taskCreateOptionsForPortal builds minimal options for a task promoted from a
// portal request: a readable name, no LLM profile and no auto-launch, so the
// operator configures and starts it with the normal console controls.
func taskCreateOptionsForPortal(orgName string) db.TaskCreateOptions {
	name := "Portal: " + orgName
	if len(name) > 120 {
		name = name[:120]
	}
	return db.TaskCreateOptions{Name: name}
}

// validRequestStatus is the lifecycle the portal understands.
func validRequestStatus(s string) bool {
	switch s {
	case "new", "scheduled", "in_progress", "done":
		return true
	}
	return false
}

// classifyScopeToken best-effort maps a free-text scope token to an engine scope
// row. Only clearly-parseable domains and IPs are classified; anything else is
// left for the operator to add manually (we never guess a target into scope).
func classifyScopeToken(tok string) (kind, value string, ok bool) {
	tok = strings.TrimSpace(strings.Trim(tok, ",;"))
	if tok == "" {
		return "", "", false
	}
	_, host, _, parsed := parseTarget(tok)
	if !parsed || host == "" {
		return "", "", false
	}
	if net.ParseIP(host) != nil {
		return "ip", host, true
	}
	return "root_domain", host, true
}

// GET /api/admin/portal/requests — all client requests across orgs (operator).
func (s *Server) adminListPortalRequests(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	list, err := pg.AdminListPortalRequests()
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las solicitudes")
		return
	}
	writeJSON(w, 200, map[string]any{"requests": list})
}

// POST /api/admin/portal/requests/{id}/promote — create an engine task from an
// approved request and link them. Does NOT launch the autonomous run.
func (s *Server) adminPromoteRequest(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	reqID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reqID <= 0 {
		writeErr(w, 400, "id inválido")
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
	if pr.EngineTaskID != nil {
		writeErr(w, 409, "esta solicitud ya está vinculada a una tarea")
		return
	}

	// Build a descriptive task. Scope text is preserved in the goal so the
	// operator sees exactly what the client authorized before starting the run.
	desc := pr.Title + " — " + pr.OrgName
	goal := pr.Title
	if strings.TrimSpace(pr.Scope) != "" {
		goal += "\n\nAlcance declarado por el cliente:\n" + pr.Scope
	}
	t, err := s.m.CreateTaskWithOptions(desc, goal, taskCreateOptionsForPortal(pr.OrgName))
	if err != nil {
		writeErr(w, 500, "no se pudo crear la tarea: "+err.Error())
		return
	}
	taskID, _ := strconv.ParseInt(t.ID, 10, 64)

	// Best-effort: seed the engine scope with clearly-parseable targets so the
	// operator starts from the client's declared scope. Failures are ignored —
	// the operator confirms/edits scope before launching.
	added := 0
	if as := s.m.Assets(); as != nil {
		// source "manual" (the operator seeded it); the reason records that it came
		// from this portal request. source is CHECK-constrained, so stay in the set.
		reason := "portal request #" + strconv.FormatInt(reqID, 10)
		for _, tok := range strings.Fields(strings.ReplaceAll(pr.Scope, ",", " ")) {
			if kind, value, ok := classifyScopeToken(tok); ok {
				if _, err := as.AddAgentScope(taskID, kind, value, reason, "manual"); err == nil {
					added++
				}
			}
		}
	}

	if err := pg.LinkRequestToTask(reqID, taskID); err != nil {
		writeErr(w, 500, "tarea creada pero no se pudo vincular la solicitud")
		return
	}
	writeJSON(w, 201, map[string]any{
		"ok":          true,
		"task_id":     t.ID,
		"scope_added": added,
		"status":      "scheduled",
	})
}

// GET /api/admin/portal/orgs — all organizations with member/request counts.
func (s *Server) adminListOrgs(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	orgs, err := pg.AdminListOrgs()
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las organizaciones")
		return
	}
	writeJSON(w, 200, map[string]any{"orgs": orgs})
}

// GET /api/admin/portal/orgs/{id}/members — members of one organization.
func (s *Server) adminListOrgMembers(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	orgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orgID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	members, err := pg.ListMembers(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar los miembros")
		return
	}
	writeJSON(w, 200, map[string]any{"members": members})
}

// PATCH /api/admin/portal/orgs/{id} {suspended} — suspend or reactivate an org.
// A suspended org is locked out of the portal (login and every authenticated
// call) until reactivated.
func (s *Server) adminSuspendOrg(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	orgID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orgID <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	var body struct {
		Suspended bool `json:"suspended"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if err := pg.SetOrgSuspended(orgID, body.Suspended); err != nil {
		writeErr(w, 500, "no se pudo actualizar la organización")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "suspended": body.Suspended})
}

// PATCH /api/admin/portal/requests/{id} {status} — operator sets the status the
// client sees (scheduled / in_progress / done).
func (s *Server) adminUpdateRequestStatus(w http.ResponseWriter, r *http.Request) {
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
		Status string `json:"status"`
	}
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	body.Status = strings.TrimSpace(body.Status)
	if !validRequestStatus(body.Status) {
		writeErr(w, 400, "estado inválido")
		return
	}
	if err := pg.UpdatePortalRequestStatus(reqID, body.Status); err != nil {
		writeErr(w, 500, "no se pudo actualizar el estado")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
