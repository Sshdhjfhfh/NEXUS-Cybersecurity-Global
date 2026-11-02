// Package scope enforces a task's authorized target list on agent tool calls.
//
// When an engagement's scope is locked (e.g. a client request promoted from the
// portal with verified targets), every network-capable tool call is inspected
// for the hosts it names, and any host outside the authorized list blocks the
// call. This is defense in depth against an agent wandering off scope — the
// most likely real-world incident. It is NOT a network boundary: a command can
// hide its destination (a script file, an encoded string, a bare hostname
// without a dot). Hard enforcement needs egress control at the network layer.
package scope

import (
	"encoding/json"
	"net/netip"
	"regexp"
	"strings"
)

// Rule is one authorized target, mirroring a task_scope row.
type Rule struct {
	Kind   string // root_domain | subdomain | ip | cidr
	Domain string // root_domain / subdomain
	Net    string // ip / cidr, CIDR notation
}

// Policy is the compiled authorized-target set of one task.
type Policy struct {
	roots []string
	hosts map[string]bool
	nets  []netip.Prefix
}

// New compiles rules. Kinds that do not name a host (company, icp, keyword) are
// ignored: they cannot authorize a concrete destination.
func New(rules []Rule) Policy {
	p := Policy{hosts: map[string]bool{}}
	for _, r := range rules {
		switch r.Kind {
		case "root_domain":
			if d := normHost(r.Domain); d != "" {
				p.roots = append(p.roots, d)
			}
		case "subdomain":
			if d := normHost(r.Domain); d != "" {
				p.hosts[d] = true
			}
		case "ip", "cidr":
			if pfx, err := netip.ParsePrefix(strings.TrimSpace(r.Net)); err == nil {
				p.nets = append(p.nets, pfx.Masked())
			}
		}
	}
	return p
}

// Empty reports whether the policy authorizes no destination at all.
func (p Policy) Empty() bool { return len(p.roots) == 0 && len(p.hosts) == 0 && len(p.nets) == 0 }

// Allows reports whether host (a hostname or IP literal) is authorized.
// Loopback is always allowed: it is the agent's own machine (local tools, the
// recording proxy), never a client target.
func (p Policy) Allows(host string) bool {
	h := normHost(host)
	if h == "" {
		return true
	}
	if ip, err := netip.ParseAddr(h); err == nil {
		ip = ip.Unmap()
		if ip.IsLoopback() {
			return true
		}
		for _, n := range p.nets {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	if p.hosts[h] {
		return true
	}
	for _, r := range p.roots {
		if h == r || strings.HasSuffix(h, "."+r) {
			return true
		}
	}
	return false
}

// shellTools carry free-form command text, so bare hostnames are checked too.
var shellTools = map[string]bool{"Bash": true, "shell_open": true, "shell_send": true}

// Inspected reports whether a tool's calls are checked. File, planning and
// record-keeping tools are not: their text may legitimately mention any host
// (a report citing a reference URL) without contacting it.
func Inspected(tool string) bool {
	return shellTools[tool] || tool == "WebFetch" || strings.HasPrefix(tool, "mcp__")
}

// Violations returns the distinct hosts named by an inspected tool call that the
// policy does not authorize (nil when the call is allowed).
func (p Policy) Violations(tool string, input []byte) []string {
	if !Inspected(tool) {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, h := range Targets(tool, input) {
		if seen[h] || p.Allows(h) {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	return out
}

var (
	reURLAuthority = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.\-]*://([^/\s'"<>\\?#]+)`)
	reToken        = regexp.MustCompile(`[A-Za-z0-9._\-]+`)
	reDomain       = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9\-]{1,62}$`)
)

// fileSuffixes are dotted tokens in shell text that are almost always file or
// module names rather than hosts ("out.json", "os.path"). A host under one of
// these TLDs written bare in a command is missed; written as a URL it is not.
var fileSuffixes = map[string]bool{
	"json": true, "txt": true, "py": true, "sh": true, "html": true, "htm": true, "xml": true,
	"js": true, "ts": true, "mjs": true, "csv": true, "log": true, "md": true, "yaml": true,
	"yml": true, "go": true, "php": true, "conf": true, "cfg": true, "ini": true, "pem": true,
	"key": true, "crt": true, "zip": true, "gz": true, "tgz": true, "tar": true, "out": true,
	"lst": true, "list": true, "db": true, "sql": true, "jpg": true, "jpeg": true, "png": true,
	"gif": true, "svg": true, "pdf": true, "bin": true, "exe": true, "dll": true, "so": true,
	"rb": true, "pl": true, "java": true, "class": true, "jar": true, "c": true, "h": true,
	"cpp": true, "rs": true, "toml": true, "lock": true, "env": true, "bak": true, "tmp": true,
	"old": true, "swp": true, "pcap": true, "har": true, "ps1": true, "bat": true, "cmd": true,
	"path": true, "dumps": true, "loads": true, "get": true, "post": true, "argv": true,
	"stdout": true, "stderr": true, "stdin": true, "join": true, "read": true, "write": true,
	"split": true, "strip": true, "items": true, "keys": true, "values": true, "format": true,
	"request": true, "response": true, "text": true, "content": true, "status": true,
}

// Targets extracts the destination hosts a tool call names. URL authorities and
// IPv4 literals are taken from any inspected tool; bare dotted hostnames only
// from shell command text.
func Targets(tool string, input []byte) []string {
	text := inputText(tool, input)
	var out []string
	for _, m := range reURLAuthority.FindAllStringSubmatch(text, -1) {
		if h := hostOfAuthority(m[1]); h != "" {
			out = append(out, h)
		}
	}
	for _, loc := range reToken.FindAllStringIndex(text, -1) {
		tok := strings.Trim(text[loc[0]:loc[1]], ".-_")
		if tok == "" {
			continue
		}
		if ip, err := netip.ParseAddr(tok); err == nil && ip.Is4() {
			out = append(out, ip.String())
			continue
		}
		// A token right after a path separator is a file name ("/tmp/a.json").
		if !shellTools[tool] || (loc[0] > 0 && (text[loc[0]-1] == '/' || text[loc[0]-1] == '\\')) {
			continue
		}
		d := strings.ToLower(tok)
		if !reDomain.MatchString(d) || fileSuffixes[d[strings.LastIndexByte(d, '.')+1:]] {
			continue
		}
		out = append(out, d)
	}
	return out
}

// inputText returns the text to scan: the command for shell tools, the URL for
// WebFetch, and every string value for anything else (MCP tools).
func inputText(tool string, input []byte) string {
	var m map[string]any
	if json.Unmarshal(input, &m) != nil {
		return string(input)
	}
	switch {
	case tool == "Bash" || tool == "shell_open":
		s, _ := m["command"].(string)
		return s
	case tool == "shell_send":
		s, _ := m["text"].(string)
		return s
	case tool == "WebFetch":
		s, _ := m["url"].(string)
		if s != "" && !strings.Contains(s, "://") {
			s = "https://" + s
		}
		return s
	}
	var b strings.Builder
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			b.WriteString(x)
			b.WriteByte('\n')
		case map[string]any:
			for _, e := range x {
				walk(e)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(m)
	return b.String()
}

// hostOfAuthority strips userinfo, port and IPv6 brackets from a URL authority.
func hostOfAuthority(a string) string {
	if i := strings.LastIndexByte(a, '@'); i >= 0 {
		a = a[i+1:]
	}
	if strings.HasPrefix(a, "[") {
		if j := strings.IndexByte(a, ']'); j > 0 {
			return normHost(a[1:j])
		}
		return ""
	}
	if i := strings.LastIndexByte(a, ':'); i >= 0 {
		a = a[:i]
	}
	return normHost(a)
}

func normHost(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	h = strings.TrimSuffix(h, ".")
	h = strings.Trim(h, "[]")
	return h
}
