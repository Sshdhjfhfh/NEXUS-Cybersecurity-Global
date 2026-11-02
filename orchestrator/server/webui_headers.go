package server

import (
	"crypto/sha256"
	"encoding/base64"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// Security headers for the embedded web UI (portal and console). In production
// the frontend is a static export served by this binary, so the per-request
// nonce CSP of web/src/proxy.ts never runs; these headers are what the browser
// gets. Static pages cannot carry a nonce, so each HTML file's CSP allows its
// own inline scripts by SHA-256 hash (computed once per file) and nothing else
// inline: an injected script has no matching hash and does not run.

// webuiCSPBase mirrors the production policy in web/src/proxy.ts.
var webuiCSPBase = []string{
	"default-src 'self'",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data: blob:",
	"media-src 'self'",
	"font-src 'self' data:",
	"connect-src 'self'",
	"frame-ancestors 'none'",
	"base-uri 'self'",
	"form-action 'self'",
	"object-src 'none'",
}

var inlineScriptRe = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)
var scriptSrcAttrRe = regexp.MustCompile(`(?i)\ssrc\s*=`)

// webuiCSPFor builds the policy for one HTML document.
func webuiCSPFor(html []byte) string {
	seen := map[string]bool{}
	script := []string{"script-src 'self'"}
	for _, m := range inlineScriptRe.FindAllSubmatch(html, -1) {
		if scriptSrcAttrRe.Match(m[1]) || len(m[2]) == 0 {
			continue // external script: covered by 'self'
		}
		sum := sha256.Sum256(m[2])
		h := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		if !seen[h] {
			seen[h] = true
			script = append(script, h)
		}
	}
	return strings.Join(append(append([]string{}, webuiCSPBase...), strings.Join(script, " ")), "; ")
}

// webuiHeaders sets the headers for one embedded file. HTML gets its CSP (cached
// per file) and is revalidated on every visit so a new release is picked up;
// content-hashed assets under _next/static are cached for a year.
type webuiHeaders struct {
	fsys fs.FS
	csp  sync.Map // file name -> policy
}

func (h *webuiHeaders) set(w http.ResponseWriter, name string) {
	hd := w.Header()
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("X-Frame-Options", "DENY")
	hd.Set("Referrer-Policy", "no-referrer")
	hd.Set("Cross-Origin-Opener-Policy", "same-origin")
	hd.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	switch {
	case strings.HasSuffix(name, ".html"):
		hd.Set("Content-Security-Policy", h.policy(name))
		hd.Set("Cache-Control", "no-cache")
	case strings.HasPrefix(name, "_next/static/"):
		hd.Set("Cache-Control", "public, max-age=31536000, immutable")
	}
}

func (h *webuiHeaders) policy(name string) string {
	if v, ok := h.csp.Load(name); ok {
		return v.(string)
	}
	b, err := fs.ReadFile(h.fsys, name)
	if err != nil {
		// Unreadable page: the strictest policy (no inline script at all).
		return webuiCSPFor(nil)
	}
	p := webuiCSPFor(b)
	h.csp.Store(name, p)
	return p
}
