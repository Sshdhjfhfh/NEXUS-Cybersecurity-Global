// Pluggable email delivery for the portal (password reset, email verification).
// With NEXUS_SMTP_HOST set it sends over SMTP; otherwise (dev) it logs the
// message so the link/token is visible in the server log. This file is the single
// seam to plug a real provider — no other code knows how mail is delivered.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
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

// sendPortalMail delivers an email, or logs it in dev when no SMTP is configured.
func sendPortalMail(to, subject, body string) error {
	host := envStr("NEXUS_SMTP_HOST", "")
	if host == "" {
		// Single line (body as %q) so the link/token is easy to find in dev logs.
		log.Printf("[portal-mail] (dev, no SMTP) to=%s subject=%q body=%q", to, subject, body)
		return nil
	}
	addr := host + ":" + envStr("NEXUS_SMTP_PORT", "587")
	from := envStr("NEXUS_SMTP_FROM", "no-reply@localhost")
	var auth smtp.Auth
	if user := envStr("NEXUS_SMTP_USER", ""); user != "" {
		auth = smtp.PlainAuth("", user, envStr("NEXUS_SMTP_PASS", ""), host)
	}
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
