package scope

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

func cmd(c string) []byte {
	b, _ := json.Marshal(map[string]string{"command": c})
	return b
}

func testPolicy() Policy {
	return New([]Rule{
		{Kind: "root_domain", Domain: "acme.com"},
		{Kind: "subdomain", Domain: "portal.client.io"},
		{Kind: "ip", Net: "203.0.113.7/32"},
		{Kind: "cidr", Net: "198.51.100.0/24"},
		{Kind: "company", Domain: "ignored"},
	})
}

func TestAllows(t *testing.T) {
	p := testPolicy()
	for host, want := range map[string]bool{
		"acme.com":           true,
		"www.acme.com":       true,
		"API.Acme.Com.":      true,
		"notacme.com":        false, // suffix match must respect the label boundary
		"acme.com.evil.net":  false,
		"portal.client.io":   true,
		"client.io":          false, // a subdomain rule does not authorize its parent
		"x.portal.client.io": false, // nor its children
		"203.0.113.7":        true,
		"203.0.113.8":        false,
		"198.51.100.200":     true,
		"10.0.0.5":           false, // private infrastructure is not implicitly in scope
		"127.0.0.1":          true,  // loopback = the agent's own machine
		"::1":                true,
		"localhost":          true,
		"::ffff:203.0.113.7": true,
		"evil.example":       false,
	} {
		if got := p.Allows(host); got != want {
			t.Errorf("Allows(%q)=%v want %v", host, got, want)
		}
	}
}

func TestViolationsShellCommands(t *testing.T) {
	p := testPolicy()
	cases := []struct {
		command string
		want    []string
	}{
		{`curl -sk https://www.acme.com/login`, nil},
		{`nmap -sV 198.51.100.0/24 -oN /tmp/scan.txt`, nil},
		{`curl https://evil.example/x`, []string{"evil.example"}},
		{`curl http://user:pw@10.1.2.3:8080/admin`, []string{"10.1.2.3"}},
		{`nmap -p- 192.0.2.10`, []string{"192.0.2.10"}},
		{`dig +short evil.example`, []string{"evil.example"}},
		{`ssh root@db.internal.net`, []string{"db.internal.net"}},
		{`sqlmap -u "https://acme.com/?id=1" --proxy=http://127.0.0.1:8788`, nil},
		// adjacent hosts must both be seen (no shared delimiter swallowing)
		{`for h in a.evil.example b.evil.example; do curl $h; done`, []string{"a.evil.example", "b.evil.example"}},
		// file and module names are not hosts
		{`python3 exploit.py > out.json && cat /etc/hosts.allow && jq . report.json`, nil},
		{`python3 -c "import os.path; print(os.path.join('a','b'))"`, nil},
		// an open redirect parameter still names the second host
		{`curl "https://acme.com/r?to=evil.example"`, []string{"evil.example"}},
		{`curl -H "Host: evil.example" https://[2001:db8::1]/`, []string{"2001:db8::1", "evil.example"}},
	}
	for _, c := range cases {
		got := p.Violations("Bash", cmd(c.command))
		sort.Strings(got)
		want := append([]string(nil), c.want...)
		sort.Strings(want)
		if len(got) == 0 && len(want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s\n  violations=%v want %v", c.command, got, want)
		}
	}
}

func TestViolationsPerTool(t *testing.T) {
	p := testPolicy()
	fetch := func(u string) []byte { b, _ := json.Marshal(map[string]string{"url": u}); return b }
	if v := p.Violations("WebFetch", fetch("https://evil.example/")); len(v) != 1 {
		t.Errorf("WebFetch out of scope not caught: %v", v)
	}
	if v := p.Violations("WebFetch", fetch("www.acme.com/robots.txt")); len(v) != 0 {
		t.Errorf("scheme-less in-scope WebFetch blocked: %v", v)
	}
	send, _ := json.Marshal(map[string]string{"text": "curl evil.example\n"})
	if v := p.Violations("shell_send", send); len(v) != 1 {
		t.Errorf("shell_send out of scope not caught: %v", v)
	}
	mcp, _ := json.Marshal(map[string]any{"target": map[string]any{"urls": []string{"https://evil.example"}}})
	if v := p.Violations("mcp__scanner__probe", mcp); len(v) != 1 {
		t.Errorf("nested MCP URL not caught: %v", v)
	}
	// A report written to disk may cite any reference; file tools are not network calls.
	write, _ := json.Marshal(map[string]string{"content": "see https://owasp.org/Top10"})
	if v := p.Violations("Write", write); v != nil {
		t.Errorf("Write must not be inspected: %v", v)
	}
	// MCP tools: bare dotted words are not treated as hosts (only URLs and IPs).
	note, _ := json.Marshal(map[string]string{"note": "parsed config.yaml and readme.txt"})
	if v := p.Violations("mcp__x__y", note); v != nil {
		t.Errorf("MCP plain text flagged: %v", v)
	}
}

func TestEmptyPolicyBlocksEveryExternalHost(t *testing.T) {
	p := New(nil)
	if !p.Empty() {
		t.Fatal("no rules must be an empty policy")
	}
	if v := p.Violations("Bash", cmd("curl https://acme.com")); len(v) != 1 {
		t.Errorf("empty locked scope must deny external hosts: %v", v)
	}
	if v := p.Violations("Bash", cmd("curl http://127.0.0.1:8788/health")); v != nil {
		t.Errorf("loopback must stay reachable: %v", v)
	}
}
