// Captación de solicitudes comerciales (leads) de la web pública de NEXUS.
// POST /api/public/leads es público (exento de auth); el resto requiere login.
// Código original de NEXUS.
package server

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nexus-ia/nexus/db"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// validServices bounds the "service of interest" field to known options so the
// public form can't be used to inject arbitrary content.
var validServices = map[string]bool{
	"": true, "offensive": true, "defensive": true, "purple": true,
	"ai-research": true, "managed": true, "other": true,
}

const (
	leadMaxLen     = 4000 // message hard cap
	leadFieldMax   = 200  // short field hard cap
	leadRatePerIP  = 5    // max submissions per window per IP
	leadRateWindow = 10 * time.Minute
)

// leadRequest is the public submission body. Website is a honeypot: real users
// never see it, so any value means a bot.
type leadRequest struct {
	Organization string `json:"organization"`
	ContactName  string `json:"contact_name"`
	Email        string `json:"email"`
	Country      string `json:"country"`
	Service      string `json:"service"`
	Message      string `json:"message"`
	Consent      bool   `json:"consent"`
	Website      string `json:"website"` // honeypot — must stay empty
}

// clientIP is the caller's address for rate limiting, brute-force lockout, the
// audit trail and the org network allowlist. X-Forwarded-For is only honored
// when the direct peer is a trusted proxy (loopback — the local Next.js proxy —
// or NEXUS_TRUSTED_PROXIES, e.g. the HTTPS proxy's Docker network), and then
// read from the right: each proxy appends the address it received the request
// from, so everything left of the last untrusted hop was written by the client
// and could be forged to dodge a per-IP limit, an allowlist or the audit trail.
func clientIP(r *http.Request) string {
	return clientIPWith(r, trustedProxies())
}

func clientIPWith(r *http.Request, trusted []*net.IPNet) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip == nil || !isTrustedProxy(ip, trusted) {
		return host
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return host
	}
	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(parts[i])
		ip := net.ParseIP(hop)
		if ip == nil {
			return hop // never skip past a hop we cannot read into client-written ones
		}
		if !isTrustedProxy(ip, trusted) {
			return ip.String()
		}
	}
	return host
}

var (
	trustedProxyOnce sync.Once
	trustedProxyNets []*net.IPNet
)

// trustedProxies are the reverse proxies besides loopback allowed to report
// the client address: NEXUS_TRUSTED_PROXIES, comma-separated CIDRs or single
// addresses, read once at first use.
func trustedProxies() []*net.IPNet {
	trustedProxyOnce.Do(func() {
		trustedProxyNets = parseTrustedProxies(os.Getenv("NEXUS_TRUSTED_PROXIES"))
	})
	return trustedProxyNets
}

func parseTrustedProxies(v string) []*net.IPNet {
	var out []*net.IPNet
	for _, item := range strings.Split(v, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !strings.Contains(item, "/") {
			if ip := net.ParseIP(item); ip != nil {
				bits := 128
				if ip.To4() != nil {
					bits = 32
				}
				item += "/" + strconv.Itoa(bits)
			}
		}
		_, n, err := net.ParseCIDR(item)
		if err != nil {
			log.Printf("[config] NEXUS_TRUSTED_PROXIES: ignoring invalid entry %q", item)
			continue
		}
		out = append(out, n)
	}
	return out
}

func isTrustedProxy(ip net.IP, trusted []*net.IPNet) bool {
	if ip.IsLoopback() {
		return true
	}
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func trimLimit(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max]
	}
	return s
}

// createLead handles the public commercial request form (POST /api/public/leads).
func (s *Server) createLead(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	if err := pg.EnsureLeadsTable(); err != nil {
		writeErr(w, 503, "almacenamiento de solicitudes no disponible")
		return
	}

	var req leadRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}

	// Honeypot: a filled hidden field means a bot. Return 200 so the bot can't
	// distinguish success from rejection, but store nothing.
	if strings.TrimSpace(req.Website) != "" {
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}

	req.Organization = trimLimit(req.Organization, leadFieldMax)
	req.ContactName = trimLimit(req.ContactName, leadFieldMax)
	req.Email = trimLimit(req.Email, leadFieldMax)
	req.Country = trimLimit(req.Country, leadFieldMax)
	req.Service = trimLimit(strings.ToLower(req.Service), 40)
	req.Message = trimLimit(req.Message, leadMaxLen)

	if req.Organization == "" || req.ContactName == "" || req.Email == "" {
		writeErr(w, 400, "organización, nombre de contacto y email son obligatorios")
		return
	}
	if !emailRe.MatchString(req.Email) {
		writeErr(w, 400, "email inválido")
		return
	}
	if !validServices[req.Service] {
		writeErr(w, 400, "servicio de interés inválido")
		return
	}
	if !req.Consent {
		writeErr(w, 400, "se requiere el consentimiento para procesar la solicitud")
		return
	}

	ip := clientIP(r)
	if n, err := pg.CountLeadsFromIPSince(ip, time.Now().Add(-leadRateWindow)); err == nil && n >= leadRatePerIP {
		writeErr(w, 429, "demasiadas solicitudes; intente más tarde")
		return
	}

	id, err := pg.InsertLead(&db.Lead{
		Organization: req.Organization, ContactName: req.ContactName, Email: req.Email,
		Country: req.Country, Service: req.Service, Message: req.Message,
		Status: "new", Consent: req.Consent, SourceIP: ip,
	})
	if err != nil {
		writeErr(w, 500, "no se pudo registrar la solicitud")
		return
	}
	log.Printf("[leads] nueva solicitud #%d org=%q service=%q ip=%s", id, req.Organization, req.Service, ip)
	writeJSON(w, 201, map[string]any{"ok": true, "id": id})
}

// listLeads returns recent leads for the admin panel (GET /api/leads, auth required).
func (s *Server) listLeads(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	if err := pg.EnsureLeadsTable(); err != nil {
		writeErr(w, 503, "almacenamiento de solicitudes no disponible")
		return
	}
	limit := atoiDefault(r.URL.Query().Get("limit"), 100)
	leads, err := pg.ListLeads(limit)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar las solicitudes")
		return
	}
	total, _ := pg.CountLeads()
	writeJSON(w, 200, map[string]any{"leads": leads, "total": total})
}

// updateLeadStatus changes a lead's workflow state (PATCH /api/leads/{id}, auth
// required). Reversible and bounded to the known status set.
func (s *Server) updateLeadStatus(w http.ResponseWriter, r *http.Request) {
	pg := s.pg(w)
	if pg == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "id inválido")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if !db.LeadStatuses[req.Status] {
		writeErr(w, 400, "estado inválido (new, contacted, qualified, closed)")
		return
	}
	ok, err := pg.UpdateLeadStatus(id, req.Status)
	if err != nil {
		writeErr(w, 500, "no se pudo actualizar la solicitud")
		return
	}
	if !ok {
		writeErr(w, 404, "solicitud no encontrada")
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "id": id, "status": req.Status})
}
