// Command nexus-detect runs NEXUS passive Sigma detection over a batch of
// telemetry events and prints the alerts (JSON or Markdown).
//
// SAFETY: it is PASSIVE. It never opens a connection to an asset and never
// applies containment. Every event's asset is authorized against the signed
// engagement scope before it is analyzed. This is the Blue-team counterpart of
// nexus-scan for the services model (detection engineering / log triage).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/agents/blue/sigma"
	"github.com/nexus-ia/nexus/agents/blue/telemetry"
	"github.com/nexus-ia/nexus/lab/engagement"
)

// identifier mirrors the telemetry engine's rule/tenant id constraint so we can
// drop rules it would reject before constructing the engine.
var identifier = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)

func main() {
	scopePath := flag.String("scope", "", "path to engagement scope JSON (required)")
	rulesDir := flag.String("rules", "", "path to a directory of Sigma rules (required)")
	eventsPath := flag.String("events", "", "path to a JSONL file of telemetry events, one per line (required)")
	tenant := flag.String("tenant", "", "tenant id for this detection request (required)")
	task := flag.String("task", "", "numeric task id for this detection request (required)")
	windowStr := flag.String("window", "5m", "temporal aggregation window, e.g. 5m, 1h")
	maxRules := flag.Int("max-rules", 1000, "maximum rules to load into the engine (engine hard cap is 1000)")
	format := flag.String("format", "json", "report format: json or md")
	flag.Parse()

	if *scopePath == "" || *rulesDir == "" || *eventsPath == "" || *tenant == "" || *task == "" {
		fmt.Fprintln(os.Stderr, "usage: nexus-detect -scope scope.json -rules DIR -events events.jsonl -tenant T -task N [-window 5m] [-format json|md]")
		os.Exit(2)
	}

	window, err := time.ParseDuration(*windowStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "window:", err)
		os.Exit(1)
	}

	scope, err := engagement.LoadScope(*scopePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope:", err)
		os.Exit(1)
	}
	auth, err := engagement.NewAuthorizer(scope)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scope:", err)
		os.Exit(1)
	}

	rules, skipped := loadRules(*rulesDir, *maxRules)
	if len(rules) == 0 {
		fmt.Fprintln(os.Stderr, "rules: no usable Sigma rules loaded from", *rulesDir)
		os.Exit(1)
	}

	engine, err := telemetry.New("nexus-detect", auth, rules, window)
	if err != nil {
		fmt.Fprintln(os.Stderr, "engine:", err)
		os.Exit(1)
	}

	events, err := loadEvents(*eventsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "events:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rep, err := engine.Analyze(ctx, *tenant, *task, events)
	if err != nil {
		fmt.Fprintln(os.Stderr, "analyze:", err)
		os.Exit(1)
	}

	switch *format {
	case "md":
		fmt.Print(markdown(scope, rep, len(rules), skipped))
	default:
		out, _ := json.MarshalIndent(struct {
			Engagement  string           `json:"engagement"`
			Client      string           `json:"client,omitempty"`
			RulesLoaded int              `json:"rules_loaded"`
			RulesSkip   int              `json:"rules_skipped"`
			Report      telemetry.Report `json:"report"`
		}{scope.Engagement, scope.Client, len(rules), skipped, rep}, "", "  ")
		fmt.Println(string(out))
	}
}

// loadRules loads a Sigma directory and keeps only rules the engine accepts:
// valid id, non-empty logsource, unique id, and successfully compilable. It
// returns the kept rules (capped at max) and the number skipped.
func loadRules(dir string, max int) ([]sigma.Rule, int) {
	parsed, _ := sigma.LoadDir(dir)
	var kept []sigma.Rule
	seen := map[string]bool{}
	skipped := 0
	for _, r := range parsed {
		if len(kept) >= max {
			skipped++
			continue
		}
		ls := normLogSource(r.LogSource)
		if !identifier.MatchString(r.ID) || seen[r.ID] || ls == (sigma.LogSource{}) {
			skipped++
			continue
		}
		if _, err := r.Compile(); err != nil {
			skipped++
			continue
		}
		seen[r.ID] = true
		kept = append(kept, r)
	}
	return kept, skipped
}

func normLogSource(s sigma.LogSource) sigma.LogSource {
	return sigma.LogSource{
		Product:  strings.ToLower(strings.TrimSpace(s.Product)),
		Service:  strings.ToLower(strings.TrimSpace(s.Service)),
		Category: strings.ToLower(strings.TrimSpace(s.Category)),
	}
}

// loadEvents reads one JSON telemetry event per non-empty line.
func loadEvents(path string) ([]telemetry.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []telemetry.Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), telemetry.MaxEventBytes)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		ev, err := telemetry.Decode([]byte(raw))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		events = append(events, ev)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func markdown(scope engagement.Scope, r telemetry.Report, rulesLoaded, rulesSkipped int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Informe de detección — %s\n\n", scope.Engagement)
	if scope.Client != "" {
		fmt.Fprintf(&b, "**Cliente:** %s  \n", scope.Client)
	}
	fmt.Fprintf(&b, "**Reglas cargadas:** %d (omitidas %d)  \n", rulesLoaded, rulesSkipped)
	fmt.Fprintf(&b, "**Eventos:** %d entrada / %d únicos / %d duplicados / %d con coincidencia  \n",
		r.InputEvents, r.UniqueEvents, r.DuplicateEvents, r.MatchedEvents)
	fmt.Fprintf(&b, "**Huella de evidencia:** `%s`\n\n", r.EvidenceDigest)

	fmt.Fprintf(&b, "## Alertas (%d)\n\n", len(r.Alerts))
	if len(r.Alerts) == 0 {
		fmt.Fprintf(&b, "_Sin coincidencias de las reglas cargadas._\n")
		return b.String()
	}
	fmt.Fprintf(&b, "| Severidad | Regla | Título | ATT&CK | Activo | Eventos | Ventana |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | --- | --- | --- |\n")
	alerts := append([]telemetry.Alert(nil), r.Alerts...)
	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Severity != alerts[j].Severity {
			return sevRank(alerts[i].Severity) > sevRank(alerts[j].Severity)
		}
		return alerts[i].RuleID < alerts[j].RuleID
	})
	for _, a := range alerts {
		att := strings.Join(a.Techniques, ", ")
		if att == "" {
			att = "—"
		}
		win := a.LastSeen.Sub(a.FirstSeen).String()
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %d | %s |\n",
			strings.ToUpper(nonEmpty(a.Severity)), a.RuleID, a.Title, att, a.Asset, len(a.EvidenceIDs), win)
	}
	return b.String()
}

func sevRank(s string) int {
	switch strings.ToLower(s) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func nonEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "info"
	}
	return s
}
