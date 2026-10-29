package server

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRenderMailEscapesContent(t *testing.T) {
	m, err := renderMail(mailContent{
		Lang: "es", Subject: "x", Title: `<script>alert(1)</script>`, Org: `"><img src=x onerror=alert(1)>`,
		Intro:    []string{"a & b"},
		Sections: []mailSection{{Title: "s", Items: []mailItem{{Sev: "critical", SevLabel: "Crítica", Text: "<b>x</b>"}}}},
		CTA:      "Abrir", CTALink: "javascript:alert(1)",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"<script>", "<img src=x", "<b>x</b>", "javascript:"} {
		if strings.Contains(m.HTML, bad) {
			t.Errorf("HTML contains unescaped %q", bad)
		}
	}
	if !strings.Contains(m.HTML, "&lt;script&gt;") || !strings.Contains(m.HTML, "a &amp; b") {
		t.Error("expected escaped text in HTML")
	}
	if !strings.Contains(m.HTML, "color:#BE123C") {
		t.Error("severity colour missing or escaped")
	}
	if !strings.Contains(m.Text, "<script>alert(1)</script>") {
		t.Error("plain-text part should carry the raw text")
	}
}

func TestBuildMIMEHeadersAreSafe(t *testing.T) {
	msg := mailMessage{Subject: "Contraseña\r\nBcc: victim@example.org", Text: "hola ñ", HTML: "<p>hola ñ</p>"}
	raw, err := buildMIME("NEXUS <no-reply@nexus.example>", "user@example-org.test\r\nCc: x@example.org", msg,
		time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	head := string(raw[:bytes.Index(raw, []byte("\r\n\r\n"))])
	for _, bad := range []string{"\r\nBcc:", "\r\nCc:"} {
		if strings.Contains(head, bad) {
			t.Fatalf("header injection: %q in\n%s", bad, head)
		}
	}
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subj, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
	if err != nil || subj != "Contraseña Bcc: victim@example.org" {
		t.Fatalf("subject = %q (%v)", subj, err)
	}
	if !strings.HasPrefix(m.Header.Get("Subject"), "=?utf-8?q?") {
		t.Errorf("subject not RFC 2047 encoded: %q", m.Header.Get("Subject"))
	}
	for _, h := range []string{"Date", "Message-ID", "Auto-Submitted"} {
		if m.Header.Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
	mt, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/related" {
		t.Fatalf("content-type %q %v", mt, err)
	}
	r := multipart.NewReader(m.Body, params["boundary"])
	alt, err := r.NextPart()
	if err != nil || !strings.HasPrefix(alt.Header.Get("Content-Type"), "multipart/alternative") {
		t.Fatalf("first part: %v %v", alt, err)
	}
	_, ap, _ := mime.ParseMediaType(alt.Header.Get("Content-Type"))
	ar := multipart.NewReader(alt, ap["boundary"])
	var types []string
	for {
		p, err := ar.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(p) // multipart decodes quoted-printable
		if !strings.Contains(string(body), "hola ñ") {
			t.Errorf("%s body = %q", p.Header.Get("Content-Type"), body)
		}
		types = append(types, p.Header.Get("Content-Type"))
	}
	if len(types) != 2 || !strings.HasPrefix(types[0], "text/plain") || !strings.HasPrefix(types[1], "text/html") {
		t.Fatalf("alternative parts = %v", types)
	}
	logo, err := r.NextPart()
	if err != nil || logo.Header.Get("Content-Id") != "<nexus-logo>" {
		t.Fatalf("logo part: %v %v", logo, err)
	}
}

func TestNotifyTextsComplete(t *testing.T) {
	for lang, nt := range notifyTexts {
		v := reflect.ValueOf(*nt)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.IsZero() {
				t.Errorf("%s: %s is empty", lang, v.Type().Field(i).Name)
			}
		}
	}
	es := notifyTextFor("es")
	if got := es.DigestSubject(1, 0); got != "Remediación: 1 hallazgo fuera de plazo · NEXUS" {
		t.Errorf("digest subject = %q", got)
	}
	if got := es.AssignedIntro("ana@example.org", es.FindingRef(7), es.sevName("critical"), "Web Q4"); got !=
		"ana@example.org te ha asignado el hallazgo #7 (severidad crítica) de la evaluación «Web Q4»." {
		t.Errorf("assigned intro = %q", got)
	}
	if notifyTextFor("fr").Lang != "es" {
		t.Error("unknown language must fall back to Spanish")
	}
}

// TestRenderMailSamples writes one preview of each email to NEXUS_MAIL_SAMPLES
// for visual review.
func TestRenderMailSamples(t *testing.T) {
	dir := os.Getenv("NEXUS_MAIL_SAMPLES")
	for _, lang := range []string{"es", "en"} {
		nt := notifyTextFor(lang)
		ref := nt.FindingRef(1042)
		samples := map[string]mailContent{
			"released": {Subject: nt.ReleasedSubject("Evaluación web Q4"), Title: nt.ReleasedTitle, Org: "Ministerio de Ejemplo",
				Intro: []string{nt.ReleasedIntro("Evaluación web Q4")}, CTA: nt.ReleasedCTA, CTALink: "/portal?tab=requests&request=42",
				Note: nt.ReleasedNote},
			"assigned": {Subject: nt.AssignedSubject(1042), Title: nt.AssignedTitle, Org: "Ministerio de Ejemplo",
				Intro: []string{nt.AssignedIntro("ana@example-org.test", ref, nt.sevName("high"), "Evaluación web Q4"),
					nt.DueLine(nt.dateStr("2026-11-27"))}, CTA: nt.OpenCTA, CTALink: "/portal?tab=remediation&finding=1042"},
			"fixed": {Subject: nt.FixedSubject(1042), Title: nt.FixedTitle, Org: "Ministerio de Ejemplo",
				Intro: []string{nt.FixedIntro(ref, nt.sevName("critical"), "Evaluación web Q4")}, CTA: nt.OpenCTA, CTALink: "/portal"},
			"digest": {Subject: nt.DigestSubject(2, 1), Title: nt.DigestTitle, Org: "Ministerio de Ejemplo",
				Intro: []string{nt.DigestIntro}, CTA: nt.DigestCTA, CTALink: "/portal?tab=remediation",
				Sections: []mailSection{
					{Title: nt.OverdueSection, Items: []mailItem{
						{Sev: "critical", SevLabel: nt.sevLabel("critical"), Text: nt.ItemText(ref, "Evaluación web Q4"), Meta: nt.OverdueMeta(nt.dateStr("2026-10-25"), 3)},
						{Sev: "medium", SevLabel: nt.sevLabel("medium"), Text: nt.ItemText(nt.FindingRef(1077), "API pública"), Meta: nt.OverdueMeta(nt.dateStr("2026-10-27"), 1)},
					}},
					{Title: nt.ExpiringSection, Items: []mailItem{
						{Sev: "low", SevLabel: nt.sevLabel("low"), Text: nt.ItemText(nt.FindingRef(1090), "API pública"), Meta: nt.ExpiresMeta(nt.dateStr("2026-11-02"))},
					}},
				}},
			"reset": {Subject: nt.ResetSubject, Title: nt.ResetTitle, Intro: []string{nt.ResetIntro}, CTA: nt.ResetCTA,
				CTALink: "/portal/reset?token=example", Note: nt.ResetNote, Security: true},
		}
		for name, c := range samples {
			c.Lang = lang
			m, err := renderMail(c)
			if err != nil {
				t.Fatalf("%s/%s: %v", name, lang, err)
			}
			if strings.Contains(m.HTML, "Inyección") || strings.Contains(m.Text, "SQL") {
				t.Fatalf("%s/%s leaks finding detail", name, lang)
			}
			if dir != "" {
				if err := os.WriteFile(filepath.Join(dir, "mail-"+name+"-"+lang+".html"), []byte(previewHTML(m.HTML)), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
