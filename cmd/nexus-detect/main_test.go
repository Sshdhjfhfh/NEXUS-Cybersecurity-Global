package main

import (
	"os"
	"path/filepath"
	"testing"
)

const validRule = `title: Suspicious PowerShell Encoded Command
id: lab-ps-encoded
level: high
tags:
  - attack.t1059.001
logsource:
  category: process_creation
detection:
  sel:
    Image|endswith: \powershell.exe
    CommandLine|contains: ' -enc '
  condition: sel
`

// missing logsource → the engine would reject it, so loadRules must skip it.
const noLogsourceRule = `title: No Logsource
id: lab-nols
level: low
detection:
  sel:
    Image|endswith: \x.exe
  condition: sel
`

func TestLoadRulesKeepsValidSkipsUnusable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.yml"), []byte(validRule), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.yml"), []byte(noLogsourceRule), 0o600); err != nil {
		t.Fatal(err)
	}
	kept, skipped := loadRules(dir, 1000)
	if len(kept) != 1 {
		t.Fatalf("expected 1 usable rule, got %d", len(kept))
	}
	if kept[0].ID != "lab-ps-encoded" {
		t.Errorf("kept wrong rule: %q", kept[0].ID)
	}
	if skipped < 1 {
		t.Errorf("expected the no-logsource rule to be skipped, skipped=%d", skipped)
	}
}

func TestLoadRulesRespectsMax(t *testing.T) {
	dir := t.TempDir()
	// Two valid rules with distinct ids; max=1 must keep only one.
	if err := os.WriteFile(filepath.Join(dir, "a.yml"), []byte(validRule), 0o600); err != nil {
		t.Fatal(err)
	}
	second := `title: Recon
id: lab-whoami
level: low
logsource:
  category: process_creation
detection:
  sel:
    Image|endswith: \whoami.exe
  condition: sel
`
	if err := os.WriteFile(filepath.Join(dir, "b.yml"), []byte(second), 0o600); err != nil {
		t.Fatal(err)
	}
	kept, skipped := loadRules(dir, 1)
	if len(kept) != 1 || skipped < 1 {
		t.Fatalf("max=1 should keep 1 and skip the rest: kept=%d skipped=%d", len(kept), skipped)
	}
}

func TestLoadEventsParsesJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	content := `{"tenant_id":"acme","task_id":"1","id":"evt-1","asset":"10.0.0.5","observed_at":"2026-10-26T10:00:00Z","received_at":"2026-10-26T10:00:01Z","source":{"category":"process_creation"},"fields":{"Image":"C:\\Windows\\powershell.exe","CommandLine":"powershell.exe -enc AA=="}}

`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	events, err := loadEvents(path)
	if err != nil {
		t.Fatalf("loadEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (blank line ignored), got %d", len(events))
	}
	if events[0].ID != "evt-1" || events[0].Asset != "10.0.0.5" {
		t.Errorf("event parsed wrong: %+v", events[0])
	}
}

func TestLoadEventsRejectsBadLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	if err := os.WriteFile(path, []byte("{not json}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEvents(path); err == nil {
		t.Fatal("expected error on malformed event line")
	}
}
