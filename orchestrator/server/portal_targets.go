// Target ownership verification. Before NEXUS tests anything for a client, the
// client proves it controls each target:
//   - DNS: a TXT record holding the token at _nexus-verify.<domain>, or
//   - HTTP: the token as the body of /.well-known/nexus-verify.txt on the host,
//   - or a NEXUS operator approves it manually on documentary evidence (signed
//     authorization letter, contract) — the only path for private addresses.
//
// The HTTP check is an outbound request to a client-supplied host, so it is
// SSRF-hardened: it only ever fetches that fixed path, resolves the host itself
// and refuses non-public addresses, dials the vetted IP (no DNS rebinding),
// follows no redirects, and reads at most 1 KB within a short timeout.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/db"
)

const (
	targetProofPath   = "/.well-known/nexus-verify.txt"
	targetDNSPrefix   = "_nexus-verify."
	targetCheckEvery  = 15 * time.Second // per-target throttle between automatic checks
	targetCheckBudget = 8 * time.Second
	maxTargetsPerOrg  = 200
)

var domainLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// normalizeTarget validates a client-supplied target and returns its kind and
// canonical value. Accepts a bare domain or IP (a pasted URL is reduced to its
// host). Wildcards, ports, paths and single-label names are rejected: each
// target is one concrete host the client can prove control of.
func normalizeTarget(raw string) (kind, value string, err error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.Index(v, "://"); i >= 0 {
		v = v[i+3:]
	}
	if i := strings.IndexAny(v, "/?#"); i >= 0 {
		v = v[:i]
	}
	v = strings.TrimSuffix(v, ".")
	if v == "" || len(v) > 253 {
		return "", "", errors.New("objetivo inválido")
	}
	if ip, perr := netip.ParseAddr(strings.Trim(v, "[]")); perr == nil {
		if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLoopback() {
			return "", "", errors.New("dirección IP no admitida")
		}
		return "ip", ip.Unmap().String(), nil
	}
	if strings.ContainsAny(v, "*:@ ") {
		return "", "", errors.New("indica un dominio o IP concreto, sin comodines ni puertos")
	}
	labels := strings.Split(v, ".")
	if len(labels) < 2 {
		return "", "", errors.New("objetivo inválido")
	}
	for _, l := range labels {
		if !domainLabel.MatchString(l) {
			return "", "", errors.New("objetivo inválido")
		}
	}
	if tld := labels[len(labels)-1]; !strings.HasPrefix(tld, "xn--") && strings.Trim(tld, "abcdefghijklmnopqrstuvwxyz") != "" {
		return "", "", errors.New("objetivo inválido")
	}
	return "domain", v, nil
}

func newTargetToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "nexus-verify-" + hex.EncodeToString(b), nil
}

// publicAddr reports whether an address is routable on the public internet.
// Everything else (loopback, RFC 1918, CGNAT, link-local incl. cloud metadata,
// ULA, documentation and benchmarking ranges) is off-limits to the checker.
func publicAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	for _, p := range nonPublicPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

var nonPublicPrefixes = func() []netip.Prefix {
	var out []netip.Prefix
	for _, s := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "64:ff9b::/96", "2001:db8::/32",
	} {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()

// resolvePublic resolves host and returns its public addresses, or an error if
// it resolves to any non-public address (a mixed answer is refused outright).
func resolvePublic(ctx context.Context, host string) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !publicAddr(ip) {
			return nil, errors.New("la dirección no es pública; solicita la aprobación manual de NEXUS")
		}
		return []netip.Addr{ip}, nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addrs) == 0 {
		return nil, errors.New("no se pudo resolver el dominio")
	}
	for _, a := range addrs {
		if !publicAddr(a) {
			return nil, errors.New("el dominio apunta a una dirección no pública; solicita la aprobación manual de NEXUS")
		}
	}
	return addrs, nil
}

// targetHTTPClient dials only the pre-vetted address and never follows redirects.
func targetHTTPClient(addr netip.Addr) *http.Client {
	dialer := &net.Dialer{Timeout: 4 * time.Second}
	return &http.Client{
		Timeout: targetCheckBudget,
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, network, hostport string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(hostport)
				if err != nil {
					return nil, err
				}
				return dialer.DialContext(ctx, "tcp", net.JoinHostPort(addr.String(), port))
			},
			TLSHandshakeTimeout:    4 * time.Second,
			ResponseHeaderTimeout:  5 * time.Second,
			MaxResponseHeaderBytes: 8 << 10,
			DisableKeepAlives:      true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// checkTargetHTTP looks for the token at https://host/.well-known/... then http.
func checkTargetHTTP(ctx context.Context, host, token string) error {
	addrs, err := resolvePublic(ctx, host)
	if err != nil {
		return err
	}
	client := targetHTTPClient(addrs[0])
	urlHost := host
	if strings.Contains(host, ":") { // IPv6 literal
		urlHost = "[" + host + "]"
	}
	var last error
	for _, scheme := range []string{"https", "http"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+urlHost+targetProofPath, nil)
		if err != nil {
			return errors.New("objetivo inválido")
		}
		req.Header.Set("User-Agent", "NEXUS-Ownership-Verifier/1.0")
		resp, err := client.Do(req)
		if err != nil {
			last = fmt.Errorf("no se pudo conectar por %s", scheme)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			last = fmt.Errorf("%s respondió %d en %s", scheme, resp.StatusCode, targetProofPath)
			continue
		}
		if strings.TrimSpace(string(body)) == token {
			return nil
		}
		last = errors.New("el archivo existe pero no contiene el token esperado")
	}
	return last
}

// checkTargetDNS looks for the token (already "nexus-verify-…", so it is
// self-describing) in the TXT records of _nexus-verify.<domain>.
func checkTargetDNS(ctx context.Context, domain, token string) error {
	txts, err := net.DefaultResolver.LookupTXT(ctx, targetDNSPrefix+domain)
	if err != nil {
		return errors.New("no se encontró el registro TXT " + targetDNSPrefix + domain)
	}
	for _, txt := range txts {
		if strings.TrimSpace(strings.Trim(txt, `"`)) == token {
			return nil
		}
	}
	return errors.New("el registro TXT existe pero no contiene el token esperado")
}

// targetsResponse adds what the client needs to publish the proof.
func targetsResponse(list []*db.PortalTarget) map[string]any {
	return map[string]any{
		"targets":       list,
		"validity_days": int(db.TargetVerificationValidity.Hours() / 24),
		"dns_prefix":    targetDNSPrefix,
		"http_path":     targetProofPath,
	}
}

func targetIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "id inválido")
		return 0, false
	}
	return id, true
}

// GET /api/portal/targets (member+)
func (s *Server) portalListTargets(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "member")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	list, err := pg.ListPortalTargets(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar los objetivos")
		return
	}
	writeJSON(w, 200, targetsResponse(list))
}

// POST /api/portal/targets {value} (admin+)
func (s *Server) portalCreateTarget(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Value string `json:"value"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	kind, value, err := normalizeTarget(req.Value)
	if err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	if list, err := pg.ListPortalTargets(orgID); err == nil && len(list) >= maxTargetsPerOrg {
		writeErr(w, 409, "has alcanzado el máximo de objetivos de tu organización")
		return
	}
	token, err := newTargetToken()
	if err != nil {
		writeErr(w, 500, "no se pudo registrar el objetivo")
		return
	}
	id, err := pg.CreatePortalTarget(orgID, userID, kind, value, token)
	if errors.Is(err, db.ErrTargetExists) {
		writeErr(w, 409, "ese objetivo ya está registrado")
		return
	}
	if err != nil {
		writeErr(w, 500, "no se pudo registrar el objetivo")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(userID, 10), "portal_user", "target.add", "target:"+strconv.FormatInt(id, 10), value)
	t, _ := pg.GetPortalTarget(orgID, id)
	writeJSON(w, 201, map[string]any{"ok": true, "target": t})
}

// POST /api/portal/targets/{id}/verify {method: dns|http} (admin+)
func (s *Server) portalVerifyTarget(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	id, ok := targetIDParam(w, r)
	if !ok {
		return
	}
	var req struct {
		Method string `json:"method"`
	}
	if err := decode(r, &req); err != nil || (req.Method != "dns" && req.Method != "http") {
		writeErr(w, 400, "método de verificación inválido")
		return
	}
	t, err := pg.GetPortalTarget(orgID, id)
	if err != nil {
		writeErr(w, 500, "no se pudo leer el objetivo")
		return
	}
	if t == nil {
		writeErr(w, 404, "objetivo no encontrado")
		return
	}
	if t.Status == "verified" {
		writeJSON(w, 200, map[string]any{"ok": true, "target": t})
		return
	}
	if t.Status == "rejected" {
		writeErr(w, 409, "NEXUS rechazó este objetivo; contacta con tu gestor")
		return
	}
	if req.Method == "dns" && t.Kind != "domain" {
		writeErr(w, 400, "la verificación por DNS solo aplica a dominios")
		return
	}
	if t.LastCheckAt != nil && time.Since(*t.LastCheckAt) < targetCheckEvery {
		writeErr(w, 429, "espera unos segundos antes de volver a comprobar")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), targetCheckBudget+2*time.Second)
	defer cancel()
	var cerr error
	if req.Method == "dns" {
		cerr = checkTargetDNS(ctx, t.Value, t.Token)
	} else {
		cerr = checkTargetHTTP(ctx, t.Value, t.Token)
	}
	errText := ""
	if cerr != nil {
		errText = cerr.Error()
	}
	if err := pg.RecordTargetCheck(id, cerr == nil, req.Method, errText); err != nil {
		writeErr(w, 500, "no se pudo guardar la verificación")
		return
	}
	if cerr == nil {
		s.audit(r, &orgID, "user:"+strconv.FormatInt(userID, 10), "portal_user", "target.verify",
			"target:"+strconv.FormatInt(id, 10), req.Method+" "+t.Value)
	}
	t, _ = pg.GetPortalTarget(orgID, id)
	writeJSON(w, 200, map[string]any{"ok": cerr == nil, "target": t, "error": errText})
}

// DELETE /api/portal/targets/{id} (admin+) — not while an evaluation that
// covers it is running.
func (s *Server) portalDeleteTarget(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	id, ok := targetIDParam(w, r)
	if !ok {
		return
	}
	t, err := pg.GetPortalTarget(orgID, id)
	if err != nil {
		writeErr(w, 500, "no se pudo leer el objetivo")
		return
	}
	if t == nil {
		writeErr(w, 404, "objetivo no encontrado")
		return
	}
	if busy, err := pg.TargetInActiveRequest(id); err != nil || busy {
		writeErr(w, 409, "el objetivo forma parte de una evaluación en curso")
		return
	}
	if err := pg.DeletePortalTarget(orgID, id); err != nil {
		writeErr(w, 500, "no se pudo eliminar el objetivo")
		return
	}
	s.audit(r, &orgID, "user:"+strconv.FormatInt(userID, 10), "portal_user", "target.delete", "target:"+strconv.FormatInt(id, 10), t.Value)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/admin/portal/targets?status=pending — operator, all orgs.
func (s *Server) adminListTargets(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "pending" && status != "verified" && status != "rejected" {
		writeErr(w, 400, "estado inválido")
		return
	}
	list, err := pg.AdminListPortalTargets(status)
	if err != nil {
		writeErr(w, 500, "no se pudieron listar los objetivos")
		return
	}
	writeJSON(w, 200, map[string]any{"targets": list})
}

// POST /api/admin/portal/targets/{id}/decision {approve, note} — operator
// approves (on documentary evidence) or rejects a target. The note is required:
// it is the record of WHY NEXUS considered the client authorized.
func (s *Server) adminDecideTarget(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	id, ok := targetIDParam(w, r)
	if !ok {
		return
	}
	var req struct {
		Approve bool   `json:"approve"`
		Note    string `json:"note"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	if len([]rune(req.Note)) < 10 || len([]rune(req.Note)) > 1000 {
		writeErr(w, 400, "la nota debe tener entre 10 y 1000 caracteres")
		return
	}
	t, err := pg.AdminGetPortalTarget(id)
	if err != nil {
		writeErr(w, 500, "no se pudo leer el objetivo")
		return
	}
	if t == nil {
		writeErr(w, 404, "objetivo no encontrado")
		return
	}
	if err := pg.SetTargetDecision(id, req.Approve, req.Note); err != nil {
		writeErr(w, 500, "no se pudo guardar la decisión")
		return
	}
	action := "target.reject"
	if req.Approve {
		action = "target.approve"
	}
	s.audit(r, &t.OrgID, "operator", "operator", action, "target:"+strconv.FormatInt(id, 10), t.Value+" — "+req.Note)
	t, _ = pg.AdminGetPortalTarget(id)
	writeJSON(w, 200, map[string]any{"ok": true, "target": t})
}
