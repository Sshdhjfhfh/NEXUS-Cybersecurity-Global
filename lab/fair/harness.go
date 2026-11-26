// Package fair provides the offline NEXUS exhibition laboratory. All active
// requests are pinned to an ephemeral loopback fixture created by this package.
// It is not an arbitrary-target scanner or a production authorization adapter.
package fair

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nexus-ia/nexus/agents/blue/detectbench"
	"github.com/nexus-ia/nexus/agents/blue/sigma"
	"github.com/nexus-ia/nexus/agents/blue/telemetry"
	"github.com/nexus-ia/nexus/evolve"
	"github.com/nexus-ia/nexus/lab"
)

const asset = "10.77.0.2" // Logical fixture identity only; never dialed.
const tenant = "nexus-fair"

type Observation struct {
	ID             string    `json:"id"`
	Path           string    `json:"path"`
	Status         int       `json:"status"`
	Body           string    `json:"body"`
	SHA256         string    `json:"sha256"`
	ExpectedAttack bool      `json:"expected_attack"`
	Succeeded      bool      `json:"succeeded"`
	At             time.Time `json:"at"`
}

type Finding struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	EvidenceID  string `json:"evidence_id"`
	Remediation string `json:"remediation"`
}

type Metrics struct {
	TruePositive  int `json:"true_positive"`
	FalsePositive int `json:"false_positive"`
	TrueNegative  int `json:"true_negative"`
	FalseNegative int `json:"false_negative"`
}

type Run struct {
	ID           int              `json:"id"`
	At           time.Time        `json:"at"`
	Secured      bool             `json:"secured"`
	Target       string           `json:"target"`
	LogicalAsset string           `json:"logical_asset"`
	Observations []Observation    `json:"observations"`
	Findings     []Finding        `json:"findings"`
	Detection    telemetry.Report `json:"detection"`
	Metrics      Metrics          `json:"metrics"`
	ElapsedMS    int64            `json:"elapsed_ms"`
}

type Evaluation struct {
	Improvement evolve.Report `json:"improvement"`
	Noisy       evolve.Report `json:"noisy"`
	Limitation  string        `json:"limitation"`
}

type State struct {
	Runs       []Run       `json:"runs"`
	Evaluation *Evaluation `json:"evaluation"`
	Target     string      `json:"target"`
	Secured    bool        `json:"secured"`
}

type Harness struct {
	mu         sync.Mutex
	server     *http.Server
	base       string
	client     *http.Client
	secured    atomic.Bool
	sequence   int
	runs       []Run
	evaluation *Evaluation
}

// New starts only an application-owned synthetic target on a random loopback
// port. It does not invoke a shell, read host files or consult a proxy.
func New() (*Harness, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	h := &Harness{base: "http://" + listener.Addr().String(), runs: []Run{}}
	h.server = &http.Server{Handler: http.HandlerFunc(h.fixture), ReadHeaderTimeout: 3 * time.Second, IdleTimeout: 10 * time.Second}
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != listener.Addr().String() {
			return nil, fmt.Errorf("destination is outside the exhibition fixture")
		}
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp4", listener.Addr().String())
	}
	h.client = &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return fmt.Errorf("redirects are outside the demonstration contract")
	}}
	go h.server.Serve(listener)
	return h, nil
}

func (h *Harness) Close() error { return h.server.Close() }

func (h *Harness) fixture(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	secured := h.secured.Load()
	switch r.URL.Path {
	case "/":
		io.WriteString(w, `{"organization":"Organización de demostración","synthetic":true}`)
	case "/api/records/1":
		io.WriteString(w, `{"id":1,"owner":"visitor","data":"Registro público ficticio"}`)
	case "/api/records/2":
		if secured {
			http.Error(w, "Acceso denegado: el registro pertenece a otro usuario", 403)
			return
		}
		io.WriteString(w, `{"id":2,"owner":"admin","data":"Registro privado FICTICIO: presupuesto de laboratorio"}`)
	case "/.env":
		if secured {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "DEMO_ONLY=true\nTOKEN=FICTICIO_SIN_VALOR\n")
	default:
		http.NotFound(w, r)
	}
}

// This authorization applies exclusively to locally generated fixture events;
// it cannot authorize a production action, target or identity.
type fixtureAuthorizer struct{ task string }

func (a fixtureAuthorizer) Authorize(ctx context.Context, action lab.Action) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if action.AgentID != "fair-blue" || action.TaskID != a.task || action.Name != "blue.analyze" || action.Target != asset {
		return fmt.Errorf("only this run's synthetic fixture events are authorized")
	}
	return nil
}

func fixtureRules() []sigma.Rule {
	makeRule := func(id, title, path string) sigma.Rule {
		return sigma.Rule{ID: id, Title: title, Level: "high", Author: "NEXUS exhibition module", License: "AGPL-3.0", LogSource: sigma.LogSource{Category: "webserver"}, Detection: map[string]any{"selection": map[string]any{"path": path}, "condition": "selection"}}
	}
	return []sigma.Rule{
		makeRule("fair.config-probe", "Intento de acceso a configuración", "/.env"),
		makeRule("fair.record-probe", "Acceso al registro reservado del escenario", "/api/records/2"),
	}
}

// probe has an exact method, destination and path contract. No user-provided URL
// reaches this function through the API.
func (h *Harness) probe(ctx context.Context, path string) (int, string, error) {
	switch path {
	case "/", "/api/records/1", "/api/records/2", "/.env":
	default:
		return 0, "", fmt.Errorf("path outside the fixed scenario")
	}
	u, err := url.Parse(h.base + path)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return 0, "", err
	}
	response, err := h.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if len(body) > 4096 {
		return 0, "", fmt.Errorf("fixture response exceeded limit")
	}
	return response.StatusCode, string(body), err
}

func (h *Harness) Run(ctx context.Context, secured bool) (Run, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	started := time.Now()
	previous := h.secured.Load()
	h.secured.Store(secured)
	committed := false
	defer func() {
		if !committed {
			h.secured.Store(previous)
		}
	}()
	h.sequence++
	run := Run{ID: h.sequence, At: started.UTC(), Secured: secured, Target: h.base, LogicalAsset: asset, Findings: []Finding{}, Observations: []Observation{}}
	events := []telemetry.Event{}
	for i, path := range []string{"/", "/api/records/1", "/.env", "/api/records/2"} {
		status, body, err := h.probe(ctx, path)
		if err != nil {
			return Run{}, err
		}
		attack := i >= 2
		succeeded := attack && status == 200 && ((path == "/.env" && strings.Contains(body, "FICTICIO_SIN_VALOR")) || (path == "/api/records/2" && strings.Contains(body, `"owner":"admin"`)))
		hash := sha256.Sum256([]byte(body))
		at := time.Now().UTC()
		observation := Observation{ID: fmt.Sprintf("run-%d-http-%d", run.ID, i+1), Path: path, Status: status, Body: body, SHA256: hex.EncodeToString(hash[:]), ExpectedAttack: attack, Succeeded: succeeded, At: at}
		run.Observations = append(run.Observations, observation)
		events = append(events, telemetry.Event{TenantID: tenant, TaskID: fmt.Sprint(run.ID), ID: observation.ID, Asset: asset, ObservedAt: at, ReceivedAt: at, Source: sigma.LogSource{Category: "webserver"}, Fields: sigma.Event{"path": path, "status": status, "method": "GET"}})
		if succeeded {
			finding := Finding{ID: "F-CONFIG", Title: "Archivo de configuración expuesto", Severity: "alta", EvidenceID: observation.ID, Remediation: "Excluir archivos de configuración del contenido servido por HTTP."}
			if path == "/api/records/2" {
				finding = Finding{ID: "F-IDOR", Title: "Acceso a registro de otro usuario", Severity: "alta", EvidenceID: observation.ID, Remediation: "Comprobar en el servidor la autorización sobre cada registro."}
			}
			run.Findings = append(run.Findings, finding)
		}
	}
	engine, err := telemetry.New("fair-blue", fixtureAuthorizer{fmt.Sprint(run.ID)}, fixtureRules(), time.Minute)
	if err != nil {
		return Run{}, err
	}
	run.Detection, err = engine.Analyze(ctx, tenant, fmt.Sprint(run.ID), events)
	if err != nil {
		return Run{}, err
	}
	matched := map[string]bool{}
	for _, alert := range run.Detection.Alerts {
		for _, id := range alert.EvidenceIDs {
			matched[id] = true
		}
	}
	for _, observation := range run.Observations {
		switch {
		case observation.ExpectedAttack && matched[observation.ID]:
			run.Metrics.TruePositive++
		case observation.ExpectedAttack:
			run.Metrics.FalseNegative++
		case matched[observation.ID]:
			run.Metrics.FalsePositive++
		default:
			run.Metrics.TrueNegative++
		}
	}
	run.ElapsedMS = time.Since(started).Milliseconds()
	h.runs = append(h.runs, run)
	if len(h.runs) > 20 {
		h.runs = h.runs[len(h.runs)-20:]
	}
	committed = true
	return run, nil
}

func (h *Harness) Evaluate(ctx context.Context) (*Evaluation, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	benchmark, err := detectbench.New()
	if err != nil {
		return nil, err
	}
	run := func(candidate string) (evolve.Report, error) {
		engine, err := evolve.New(evolve.Config{Baseline: "v1", Candidates: []string{candidate}, Policy: evolve.DefaultPolicy()})
		if err != nil {
			return evolve.Report{}, err
		}
		return engine.Run(ctx, benchmark)
	}
	good, err := run("v2")
	if err != nil {
		return nil, err
	}
	noisy, err := run("noisy")
	if err != nil {
		return nil, err
	}
	h.evaluation = &Evaluation{Improvement: good, Noisy: noisy, Limitation: "Experimento independiente con eventos sintéticos y estrategias predefinidas. No reentrena una IA, no modifica las reglas de la prueba HTTP ni mide precisión en organizaciones reales."}
	return h.evaluation, nil
}

func (h *Harness) State() State {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Copy through JSON: callers cannot mutate evidence owned by the harness.
	raw, _ := json.Marshal(State{Runs: h.runs, Evaluation: h.evaluation, Target: h.base, Secured: h.secured.Load()})
	var state State
	json.Unmarshal(raw, &state)
	return state
}
