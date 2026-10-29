package server

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

// GET /api/portal/requests/{id}/report?kind=executive|technical&lang=es|en
// Branded PDF for one of the caller's evaluations. Only once an operator has
// released its results (same gate as the results view); every download is
// recorded in the audit trail.
func (s *Server) portalRequestReport(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "member")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	rq, kind, ok := reportRequest(w, r, pg)
	if !ok {
		return
	}
	if rq.OrgID != orgID {
		writeErr(w, 404, "solicitud no encontrada")
		return
	}
	if !rq.ResultsReleased || rq.EngineTaskID == nil {
		writeErr(w, 404, "resultado no disponible")
		return
	}
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		if _, pref, err := pg.GetPortalUserPrefs(userID); err == nil {
			lang = pref
		}
	}
	actor := ""
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		actor = u.Email
	}
	if s.writeReport(w, pg, rq, kind, lang, false) {
		s.audit(r, &orgID, actor, "portal_user", "report.download", "request:"+strconv.FormatInt(rq.ID, 10), kind)
	}
}

// GET /api/admin/portal/requests/{id}/report?kind=...&lang=... — operator copy.
// Unreleased results are rendered as a watermarked draft for internal review.
func (s *Server) adminRequestReport(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	rq, kind, ok := reportRequest(w, r, pg)
	if !ok {
		return
	}
	if rq.EngineTaskID == nil {
		writeErr(w, 409, "la solicitud aún no tiene una tarea asociada")
		return
	}
	if s.writeReport(w, pg, rq, kind, r.URL.Query().Get("lang"), !rq.ResultsReleased) {
		s.audit(r, &rq.OrgID, "operator", "operator", "report.download", "request:"+strconv.FormatInt(rq.ID, 10), kind)
	}
}

func reportRequest(w http.ResponseWriter, r *http.Request, pg *db.DB) (*db.AdminPortalRequest, string, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "id inválido")
		return nil, "", false
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = reportExecutive
	}
	if kind != reportExecutive && kind != reportTechnical {
		writeErr(w, 400, "tipo de informe inválido")
		return nil, "", false
	}
	if l := r.URL.Query().Get("lang"); !validPortalLang(l) {
		writeErr(w, 400, "idioma no soportado")
		return nil, "", false
	}
	rq, err := pg.GetPortalRequest(id)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return nil, "", false
	}
	if rq == nil {
		writeErr(w, 404, "solicitud no encontrada")
		return nil, "", false
	}
	return rq, kind, true
}

var reSlug = regexp.MustCompile(`[^a-z0-9]+`)

func reportSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n").Replace(s)
	s = strings.Trim(reSlug.ReplaceAllString(s, "-"), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	if s == "" {
		s = "evaluacion"
	}
	return s
}

// writeReport renders and streams the PDF; it reports whether it was sent.
func (s *Server) writeReport(w http.ResponseWriter, pg *db.DB, rq *db.AdminPortalRequest, kind, lang string, draft bool) bool {
	if lang != "en" {
		lang = "es"
	}
	now := time.Now()
	items, reports, err := pg.ListRequestReportItems(rq.OrgID, rq.ID, now)
	if err != nil {
		writeErr(w, 500, "no se pudo generar el informe")
		return false
	}
	pdf, err := renderReport(&reportInput{
		Kind: kind, Lang: lang, OrgName: rq.OrgName, Title: rq.Title, Scope: rq.Scope,
		RequestID: rq.ID, RequestedAt: rq.CreatedAt, GeneratedAt: now, Draft: draft,
		Items: items, Reports: reports,
	})
	if err != nil {
		writeErr(w, 500, "no se pudo generar el informe")
		return false
	}
	name := map[string]map[string]string{
		"es": {reportExecutive: "informe-ejecutivo", reportTechnical: "informe-tecnico"},
		"en": {reportExecutive: "executive-report", reportTechnical: "technical-report"},
	}[lang][kind]
	file := "NEXUS-" + name + "-" + reportSlug(rq.Title) + "-" + now.UTC().Format("20060102") + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+file+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
	w.WriteHeader(200)
	_, _ = w.Write(pdf)
	return true
}
