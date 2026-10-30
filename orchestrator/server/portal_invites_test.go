package server

import (
	"strings"
	"testing"
)

func TestInviteMailRendering(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		nt := notifyTextFor(lang)
		org := "Acme <b>Corp</b>"
		c := mailContent{
			Lang: lang, Subject: nt.InviteSubject(org), Title: nt.InviteTitle, Org: org,
			Intro: []string{nt.InviteIntro("admin@acme.test", org, nt.RoleName["member"])},
			CTA:   nt.InviteCTA, CTALink: "https://portal.test/portal/accept?token=abc123",
			Note: nt.InviteNote, Invite: true,
		}
		m, err := renderMail(c)
		if err != nil {
			t.Fatalf("%s: render: %v", lang, err)
		}
		for _, want := range []string{nt.FooterInvite, "accept?token=abc123", "admin@acme.test", nt.RoleName["member"]} {
			if !strings.Contains(m.Text, want) {
				t.Errorf("%s: text body lacks %q", lang, want)
			}
		}
		// An invitee has no account: no opt-out or account-security footer.
		for _, unwanted := range []string{nt.FooterOptOut, nt.FooterSecurity} {
			if strings.Contains(m.Text, unwanted) {
				t.Errorf("%s: text body has footer %q", lang, unwanted)
			}
		}
		if strings.Contains(m.HTML, "<b>Corp</b>") {
			t.Errorf("%s: organization name not escaped in HTML", lang)
		}
	}
	for lang, nt := range notifyTexts {
		for _, role := range []string{"admin", "member", "auditor"} {
			if nt.RoleName[role] == "" {
				t.Errorf("%s: no display name for role %q", lang, role)
			}
		}
		if nt.InviteIntro("", "Acme", "x") == "" || strings.Contains(nt.InviteIntro("", "Acme", "x"), "  ") {
			t.Errorf("%s: invitation intro without inviter is malformed", lang)
		}
	}
}
