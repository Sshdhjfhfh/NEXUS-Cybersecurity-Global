package sigma

import "testing"

// Original NEXUS test rules (not derived from any third-party rule content).

const ruleAndContains = `
title: WebDAV executable download
id: 00000000-0000-0000-0000-000000000001
logsource:
    product: zeek
    service: http
detection:
    sel_webdav:
        uri|contains: webdav
    sel_exe:
        uri|endswith: .exe
    condition: sel_webdav and sel_exe
level: medium
tags:
    - attack.command-and-control
    - attack.t1105
`

const ruleOneOfThem = `
title: Sensitive path or server error
id: 00000000-0000-0000-0000-000000000002
logsource:
    category: webserver
detection:
    sel_path:
        path|contains:
            - /.git
            - /admin
    sel_err:
        status: 500
    condition: 1 of them
level: low
`

const ruleNot = `
title: Login without MFA
id: 00000000-0000-0000-0000-000000000003
logsource:
    category: authentication
detection:
    sel_login:
        event: login_success
    sel_mfa:
        mfa: "true"
    condition: sel_login and not sel_mfa
level: high
`

func mustParse(t *testing.T, y string) Rule {
	t.Helper()
	r, err := Parse([]byte(y))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return r
}

func mustMatch(t *testing.T, r Rule, ev Event) bool {
	t.Helper()
	m, err := r.Match(ev)
	if err != nil {
		t.Fatalf("match %q: %v", r.Title, err)
	}
	return m
}

func TestAndWithContainsAndEndswith(t *testing.T) {
	r := mustParse(t, ruleAndContains)
	if r.ID != "00000000-0000-0000-0000-000000000001" || len(r.Tags) != 2 {
		t.Fatalf("metadata not parsed: id=%q tags=%v", r.ID, r.Tags)
	}
	if !mustMatch(t, r, Event{"uri": "http://x/webdav/payload.exe"}) {
		t.Error("expected match for webdav + .exe")
	}
	if mustMatch(t, r, Event{"uri": "http://x/webdav/readme.txt"}) {
		t.Error("must not match without .exe")
	}
	if mustMatch(t, r, Event{"uri": "http://x/files/payload.exe"}) {
		t.Error("must not match without webdav")
	}
}

func TestOneOfThemWithListValue(t *testing.T) {
	r := mustParse(t, ruleOneOfThem)
	if !mustMatch(t, r, Event{"path": "/srv/.git/config", "status": 200}) {
		t.Error("expected match via sensitive path")
	}
	if !mustMatch(t, r, Event{"path": "/home", "status": 500}) {
		t.Error("expected match via 500 status")
	}
	if mustMatch(t, r, Event{"path": "/home", "status": 200}) {
		t.Error("must not match a benign request")
	}
}

func TestNotModifier(t *testing.T) {
	r := mustParse(t, ruleNot)
	if !mustMatch(t, r, Event{"event": "login_success", "mfa": "false"}) {
		t.Error("expected match: login without mfa")
	}
	if mustMatch(t, r, Event{"event": "login_success", "mfa": "true"}) {
		t.Error("must not match when mfa present")
	}
}

func TestParseRejectsMissingCondition(t *testing.T) {
	_, err := Parse([]byte("title: x\ndetection:\n  sel:\n    a: b\n"))
	if err == nil {
		t.Error("expected error for rule without condition")
	}
}
