// Rules of Engagement (RoE) and the authorization gate in front of the engine.
// An evaluation request can only become an engine task when (1) every target it
// covers is verified, and (2) an org admin accepted the RoE for it: who signs,
// for which window, with which emergency contact and exclusions, against an
// exact, versioned text whose SHA-256 is stored with the acceptance.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

const roeVersion = "2026-10"

// maxRoEWindowDays caps one authorization window; longer engagements are
// authorized in successive windows.
const maxRoEWindowDays = 90

var roeTerms = map[string][]string{
	"es": {
		"Autorización. Quien acepta declara que tiene autoridad para autorizar estas pruebas en nombre de la organización, y que la organización es titular de los objetivos listados o cuenta con autorización escrita de su titular.",
		"Alcance. NEXUS solo probará los objetivos verificados que figuran en esta solicitud. Cualquier otro sistema queda fuera de alcance, incluidos los servicios de terceros (nube, SaaS, CDN) salvo que la organización disponga de su autorización.",
		"Ventana. Las pruebas se realizarán únicamente dentro de la ventana indicada. Fuera de ella NEXUS no iniciará ninguna prueba.",
		"Técnicas. Las pruebas buscan demostrar vulnerabilidades sin causar daño: NEXUS no realizará ataques de denegación de servicio, ingeniería social ni modificación o borrado de datos, y solo accederá a la información imprescindible para demostrar un hallazgo.",
		"Riesgos. Aunque se minimizan, las pruebas pueden generar carga, alertas o bloqueos en sistemas de seguridad. La organización informará a su equipo de seguridad y proveedores, y mantendrá copias de seguridad al día.",
		"Detención. Cualquiera de las partes puede detener las pruebas en cualquier momento. NEXUS las detendrá y avisará al contacto de emergencia si detecta un impacto inesperado o indicios de un incidente previo.",
		"Confidencialidad. Los resultados solo estarán disponibles para la organización a través de su portal y se tratarán como información confidencial.",
		"Registro. Esta aceptación queda registrada con fecha, dirección IP, usuario y la huella SHA-256 de este texto.",
	},
	"en": {
		"Authorization. The person accepting declares that they have authority to authorize this testing on behalf of the organization, and that the organization owns the listed targets or holds written authorization from their owner.",
		"Scope. NEXUS will only test the verified targets listed in this request. Every other system is out of scope, including third-party services (cloud, SaaS, CDN) unless the organization holds their authorization.",
		"Window. Testing will only take place within the stated window. NEXUS will not start any test outside it.",
		"Techniques. Testing aims to demonstrate vulnerabilities without causing harm: NEXUS will not perform denial-of-service attacks, social engineering, or modification or deletion of data, and will only access the information strictly needed to demonstrate a finding.",
		"Risks. Although minimized, testing may cause load, alerts or blocks in security systems. The organization will inform its security team and providers, and keep backups up to date.",
		"Stopping. Either party may stop the testing at any time. NEXUS will stop it and alert the emergency contact if it detects unexpected impact or signs of a pre-existing incident.",
		"Confidentiality. Results will only be available to the organization through its portal and will be handled as confidential information.",
		"Record. This acceptance is recorded with date, IP address, user and the SHA-256 fingerprint of this text.",
	},
}

// roeHash fingerprints the exact text shown in one language for one version.
func roeHash(lang string) string {
	sum := sha256.Sum256([]byte(roeVersion + "\n" + lang + "\n" + strings.Join(roeTerms[lang], "\n")))
	return hex.EncodeToString(sum[:])
}

func roeLang(l string) string {
	if l == "en" {
		return "en"
	}
	return "es"
}

// clientRequest resolves a request of the caller's org (nil → 404 written).
func (s *Server) clientRequest(w http.ResponseWriter, pg *db.DB, orgID int64, r *http.Request) *db.AdminPortalRequest {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "id inválido")
		return nil
	}
	pr, err := pg.GetPortalRequest(id)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la solicitud")
		return nil
	}
	if pr == nil || pr.OrgID != orgID {
		writeErr(w, 404, "solicitud no encontrada")
		return nil
	}
	return pr
}

// GET /api/portal/requests/{id}/authorization?lang= (member+) — targets, RoE
// status and the current terms to accept.
func (s *Server) portalRequestAuthorization(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "member")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	pr := s.clientRequest(w, pg, orgID, r)
	if pr == nil {
		return
	}
	auths, err := pg.RequestAuthorizations([]int64{pr.ID})
	if err != nil {
		writeErr(w, 500, "no se pudo leer la autorización")
		return
	}
	lang := roeLang(r.URL.Query().Get("lang"))
	writeJSON(w, 200, map[string]any{
		"authorization":   auths[pr.ID],
		"promoted":        pr.EngineTaskID != nil,
		"terms":           map[string]any{"version": roeVersion, "lang": lang, "text": roeTerms[lang], "hash": roeHash(lang)},
		"max_window_days": maxRoEWindowDays,
	})
}

// POST /api/portal/requests/{id}/roe (admin+) — accept the Rules of Engagement.
func (s *Server) portalAcceptRoE(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	pr := s.clientRequest(w, pg, orgID, r)
	if pr == nil {
		return
	}
	var req struct {
		Accept           bool   `json:"accept"`
		Lang             string `json:"lang"`
		Version          string `json:"version"`
		Hash             string `json:"hash"`
		SignerName       string `json:"signer_name"`
		SignerTitle      string `json:"signer_title"`
		WindowStart      string `json:"window_start"`
		WindowEnd        string `json:"window_end"`
		EmergencyContact string `json:"emergency_contact"`
		Exclusions       string `json:"exclusions"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	if !req.Accept {
		writeErr(w, 400, "debes aceptar las reglas de enfrentamiento")
		return
	}
	lang := roeLang(req.Lang)
	if req.Version != roeVersion || req.Hash != roeHash(lang) {
		writeErr(w, 409, "los términos han cambiado; revísalos de nuevo")
		return
	}
	if pr.EngineTaskID != nil {
		writeErr(w, 409, "la evaluación ya está en marcha")
		return
	}
	trim := func(s string, min, max int) (string, bool) {
		s = strings.TrimSpace(s)
		n := len([]rune(s))
		return s, n >= min && n <= max
	}
	var okName, okTitle, okContact, okExcl bool
	req.SignerName, okName = trim(req.SignerName, 3, 120)
	req.SignerTitle, okTitle = trim(req.SignerTitle, 2, 120)
	req.EmergencyContact, okContact = trim(req.EmergencyContact, 5, 200)
	req.Exclusions, okExcl = trim(req.Exclusions, 0, 2000)
	if !okName || !okTitle {
		writeErr(w, 400, "indica el nombre y el cargo de quien autoriza")
		return
	}
	if !okContact {
		writeErr(w, 400, "indica un contacto de emergencia")
		return
	}
	if !okExcl {
		writeErr(w, 400, "las exclusiones deben tener como máximo 2000 caracteres")
		return
	}
	start, err1 := time.Parse("2006-01-02", req.WindowStart)
	end, err2 := time.Parse("2006-01-02", req.WindowEnd)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if err1 != nil || err2 != nil || start.Before(today) || end.Before(start) || end.Sub(start) > maxRoEWindowDays*24*time.Hour {
		writeErr(w, 400, "ventana de pruebas inválida (desde hoy, máximo 90 días)")
		return
	}
	auths, err := pg.RequestAuthorizations([]int64{pr.ID})
	if err != nil {
		writeErr(w, 500, "no se pudo leer la autorización")
		return
	}
	if len(auths[pr.ID].Targets) == 0 {
		writeErr(w, 400, "la solicitud no tiene objetivos")
		return
	}
	email := ""
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		email = u.Email
	}
	roe := &db.RoE{
		RequestID: pr.ID, AcceptedEmail: email, SignerName: req.SignerName, SignerTitle: req.SignerTitle,
		WindowStart: start.Format("2006-01-02"), WindowEnd: end.Format("2006-01-02"),
		EmergencyContact: req.EmergencyContact, Exclusions: req.Exclusions,
		Version: roeVersion + "/" + lang, Hash: roeHash(lang), IP: clientIP(r),
	}
	created, err := pg.AcceptRoE(orgID, userID, roe)
	if err != nil {
		writeErr(w, 500, "no se pudo registrar la aceptación")
		return
	}
	if !created {
		writeErr(w, 409, "las reglas de enfrentamiento ya están aceptadas")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(userID, 10), "portal_user", "roe.accept",
		"request:"+strconv.FormatInt(pr.ID, 10), roe.Version+" "+roe.WindowStart+".."+roe.WindowEnd+" sha256:"+roe.Hash[:16])
	auths, _ = pg.RequestAuthorizations([]int64{pr.ID})
	writeJSON(w, 201, map[string]any{"ok": true, "authorization": auths[pr.ID]})
}

// authorizationProblems lists why a request is not ready for the engine (empty
// = ready). The operator console shows these verbatim.
func authorizationProblems(a *db.RequestAuth) []string {
	var out []string
	if a == nil || len(a.Targets) == 0 {
		return []string{"la solicitud no tiene objetivos"}
	}
	for _, t := range a.Targets {
		switch t.Status {
		case "pending":
			out = append(out, t.Value+": propiedad sin verificar")
		case "rejected":
			out = append(out, t.Value+": rechazado por NEXUS")
		case "expired":
			out = append(out, t.Value+": verificación caducada")
		}
	}
	if a.RoE == nil {
		out = append(out, "reglas de enfrentamiento sin aceptar")
	} else if a.RoE.WindowEnd < time.Now().UTC().Format("2006-01-02") {
		out = append(out, "la ventana de pruebas ya terminó")
	}
	return out
}

// engineScopeForTarget maps a verified target to the narrowest engine scope
// that matches what was proven: a whole root domain only when its owner proved
// control of the DNS zone itself; otherwise exactly that host (or IP).
func engineScopeForTarget(t *db.PortalTarget) (kind, value string) {
	if t.Kind == "ip" {
		return "ip", t.Value
	}
	if root, _ := db.RootDomain(t.Value); root == t.Value && t.Method == "dns" {
		return "root_domain", t.Value
	}
	return "subdomain", t.Value
}
