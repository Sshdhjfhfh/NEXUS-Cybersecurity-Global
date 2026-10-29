// Client remediation workspace (see db/portal_remediation.go for the model).
// Who may do what — enforced here, server-side, from the role resolved per request:
//   - any member: read, comment, report progress (open / in progress), request a
//     NEXUS verification (retest);
//   - admin/owner: assign owners, set target dates, accept or revoke a risk.
//
// A finding is never marked fixed from the portal: only a NEXUS retest does that.
package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nexus-ia/nexus/db"
)

const (
	maxRemediationComment = 2000
	maxRetestNote         = 1000
	minRiskReason         = 10
	maxRiskReason         = 2000
	maxRiskAcceptDays     = 365
)

// remediationTarget authenticates the caller, resolves their live role and loads
// the finding's remediation item scoped to their org (404 when not theirs).
func (s *Server) remediationTarget(w http.ResponseWriter, r *http.Request) (pg *db.DB, orgID, userID int64, role, actor string, it *db.RemediationItem, ok bool) {
	orgID, userID, role, ok = s.portalRequireRole(w, r, "member")
	if !ok {
		return
	}
	ok = false
	pg = s.portalDB(w)
	if pg == nil {
		return
	}
	fid, err := strconv.ParseInt(r.PathValue("fid"), 10, 64)
	if err != nil || fid <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	it, err = pg.GetRemediation(orgID, fid, time.Now())
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if it == nil {
		writeErr(w, 404, "hallazgo no encontrado")
		return
	}
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		actor = u.Email
	}
	return pg, orgID, userID, role, actor, it, true
}

func (s *Server) remediationChanged(r *http.Request, pg *db.DB, orgID, fid int64, actor, kind, detail, auditAction string) {
	// The change itself is already saved (and audited below); a failed timeline
	// insert must not turn it into an error for the client.
	_ = pg.AddRemediationEvent(orgID, fid, actor, kind, detail)
	if auditAction != "" {
		s.audit(r, &orgID, actor, "portal_user", auditAction, "finding:"+strconv.FormatInt(fid, 10), detail)
	}
}

// GET /api/portal/remediation — every released finding with its remediation state.
func (s *Server) portalListRemediation(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "member")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	items, err := pg.ListRemediation(orgID, time.Now())
	if err != nil {
		writeErr(w, 500, "no se pudo cargar la remediación")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "sla_days": db.RemediationSLADays})
}

// GET /api/portal/remediation/{fid} — one item with its report and timeline.
func (s *Server) portalGetRemediation(w http.ResponseWriter, r *http.Request) {
	pg, orgID, _, _, _, it, ok := s.remediationTarget(w, r)
	if !ok {
		return
	}
	report, err := pg.GetRemediationReport(it.FindingID)
	if err != nil {
		writeErr(w, 500, "no se pudo obtener el hallazgo")
		return
	}
	events, err := pg.ListRemediationEvents(orgID, it.FindingID)
	if err != nil {
		writeErr(w, 500, "no se pudo obtener el hallazgo")
		return
	}
	writeJSON(w, 200, map[string]any{"item": it, "report": report, "events": events})
}

// PATCH /api/portal/remediation/{fid} {assignee_id?, due_date?, state?}
// assignee_id / due_date accept null to clear (due falls back to the SLA).
func (s *Server) portalUpdateRemediation(w http.ResponseWriter, r *http.Request) {
	pg, orgID, _, role, actor, it, ok := s.remediationTarget(w, r)
	if !ok {
		return
	}
	var req struct {
		AssigneeID json.RawMessage `json:"assignee_id"`
		DueDate    json.RawMessage `json:"due_date"`
		State      *string         `json:"state"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	isAdmin := roleRank(role) >= roleRank("admin")
	if (req.AssigneeID != nil || req.DueDate != nil) && !isAdmin {
		writeErr(w, 403, "no tienes permisos para esta acción")
		return
	}
	if it.State == db.RemVerifiedFixed {
		writeErr(w, 409, "el hallazgo ya está corregido y verificado")
		return
	}

	// Validate everything before writing anything.
	var assignee *int64
	if req.AssigneeID != nil && string(req.AssigneeID) != "null" {
		var id int64
		if err := json.Unmarshal(req.AssigneeID, &id); err != nil || id <= 0 {
			writeErr(w, 400, "responsable inválido")
			return
		}
		member, err := pg.ActiveOrgMember(orgID, id)
		if err != nil {
			writeErr(w, 503, "almacenamiento no disponible")
			return
		}
		if !member {
			writeErr(w, 400, "el responsable debe ser un miembro activo de la organización")
			return
		}
		assignee = &id
	}
	var due *time.Time
	if req.DueDate != nil && string(req.DueDate) != "null" {
		var ds string
		if err := json.Unmarshal(req.DueDate, &ds); err != nil {
			writeErr(w, 400, "fecha inválida")
			return
		}
		d, err := time.Parse("2006-01-02", ds)
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if err != nil || d.Before(today.AddDate(-1, 0, 0)) || d.After(today.AddDate(2, 0, 0)) {
			writeErr(w, 400, "fecha inválida")
			return
		}
		due = &d
	}
	if req.State != nil {
		if *req.State != "open" && *req.State != "in_progress" {
			writeErr(w, 400, "estado inválido")
			return
		}
		// Moving a client-accepted risk back to work revokes the acceptance, which
		// is an admin decision.
		if it.State == db.RemRiskAccepted && !isAdmin {
			writeErr(w, 403, "solo un administrador puede revocar una aceptación de riesgo")
			return
		}
	}

	fid := it.FindingID
	if req.AssigneeID != nil {
		if err := pg.SetRemediationAssignee(orgID, fid, assignee); err != nil {
			writeErr(w, 500, "no se pudo actualizar")
			return
		}
		detail := ""
		if assignee != nil {
			if u, err := pg.GetPortalUserByID(*assignee); err == nil && u != nil {
				detail = u.Email
			}
		}
		kind := "assigned"
		if assignee == nil {
			kind = "unassigned"
		}
		s.remediationChanged(r, pg, orgID, fid, actor, kind, detail, "remediation.assign")
	}
	if req.DueDate != nil {
		if err := pg.SetRemediationDueDate(orgID, fid, due); err != nil {
			writeErr(w, 500, "no se pudo actualizar")
			return
		}
		detail, kind := "", "due_cleared"
		if due != nil {
			detail, kind = due.Format("2006-01-02"), "due_set"
		}
		s.remediationChanged(r, pg, orgID, fid, actor, kind, detail, "remediation.due")
	}
	if req.State != nil {
		if err := pg.SetRemediationWorkState(orgID, fid, *req.State); err != nil {
			writeErr(w, 500, "no se pudo actualizar")
			return
		}
		s.remediationChanged(r, pg, orgID, fid, actor, "state", *req.State, "remediation.state")
	}
	updated, _ := pg.GetRemediation(orgID, fid, time.Now())
	writeJSON(w, 200, map[string]any{"ok": true, "item": updated})
}

// POST /api/portal/remediation/{fid}/comments {text}
func (s *Server) portalCommentRemediation(w http.ResponseWriter, r *http.Request) {
	pg, orgID, _, _, actor, it, ok := s.remediationTarget(w, r)
	if !ok {
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" || utf8.RuneCountInString(req.Text) > maxRemediationComment {
		writeErr(w, 400, "el comentario debe tener entre 1 y 2000 caracteres")
		return
	}
	if err := pg.AddRemediationEvent(orgID, it.FindingID, actor, "comment", req.Text); err != nil {
		writeErr(w, 500, "no se pudo guardar el comentario")
		return
	}
	writeJSON(w, 201, map[string]any{"ok": true})
}

// POST /api/portal/remediation/{fid}/retest {note} — ask NEXUS to verify the fix.
func (s *Server) portalRequestRetest(w http.ResponseWriter, r *http.Request) {
	pg, orgID, userID, _, actor, it, ok := s.remediationTarget(w, r)
	if !ok {
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	if utf8.RuneCountInString(req.Note) > maxRetestNote {
		writeErr(w, 400, "la nota debe tener como máximo 1000 caracteres")
		return
	}
	switch it.State {
	case db.RemVerifiedFixed:
		writeErr(w, 409, "el hallazgo ya está corregido y verificado")
		return
	case db.RemRetestRequested, db.RemRetestRunning:
		writeErr(w, 409, "ya hay una verificación pendiente para este hallazgo")
		return
	case db.RemRiskAccepted:
		writeErr(w, 409, "el riesgo está aceptado; revócalo antes de solicitar una verificación")
		return
	}
	if err := pg.RequestRemediationRetest(orgID, it.FindingID, userID, req.Note); err != nil {
		writeErr(w, 500, "no se pudo solicitar la verificación")
		return
	}
	s.remediationChanged(r, pg, orgID, it.FindingID, actor, "retest_requested", req.Note, "remediation.retest_request")
	updated, _ := pg.GetRemediation(orgID, it.FindingID, time.Now())
	writeJSON(w, 200, map[string]any{"ok": true, "item": updated})
}

// POST /api/portal/remediation/{fid}/accept-risk {reason, expires_at} (admin+)
func (s *Server) portalAcceptRisk(w http.ResponseWriter, r *http.Request) {
	pg, orgID, userID, role, actor, it, ok := s.remediationTarget(w, r)
	if !ok {
		return
	}
	if roleRank(role) < roleRank("admin") {
		writeErr(w, 403, "no tienes permisos para esta acción")
		return
	}
	var req struct {
		Reason    string `json:"reason"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if n := utf8.RuneCountInString(req.Reason); n < minRiskReason || n > maxRiskReason {
		writeErr(w, 400, "la justificación debe tener entre 10 y 2000 caracteres")
		return
	}
	exp, err := time.Parse("2006-01-02", req.ExpiresAt)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if err != nil || !exp.After(today) || exp.After(today.AddDate(0, 0, maxRiskAcceptDays)) {
		writeErr(w, 400, "la aceptación debe caducar entre mañana y dentro de 365 días")
		return
	}
	if it.State == db.RemVerifiedFixed {
		writeErr(w, 409, "el hallazgo ya está corregido y verificado")
		return
	}
	if err := pg.AcceptRemediationRisk(orgID, it.FindingID, userID, req.Reason, exp); err != nil {
		writeErr(w, 500, "no se pudo aceptar el riesgo")
		return
	}
	// Detail is "<YYYY-MM-DD> <reason>": language-neutral, so the portal can
	// render it in the reader's language.
	s.remediationChanged(r, pg, orgID, it.FindingID, actor, "risk_accepted",
		req.ExpiresAt+" "+req.Reason, "remediation.risk_accept")
	updated, _ := pg.GetRemediation(orgID, it.FindingID, time.Now())
	writeJSON(w, 200, map[string]any{"ok": true, "item": updated})
}

// POST /api/portal/remediation/{fid}/revoke-risk (admin+) — back to open.
func (s *Server) portalRevokeRisk(w http.ResponseWriter, r *http.Request) {
	pg, orgID, _, role, actor, it, ok := s.remediationTarget(w, r)
	if !ok {
		return
	}
	if roleRank(role) < roleRank("admin") {
		writeErr(w, 403, "no tienes permisos para esta acción")
		return
	}
	if it.RiskExpiresAt == "" {
		writeErr(w, 409, "no hay una aceptación de riesgo que revocar")
		return
	}
	if err := pg.SetRemediationWorkState(orgID, it.FindingID, "open"); err != nil {
		writeErr(w, 500, "no se pudo actualizar")
		return
	}
	s.remediationChanged(r, pg, orgID, it.FindingID, actor, "risk_revoked", "", "remediation.risk_revoke")
	updated, _ := pg.GetRemediation(orgID, it.FindingID, time.Now())
	writeJSON(w, 200, map[string]any{"ok": true, "item": updated})
}

// GET /api/admin/portal/retest-requests — client verification requests awaiting a
// NEXUS retest (operator console; the retest itself is launched from the finding
// detail with the existing human-in-the-loop retest flow).
func (s *Server) adminListRetestRequests(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	items, err := pg.AdminListRetestRequests()
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las verificaciones")
		return
	}
	writeJSON(w, 200, map[string]any{"requests": items})
}
