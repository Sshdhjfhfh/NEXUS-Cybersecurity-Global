// Single sign-on for portal organizations: OpenID Connect, authorization code
// flow with PKCE (S256), nonce and single-use state. Each org configures its own
// identity provider (Entra ID, Google Workspace, Okta, Keycloak...). An org can
// only claim email domains it has proven it owns (DNS-verified targets), and an
// SSO identity can only ever sign in to an account of that same org.
package server

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nexus-ia/nexus/db"
)

const (
	ssoMaxDomains = 20
	idpCacheTTL   = 10 * time.Minute
	idpBodyLimit  = 1 << 20
)

// ssoRedirectURI is the callback registered at the identity provider. It is
// fixed server-side (never taken from the request) so an authorization code can
// only ever be delivered to the portal.
func ssoRedirectURI() string {
	return strings.TrimRight(envStr("NEXUS_PORTAL_BASE_URL", "http://localhost:3000"), "/") + "/portal/sso/callback"
}

// ssoDevInsecure lets a local test identity provider run over plain HTTP on a
// private address. Never set in production: there, issuers must be HTTPS on a
// public address (the same SSRF guard as target verification).
func ssoDevInsecure() bool { return os.Getenv("NEXUS_SSO_DEV_INSECURE") == "1" }

func randB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// --- talking to the identity provider -------------------------------------

func checkIdPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("URL inválida")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && ssoDevInsecure()) {
		return nil, errors.New("el proveedor debe usar https")
	}
	return u, nil
}

// idpDo performs a request to the identity provider: HTTPS only, public
// addresses only (dialing the vetted address), no redirects, bounded body.
func idpDo(ctx context.Context, req *http.Request) ([]byte, int, error) {
	u, err := checkIdPURL(req.URL.String())
	if err != nil {
		return nil, 0, err
	}
	var client *http.Client
	if ssoDevInsecure() {
		client = &http.Client{
			Timeout:       targetCheckBudget,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	} else {
		addrs, err := resolvePublic(ctx, u.Hostname())
		if err != nil {
			return nil, 0, err
		}
		client = targetHTTPClient(addrs[0])
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, 0, errors.New("el proveedor de identidad no responde")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, idpBodyLimit))
	return body, resp.StatusCode, err
}

func idpGetJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	body, code, err := idpDo(ctx, req)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("el proveedor respondió %d", code)
	}
	return json.Unmarshal(body, out)
}

type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

var idpCache = struct {
	sync.Mutex
	disc map[string]cachedDisc
	jwks map[string]cachedJWKS
}{disc: map[string]cachedDisc{}, jwks: map[string]cachedJWKS{}}

type cachedDisc struct {
	d  *oidcDiscovery
	at time.Time
}
type cachedJWKS struct {
	keys map[string]crypto.PublicKey
	at   time.Time
}

// oidcDiscover reads (and caches) the issuer's OpenID configuration. Per the
// spec, the document's issuer must equal the configured one exactly.
func oidcDiscover(ctx context.Context, issuer string, fresh bool) (*oidcDiscovery, error) {
	if !fresh {
		idpCache.Lock()
		c, ok := idpCache.disc[issuer]
		idpCache.Unlock()
		if ok && time.Since(c.at) < idpCacheTTL {
			return c.d, nil
		}
	}
	var d oidcDiscovery
	if err := idpGetJSON(ctx, issuer+"/.well-known/openid-configuration", &d); err != nil {
		return nil, err
	}
	if d.Issuer != issuer {
		return nil, errors.New("el emisor publicado no coincide con el configurado")
	}
	for _, e := range []string{d.AuthorizationEndpoint, d.TokenEndpoint, d.JWKSURI} {
		if _, err := checkIdPURL(e); err != nil {
			return nil, errors.New("la configuración del proveedor está incompleta o no usa https")
		}
	}
	idpCache.Lock()
	idpCache.disc[issuer] = cachedDisc{&d, time.Now()}
	idpCache.Unlock()
	return &d, nil
}

// idpKey returns the provider's signing key for kid, refreshing the JWKS once
// when the kid is unknown (key rotation).
func idpKey(ctx context.Context, jwksURI, kid string) (crypto.PublicKey, error) {
	for attempt := 0; attempt < 2; attempt++ {
		idpCache.Lock()
		c, ok := idpCache.jwks[jwksURI]
		idpCache.Unlock()
		if !ok || attempt == 1 || time.Since(c.at) > idpCacheTTL {
			keys, err := fetchJWKS(ctx, jwksURI)
			if err != nil {
				return nil, err
			}
			c = cachedJWKS{keys, time.Now()}
			idpCache.Lock()
			idpCache.jwks[jwksURI] = c
			idpCache.Unlock()
		}
		if k, ok := c.keys[kid]; ok {
			return k, nil
		}
		if kid == "" && len(c.keys) == 1 {
			for _, k := range c.keys {
				return k, nil
			}
		}
	}
	return nil, errors.New("clave de firma desconocida")
}

func fetchJWKS(ctx context.Context, jwksURI string) (map[string]crypto.PublicKey, error) {
	var set struct {
		Keys []struct {
			Kty, Kid, Use, Crv, N, E, X, Y string
		} `json:"keys"`
	}
	if err := idpGetJSON(ctx, jwksURI, &set); err != nil {
		return nil, err
	}
	out := map[string]crypto.PublicKey{}
	dec := base64.RawURLEncoding
	for _, k := range set.Keys {
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		switch k.Kty {
		case "RSA":
			n, err1 := dec.DecodeString(k.N)
			e, err2 := dec.DecodeString(k.E)
			if err1 != nil || err2 != nil || len(n) < 256 || len(e) == 0 || len(e) > 4 {
				continue // RSA keys under 2048 bits are not accepted
			}
			out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
		case "EC":
			if k.Crv != "P-256" {
				continue
			}
			x, err1 := dec.DecodeString(k.X)
			y, err2 := dec.DecodeString(k.Y)
			if err1 != nil || err2 != nil {
				continue
			}
			pub := &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}
			if !pub.Curve.IsOnCurve(pub.X, pub.Y) {
				continue
			}
			out[k.Kid] = pub
		}
	}
	if len(out) == 0 {
		return nil, errors.New("el proveedor no publica claves de firma válidas")
	}
	return out, nil
}

type ssoIdentity struct {
	Subject string // issuer|sub — stable, never the email
	Email   string
}

// verifyIDToken validates an ID token: signature with the provider's published
// key (RS256/ES256 only, never "none" or HMAC), issuer, audience (and azp when
// there are several audiences), expiry, nonce, and a verified email in one of
// the org's domains.
func verifyIDToken(ctx context.Context, cfg *db.PortalSSO, disc *oidcDiscovery, raw, nonce string) (*ssoIdentity, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return idpKey(ctx, disc.JWKSURI, kid)
	},
		jwt.WithValidMethods([]string{"RS256", "ES256"}),
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.ClientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(60*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("id_token inválido: %w", err)
	}
	if aud, _ := claims.GetAudience(); len(aud) > 1 {
		if azp, _ := claims["azp"].(string); azp != cfg.ClientID {
			return nil, errors.New("azp no coincide")
		}
	}
	got, _ := claims["nonce"].(string)
	if subtle.ConstantTimeCompare([]byte(got), []byte(nonce)) != 1 {
		return nil, errors.New("nonce no coincide")
	}
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	email = strings.ToLower(strings.TrimSpace(email))
	if sub == "" || !emailRe.MatchString(email) {
		return nil, errors.New("faltan sub o email")
	}
	// The org's own IdP vouches for its users; an explicit "not verified" is refused.
	if v, present := claims["email_verified"]; present && v != true && v != "true" {
		return nil, errors.New("email no verificado por el proveedor")
	}
	domain := email[strings.LastIndexByte(email, '@')+1:]
	ok := false
	for _, d := range cfg.EmailDomains {
		ok = ok || d == domain
	}
	if !ok {
		return nil, errors.New("dominio de email no permitido")
	}
	return &ssoIdentity{Subject: cfg.Issuer + "|" + sub, Email: email}, nil
}

// --- configuration (owner edits; admins and auditors read) -----------------

func ssoView(c *db.PortalSSO) any {
	if c == nil {
		return nil
	}
	return map[string]any{
		"issuer": c.Issuer, "client_id": c.ClientID, "has_secret": c.SecretEnc != "",
		"email_domains": c.EmailDomains, "enabled": c.Enabled, "enforce": c.Enforce,
		"auto_provision": c.AutoProvision, "default_role": c.DefaultRole,
		"updated_at": c.UpdatedAt, "updated_by": c.UpdatedBy,
	}
}

func (s *Server) writeSSOConfig(w http.ResponseWriter, pg *db.DB, orgID int64) {
	c, err := pg.GetPortalSSO(orgID)
	if err != nil {
		writeErr(w, 500, "no se pudo leer la configuración SSO")
		return
	}
	domains, err := pg.VerifiedDNSDomains(orgID)
	if err != nil {
		domains = []string{}
	}
	writeJSON(w, 200, map[string]any{
		"sso": ssoView(c), "redirect_uri": ssoRedirectURI(), "verified_domains": domains,
	})
}

// GET /api/portal/sso
func (s *Server) portalGetSSO(w http.ResponseWriter, r *http.Request) {
	orgID, _, _, ok := s.portalRequireRole(w, r, "admin")
	if !ok {
		return
	}
	if pg := s.portalDB(w); pg != nil {
		s.writeSSOConfig(w, pg, orgID)
	}
}

// PUT /api/portal/sso {issuer, client_id, client_secret, email_domains, enabled,
// enforce, auto_provision, default_role} — owner only. An empty client_secret
// keeps the stored one.
func (s *Server) portalSetSSO(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "owner")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Issuer        string   `json:"issuer"`
		ClientID      string   `json:"client_id"`
		ClientSecret  string   `json:"client_secret"`
		EmailDomains  []string `json:"email_domains"`
		Enabled       bool     `json:"enabled"`
		Enforce       bool     `json:"enforce"`
		AutoProvision bool     `json:"auto_provision"`
		DefaultRole   string   `json:"default_role"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	req.Issuer = strings.TrimRight(strings.TrimSpace(req.Issuer), "/")
	req.ClientID = strings.TrimSpace(req.ClientID)
	if u, err := checkIdPURL(req.Issuer); err != nil || u.RawQuery != "" || len(req.Issuer) > 500 {
		writeErr(w, 400, "emisor (issuer) inválido: debe ser una URL https")
		return
	}
	if req.ClientID == "" || len(req.ClientID) > 300 || len(req.ClientSecret) > 2000 {
		writeErr(w, 400, "client_id o client_secret inválido")
		return
	}
	if req.DefaultRole == "" {
		req.DefaultRole = "member"
	}
	// Auto-provisioned accounts never get admin rights: those are granted by a person.
	if req.DefaultRole != "member" && req.DefaultRole != roleAuditor {
		writeErr(w, 400, "el rol por defecto debe ser member o auditor")
		return
	}
	if req.Enforce && !req.Enabled {
		writeErr(w, 400, "activa el SSO antes de exigirlo")
		return
	}
	verified, err := pg.VerifiedDNSDomains(orgID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	isVerified := map[string]bool{}
	for _, d := range verified {
		isVerified[d] = true
	}
	domains := []string{}
	seen := map[string]bool{}
	for _, d := range req.EmailDomains {
		d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(d), "@")))
		if d == "" || seen[d] {
			continue
		}
		if !isVerified[d] {
			writeErr(w, 400, fmt.Sprintf("el dominio %s no tiene la propiedad verificada por DNS en Objetivos", d))
			return
		}
		seen[d] = true
		domains = append(domains, d)
	}
	if len(domains) == 0 || len(domains) > ssoMaxDomains {
		writeErr(w, 400, fmt.Sprintf("indica entre 1 y %d dominios de email", ssoMaxDomains))
		return
	}
	prev, err := pg.GetPortalSSO(orgID)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	enc := ""
	if req.ClientSecret != "" {
		if enc, err = s.encryptSecret(req.ClientSecret); err != nil {
			writeErr(w, 500, "no se pudo cifrar el secreto")
			return
		}
	} else if prev == nil || prev.SecretEnc == "" {
		writeErr(w, 400, "client_id o client_secret inválido")
		return
	}
	if req.Enabled {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if _, err := oidcDiscover(ctx, req.Issuer, true); err != nil {
			writeErr(w, 400, "no se pudo leer la configuración OIDC del proveedor: "+err.Error())
			return
		}
	}
	u, err := pg.GetPortalUserByID(userID)
	if err != nil || u == nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	err = pg.SavePortalSSO(db.PortalSSO{
		OrgID: orgID, Issuer: req.Issuer, ClientID: req.ClientID, SecretEnc: enc, EmailDomains: domains,
		Enabled: req.Enabled, Enforce: req.Enforce, AutoProvision: req.AutoProvision,
		DefaultRole: req.DefaultRole, UpdatedBy: u.Email,
	})
	if errors.Is(err, db.ErrSSODomainTaken) {
		writeErr(w, 409, "ese dominio ya está configurado para SSO en otra organización")
		return
	}
	if err != nil {
		writeErr(w, 500, "no se pudo guardar la configuración SSO")
		return
	}
	detail := fmt.Sprintf("issuer=%s domains=[%s] enabled=%t enforce=%t auto_provision=%t default_role=%s",
		req.Issuer, strings.Join(domains, " "), req.Enabled, req.Enforce, req.AutoProvision, req.DefaultRole)
	if req.ClientSecret != "" {
		detail += " client_secret=changed"
	}
	s.audit(r, &orgID, u.Email, "portal_user", "sso.update", "org:"+strconv.FormatInt(orgID, 10), detail)
	s.writeSSOConfig(w, pg, orgID)
}

// DELETE /api/portal/sso — owner removes SSO (password sign-in applies again).
func (s *Server) portalDeleteSSO(w http.ResponseWriter, r *http.Request) {
	orgID, userID, _, ok := s.portalRequireRole(w, r, "owner")
	if !ok {
		return
	}
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	if err := pg.DeletePortalSSO(orgID); err != nil {
		writeErr(w, 500, "no se pudo eliminar la configuración SSO")
		return
	}
	email := ""
	if u, err := pg.GetPortalUserByID(userID); err == nil && u != nil {
		email = u.Email
	}
	s.audit(r, &orgID, email, "portal_user", "sso.delete", "org:"+strconv.FormatInt(orgID, 10), "")
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- sign-in -----------------------------------------------------------------

// POST /api/portal/sso/start {email} — public. Finds the org that claims the
// email's domain and returns the provider's authorization URL.
func (s *Server) portalSSOStart(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := decode(r, &req); err != nil {
		writeErr(w, 400, "cuerpo inválido")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !emailRe.MatchString(email) || len(email) > 200 {
		writeErr(w, 400, "email inválido")
		return
	}
	cfg, err := pg.FindPortalSSOByDomain(email[strings.LastIndexByte(email, '@')+1:])
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if cfg == nil {
		writeErr(w, 404, "no hay inicio de sesión único configurado para ese dominio")
		return
	}
	if s.orgSuspended(cfg.OrgID) {
		writeErr(w, 403, "organización suspendida; contacta con soporte")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	disc, err := oidcDiscover(ctx, cfg.Issuer, false)
	if err != nil {
		writeErr(w, 502, "el proveedor de identidad no responde")
		return
	}
	state, verifier, nonce := randB64(32), randB64(32), randB64(24)
	sum := sha256.Sum256([]byte(verifier))
	if err := pg.PutSSOState(hashToken(state), db.SSOState{
		OrgID: cfg.OrgID, Verifier: verifier, Nonce: nonce, RedirectURI: ssoRedirectURI(), LoginHint: email,
	}); err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {cfg.ClientID},
		"redirect_uri":          {ssoRedirectURI()},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
		"login_hint":            {email},
	}
	sep := "?"
	if strings.Contains(disc.AuthorizationEndpoint, "?") {
		sep = "&"
	}
	writeJSON(w, 200, map[string]any{"url": disc.AuthorizationEndpoint + sep + q.Encode()})
}

// POST /api/portal/sso/callback {code, state} — public. Exchanges the code
// (with the PKCE verifier and the client secret), validates the ID token and
// signs the user in to the org that started the flow.
func (s *Server) portalSSOCallback(w http.ResponseWriter, r *http.Request) {
	pg := s.portalDB(w)
	if pg == nil {
		return
	}
	var req struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}
	if err := decode(r, &req); err != nil || req.State == "" || req.Code == "" || len(req.Code) > 4096 {
		writeErr(w, 400, "inicio de sesión caducado o no válido; vuelve a intentarlo")
		return
	}
	st, err := pg.TakeSSOState(hashToken(req.State))
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if st == nil {
		writeErr(w, 400, "inicio de sesión caducado o no válido; vuelve a intentarlo")
		return
	}
	cfg, err := pg.GetPortalSSO(st.OrgID)
	if err != nil || cfg == nil || !cfg.Enabled {
		writeErr(w, 403, "el inicio de sesión único no está activo para esta organización")
		return
	}
	fail := func(reason string) {
		s.audit(r, &st.OrgID, st.LoginHint, "anonymous", "auth.sso_failed", "", reason)
		writeErr(w, 401, "no se pudo validar tu identidad con el proveedor")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	disc, err := oidcDiscover(ctx, cfg.Issuer, false)
	if err != nil {
		fail("discovery: " + err.Error())
		return
	}
	secret, err := s.decryptSecret(cfg.SecretEnc)
	if err != nil {
		fail("client secret unreadable")
		return
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {req.Code},
		"redirect_uri":  {st.RedirectURI},
		"code_verifier": {st.Verifier},
	}
	treq, err := http.NewRequest(http.MethodPost, disc.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		fail("token request")
		return
	}
	treq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	treq.Header.Set("Accept", "application/json")
	treq.SetBasicAuth(url.QueryEscape(cfg.ClientID), url.QueryEscape(secret))
	body, code, err := idpDo(ctx, treq)
	if err != nil || code != 200 {
		fail(fmt.Sprintf("token endpoint status %d", code))
		return
	}
	var tok struct {
		IDToken string `json:"id_token"`
	}
	if json.Unmarshal(body, &tok) != nil || tok.IDToken == "" {
		fail("no id_token")
		return
	}
	id, err := verifyIDToken(ctx, cfg, disc, tok.IDToken, st.Nonce)
	if err != nil {
		fail(err.Error())
		return
	}

	u, err := pg.GetPortalUserByEmail(id.Email)
	if err != nil {
		writeErr(w, 503, "almacenamiento no disponible")
		return
	}
	if u == nil {
		if !cfg.AutoProvision {
			s.audit(r, &st.OrgID, id.Email, "anonymous", "auth.sso_failed", "", "no account, auto-provisioning off")
			writeErr(w, 403, "no tienes cuenta en esta organización; pide una invitación a un administrador")
			return
		}
		uid, err := pg.CreateSSOUser(cfg.OrgID, id.Email, cfg.DefaultRole, id.Subject)
		if err != nil {
			writeErr(w, 500, "no se pudo crear la cuenta")
			return
		}
		s.audit(r, &st.OrgID, id.Email, "portal_user", "member.sso_provision",
			"user:"+strconv.FormatInt(uid, 10), "role="+cfg.DefaultRole)
		if u, err = pg.GetPortalUserByID(uid); err != nil || u == nil {
			writeErr(w, 500, "no se pudo crear la cuenta")
			return
		}
	} else {
		// An identity from this org's IdP can never open an account elsewhere.
		if u.OrgID != cfg.OrgID {
			s.audit(r, &st.OrgID, id.Email, "anonymous", "auth.sso_failed", "", "account belongs to another org")
			writeErr(w, 403, "esta cuenta pertenece a otra organización")
			return
		}
		linked, err := pg.LinkSSOSubject(u.ID, id.Subject)
		if err != nil {
			writeErr(w, 503, "almacenamiento no disponible")
			return
		}
		if !linked {
			s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.sso_failed", "", "subject mismatch")
			writeErr(w, 403, "la identidad del proveedor no coincide con la vinculada a esta cuenta")
			return
		}
	}
	if u.Disabled {
		s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login_blocked", "", "cuenta desactivada")
		writeErr(w, 403, "tu cuenta ha sido desactivada; contacta con un administrador")
		return
	}
	if s.orgSuspended(u.OrgID) {
		writeErr(w, 403, "organización suspendida; contacta con soporte")
		return
	}
	if p, perr := pg.GetOrgSecurityPolicy(u.OrgID); perr == nil && p != nil {
		if ip := policyClientIP(r); !ipAllowed(ip, p.IPAllowlist) {
			s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.ip_blocked", "user:"+strconv.FormatInt(u.ID, 10), "ip="+ip)
			writeErr(w, 403, "acceso no permitido desde esta red por la política de la organización")
			return
		}
	}
	s.audit(r, &u.OrgID, u.Email, "portal_user", "auth.login", "user:"+strconv.FormatInt(u.ID, 10), "method=sso")
	jwtTok, err := signPortalJWTWith(s.jwtKey, u.ID, u.OrgID, "sso")
	if err != nil {
		writeErr(w, 500, "no se pudo emitir el token")
		return
	}
	writeJSON(w, 200, map[string]any{"token": jwtTok})
}
