package server

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func cspHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

func TestWebuiCSPHashesOnlyThePagesOwnInlineScripts(t *testing.T) {
	boot := `self.__next_f.push([1,"x"])`
	page := `<html><head><script src="/_next/static/chunks/a.js" async=""></script>` +
		`<script>` + boot + `</script><SCRIPT type="text/javascript">document.documentElement.dataset.t="d"</SCRIPT>` +
		`<script>` + boot + `</script></head></html>`
	csp := webuiCSPFor([]byte(page))
	for _, want := range []string{
		"default-src 'self'", "frame-ancestors 'none'", "object-src 'none'", "connect-src 'self'",
		cspHash(boot), cspHash(`document.documentElement.dataset.t="d"`),
	} {
		if !strings.Contains(csp, want) {
			t.Fatalf("csp lacks %q:\n%s", want, csp)
		}
	}
	if strings.Count(csp, cspHash(boot)) != 1 {
		t.Fatalf("duplicate hash in %s", csp)
	}
	if strings.Contains(csp, "unsafe-inline' ;") || strings.Contains(strings.SplitN(csp, "script-src", 2)[1], "unsafe") {
		t.Fatalf("script-src must not allow unsafe inline/eval: %s", csp)
	}
	if strings.Contains(csp, cspHash("")) {
		t.Fatalf("external script body hashed: %s", csp)
	}
}

func TestWebuiHeaders(t *testing.T) {
	h := &webuiHeaders{fsys: fstest.MapFS{
		"portal/index.html": {Data: []byte(`<script>a()</script>`)},
	}}
	w := httptest.NewRecorder()
	h.set(w, "portal/index.html")
	hd := w.Header()
	if hd.Get("X-Frame-Options") != "DENY" || hd.Get("X-Content-Type-Options") != "nosniff" ||
		hd.Get("Referrer-Policy") != "no-referrer" || hd.Get("Cache-Control") != "no-cache" {
		t.Fatalf("html headers = %v", hd)
	}
	if csp := hd.Get("Content-Security-Policy"); !strings.Contains(csp, cspHash("a()")) {
		t.Fatalf("csp = %q", csp)
	}
	w = httptest.NewRecorder()
	h.set(w, "_next/static/chunks/x.js")
	if w.Header().Get("Content-Security-Policy") != "" || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset headers = %v", w.Header())
	}
	w = httptest.NewRecorder()
	h.set(w, "missing.html")
	if csp := w.Header().Get("Content-Security-Policy"); !strings.HasSuffix(csp, "script-src 'self'") {
		t.Fatalf("missing page csp = %q", csp)
	}
}
