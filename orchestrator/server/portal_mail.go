// Portal email: NEXUS-branded messages (HTML + plain-text alternative, logo
// inline as a CID image) and their delivery. With NEXUS_SMTP_HOST set it sends
// over SMTP; otherwise (dev) it logs the message so links are visible in the
// server log, and NEXUS_MAIL_OUTBOX_DIR additionally keeps .eml/.html copies to
// preview. This file is the single seam to plug a real provider.
//
// Security mail (password reset, email verification) is sent directly and never
// stored, because it carries one-time tokens. Notification mail goes through the
// portal_mail_outbox table (portal_notify.go) and never carries technical detail
// about findings: severity, a reference number and a link to the portal only.
package server

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func envStr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// hashToken returns the hex SHA-256 of a token — what is stored server-side. The
// plaintext token only ever travels in the email.
func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// portalLink builds an absolute (or root-relative, in dev) link into the portal.
func portalLink(path string) string {
	return strings.TrimRight(envStr("NEXUS_PORTAL_BASE_URL", ""), "/") + path
}

// consoleLink builds a link into the operator console (for NEXUS staff mail).
func consoleLink(path string) string {
	return strings.TrimRight(envStr("NEXUS_CONSOLE_BASE_URL", ""), "/") + path
}

// ---- content ----------------------------------------------------------------

// mailItem is one line of a list: a severity chip, a short text and a detail.
type mailItem struct {
	Sev, SevLabel, Text, Meta string
}

type mailSection struct {
	Title string
	Items []mailItem
}

// mailContent is what a message says; renderMail turns it into the branded
// HTML and the plain-text alternative. Every field is escaped by html/template.
type mailContent struct {
	Lang      string
	Subject   string
	Preheader string // inbox preview line
	Title     string
	Org       string // organization the message is about (optional)
	Intro     []string
	Sections  []mailSection
	CTA       string
	CTALink   string
	Note      string // small print under the button
	Security  bool   // account-security mail: cannot be switched off
	Staff     bool   // internal mail to NEXUS staff
}

// mailMessage is a rendered message ready to send.
type mailMessage struct {
	Subject string
	Text    string
	HTML    string // references the logo as cid:nexus-logo
}

const mailLogoCID = "nexus-logo"

var mailSevColor = map[string][2]string{ // text, background
	"critical": {"#BE123C", "#FFE4E6"},
	"high":     {"#C2410C", "#FFEDD5"},
	"medium":   {"#A16207", "#FEF3C7"},
	"low":      {"#0369A1", "#E0F2FE"},
	"unrated":  {"#475569", "#F1F5F9"},
}

var mailTmpl = template.Must(template.New("mail").Funcs(template.FuncMap{
	// Colours come from the fixed table above, never from input.
	"sevfg": func(s string) template.CSS { return template.CSS(mailSevColor[sevKey(s)][0]) },
	"sevbg": func(s string) template.CSS { return template.CSS(mailSevColor[sevKey(s)][1]) },
}).Parse(`<!DOCTYPE html>
<html lang="{{.C.Lang}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light"><title>{{.C.Subject}}</title></head>
<body style="margin:0;padding:0;background:#EEF2F7;-webkit-text-size-adjust:100%">
<div style="display:none;max-height:0;overflow:hidden;opacity:0">{{.C.Preheader}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#EEF2F7" style="background:#EEF2F7">
<tr><td align="center" style="padding:24px 12px">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:600px">
<tr><td bgcolor="#0B1B3A" style="background:#0B1B3A;border-radius:14px 14px 0 0;padding:22px 28px">
<img src="{{.Logo}}" width="180" height="60" alt="NEXUS Cybersecurity Global" style="display:block;border:0;width:180px;height:60px;color:#FFFFFF;font:600 18px Arial,sans-serif">
</td></tr>
<tr><td bgcolor="#22D3EE" style="background:#22D3EE;height:3px;line-height:3px;font-size:0">&nbsp;</td></tr>
<tr><td bgcolor="#FFFFFF" style="background:#FFFFFF;padding:30px 28px 8px;font-family:Arial,Helvetica,sans-serif;color:#334155;font-size:15px;line-height:1.55">
{{if .C.Org}}<p style="margin:0 0 6px;font-size:12px;letter-spacing:.06em;text-transform:uppercase;color:#0E7490;font-weight:700">{{.C.Org}}</p>{{end}}
<h1 style="margin:0 0 16px;font-size:22px;line-height:1.3;color:#0F172A">{{.C.Title}}</h1>
{{range .C.Intro}}<p style="margin:0 0 14px">{{.}}</p>{{end}}
{{range .C.Sections}}
<p style="margin:20px 0 8px;font-size:13px;font-weight:700;color:#0F172A">{{.Title}}</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border:1px solid #E2E8F0;border-radius:10px;border-collapse:separate">
{{range $i, $it := .Items}}<tr><td style="padding:10px 14px;{{if $i}}border-top:1px solid #E2E8F0;{{end}}font-size:14px">
<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:11px;font-weight:700;text-transform:uppercase;color:{{sevfg $it.Sev}};background:{{sevbg $it.Sev}}">{{$it.SevLabel}}</span>
<span style="color:#0F172A;font-weight:600">&nbsp;{{$it.Text}}</span>
{{if $it.Meta}}<br><span style="font-size:13px;color:#64748B">{{$it.Meta}}</span>{{end}}
</td></tr>{{end}}
</table>
{{end}}
{{if .C.CTA}}<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:24px 0 8px"><tr>
<td bgcolor="#0E7490" style="background:#0E7490;border-radius:8px"><a href="{{.C.CTALink}}" style="display:inline-block;padding:12px 22px;font-family:Arial,Helvetica,sans-serif;font-size:15px;font-weight:700;color:#FFFFFF;text-decoration:none">{{.C.CTA}}</a></td>
</tr></table>{{end}}
{{if .C.Note}}<p style="margin:14px 0 0;font-size:13px;color:#64748B">{{.C.Note}}</p>{{end}}
<p style="margin:22px 0 0">&nbsp;</p>
</td></tr>
<tr><td bgcolor="#F8FAFC" style="background:#F8FAFC;border-top:1px solid #E2E8F0;border-radius:0 0 14px 14px;padding:18px 28px;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.5;color:#64748B">
{{range .Footer}}<p style="margin:0 0 6px">{{.}}</p>{{end}}
</td></tr>
</table>
</td></tr></table>
</body></html>`))

// renderMail renders the branded HTML and the plain-text alternative.
func renderMail(c mailContent) (mailMessage, error) {
	t := notifyTexts[c.Lang]
	if t == nil {
		t = notifyTexts["es"]
		c.Lang = "es"
	}
	footer := []string{t.FooterAuto}
	switch {
	case c.Security:
		footer = append(footer, t.FooterSecurity)
	case c.Staff:
		footer = append(footer, t.FooterStaff)
	default:
		footer = append(footer, t.FooterMinimal, t.FooterOptOut)
	}
	var html bytes.Buffer
	if err := mailTmpl.Execute(&html, map[string]any{
		"C": c, "Footer": footer, "Logo": template.URL("cid:" + mailLogoCID),
	}); err != nil {
		return mailMessage{}, err
	}

	var txt strings.Builder
	txt.WriteString("NEXUS Cybersecurity Global\n\n")
	if c.Org != "" {
		txt.WriteString(c.Org + "\n")
	}
	txt.WriteString(c.Title + "\n\n")
	for _, p := range c.Intro {
		txt.WriteString(p + "\n\n")
	}
	for _, s := range c.Sections {
		txt.WriteString(s.Title + "\n")
		for _, it := range s.Items {
			txt.WriteString("  - [" + it.SevLabel + "] " + it.Text)
			if it.Meta != "" {
				txt.WriteString(" — " + it.Meta)
			}
			txt.WriteString("\n")
		}
		txt.WriteString("\n")
	}
	if c.CTA != "" {
		txt.WriteString(c.CTA + ": " + c.CTALink + "\n\n")
	}
	if c.Note != "" {
		txt.WriteString(c.Note + "\n\n")
	}
	txt.WriteString("--\n" + strings.Join(footer, "\n") + "\n")
	return mailMessage{Subject: c.Subject, Text: txt.String(), HTML: html.String()}, nil
}

// ---- MIME -------------------------------------------------------------------

var (
	mailLogoOnce sync.Once
	mailLogo     []byte
)

func mailLogoPNG() []byte {
	mailLogoOnce.Do(func() {
		b, err := reportAssets.ReadFile("reportassets/nexus-logo-email.png")
		if err != nil {
			log.Printf("[portal-mail] logo: %v", err)
		}
		mailLogo = b
	})
	return mailLogo
}

// previewHTML returns the HTML with the CID logo inlined as a data URI, for
// viewing the message in a browser.
func previewHTML(html string) string {
	return strings.ReplaceAll(html, "cid:"+mailLogoCID,
		"data:image/png;base64,"+base64.StdEncoding.EncodeToString(mailLogoPNG()))
}

// headerSafe folds every run of whitespace (CR and LF included) into one space,
// so a header value can never start a new header line.
func headerSafe(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func mailDomain(from string) string {
	if i := strings.LastIndex(from, "@"); i >= 0 {
		return strings.Trim(from[i+1:], "> ")
	}
	return "localhost"
}

// buildMIME assembles a multipart/related message (alternative text + HTML, and
// the inline logo).
func buildMIME(from, to string, m mailMessage, now time.Time) ([]byte, error) {
	var buf bytes.Buffer
	rel := multipart.NewWriter(&buf)
	var id [12]byte
	_, _ = rand.Read(id[:])
	hdr := []string{
		"From: " + headerSafe(from),
		"To: " + headerSafe(to),
		"Subject: " + mime.QEncoding.Encode("utf-8", headerSafe(m.Subject)),
		"Date: " + now.Format(time.RFC1123Z),
		"Message-ID: <" + hex.EncodeToString(id[:]) + "@" + headerSafe(mailDomain(from)) + ">",
		"MIME-Version: 1.0",
		"Auto-Submitted: auto-generated",
		`Content-Type: multipart/related; type="multipart/alternative"; boundary="` + rel.Boundary() + `"`,
	}
	var head bytes.Buffer
	head.WriteString(strings.Join(hdr, "\r\n") + "\r\n\r\n")

	var altBuf bytes.Buffer
	alt := multipart.NewWriter(&altBuf)
	for _, part := range []struct{ ctype, body string }{
		{"text/plain; charset=UTF-8", m.Text},
		{"text/html; charset=UTF-8", m.HTML},
	} {
		w, err := alt.CreatePart(textproto.MIMEHeader{
			"Content-Type": {part.ctype}, "Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		qp := quotedprintable.NewWriter(w)
		if _, err := qp.Write([]byte(part.body)); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
	}
	if err := alt.Close(); err != nil {
		return nil, err
	}
	w, err := rel.CreatePart(textproto.MIMEHeader{
		"Content-Type": {`multipart/alternative; boundary="` + alt.Boundary() + `"`},
	})
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(altBuf.Bytes()); err != nil {
		return nil, err
	}
	if logo := mailLogoPNG(); len(logo) > 0 {
		w, err := rel.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {`image/png; name="nexus.png"`},
			"Content-Transfer-Encoding": {"base64"},
			"Content-ID":                {"<" + mailLogoCID + ">"},
			"Content-Disposition":       {`inline; filename="nexus.png"`},
		})
		if err != nil {
			return nil, err
		}
		enc := base64.StdEncoding.EncodeToString(logo)
		for len(enc) > 76 {
			fmt.Fprintf(w, "%s\r\n", enc[:76])
			enc = enc[76:]
		}
		fmt.Fprintf(w, "%s\r\n", enc)
	}
	if err := rel.Close(); err != nil {
		return nil, err
	}
	return append(head.Bytes(), buf.Bytes()...), nil
}

// ---- delivery ---------------------------------------------------------------

// deliverMail sends one rendered message (or logs it in dev).
func deliverMail(to string, m mailMessage) error {
	to = headerSafe(to)
	from := envStr("NEXUS_SMTP_FROM", "NEXUS Cybersecurity Global <no-reply@localhost>")
	raw, err := buildMIME(from, to, m, time.Now())
	if err != nil {
		return err
	}
	if dir := envStr("NEXUS_MAIL_OUTBOX_DIR", ""); dir != "" {
		base := filepath.Join(dir, fmt.Sprintf("%d", time.Now().UnixNano()))
		_ = os.WriteFile(base+".eml", raw, 0o600)
		_ = os.WriteFile(base+".html", []byte(previewHTML(m.HTML)), 0o600)
	}
	host := envStr("NEXUS_SMTP_HOST", "")
	if host == "" {
		// Single line (text body as %q) so links are easy to find in dev logs.
		log.Printf("[portal-mail] (dev, no SMTP) to=%s subject=%q body=%q", to, m.Subject, m.Text)
		return nil
	}
	addr := host + ":" + envStr("NEXUS_SMTP_PORT", "587")
	var auth smtp.Auth
	if user := envStr("NEXUS_SMTP_USER", ""); user != "" {
		auth = smtp.PlainAuth("", user, envStr("NEXUS_SMTP_PASS", ""), host)
	}
	return smtp.SendMail(addr, auth, envelopeAddr(from), []string{to}, raw)
}

// envelopeAddr extracts the bare address from "Name <addr>".
func envelopeAddr(from string) string {
	if i, j := strings.LastIndex(from, "<"), strings.LastIndex(from, ">"); i >= 0 && j > i {
		return from[i+1 : j]
	}
	return from
}

// sendSecurityMail renders and sends account-security mail right away. It is
// never queued or stored: it carries one-time tokens.
func sendSecurityMail(to string, c mailContent) error {
	c.Security = true
	m, err := renderMail(c)
	if err != nil {
		return err
	}
	return deliverMail(to, m)
}
