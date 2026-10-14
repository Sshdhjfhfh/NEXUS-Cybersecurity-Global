package server

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/netip"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/agents/blue"
	"github.com/nexus-ia/nexus/lab/network"
	"github.com/nexus-ia/nexus/lab/security"
	"github.com/nexus-ia/nexus/orchestrator/level12"
	"github.com/nexus-ia/nexus/swarm"
)

// ConfigureLevel12 is a host composition hook. HTTP clients cannot install or bypass guards.
// A nil gate exposes history and readiness while keeping analysis and feedback closed.
func (s *Server) ConfigureLevel12(cfg level12.Config) error {
	cfg.Repository = s.m.pg.Level12()
	if s.m.traffic != nil {
		cfg.Source = &recordedLabSource{m: s.m}
	} else {
		cfg.Source = nil
	}
	service, err := level12.New(cfg)
	if err != nil {
		return err
	}
	s.labMu.Lock()
	s.labService = service
	s.labMu.Unlock()
	return nil
}
func (s *Server) laboratory() *level12.Service {
	s.labMu.RLock()
	defer s.labMu.RUnlock()
	return s.labService
}
func (s *Server) registerLabRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/lab/status", s.labStatus)
	mux.HandleFunc("GET /api/lab/tasks/{id}/runs", s.labRuns)
	mux.HandleFunc("POST /api/lab/tasks/{id}/runs", s.labAnalyze)
	mux.HandleFunc("GET /api/lab/tasks/{id}/runs/{runID}", s.labRun)
	mux.HandleFunc("GET /api/lab/tasks/{id}/predictions", s.labPredictions)
	mux.HandleFunc("POST /api/lab/tasks/{id}/runs/{runID}/feedback", s.labFeedback)
}
func (s *Server) labStatus(w http.ResponseWriter, r *http.Request) {
	service := s.laboratory()
	if service == nil {
		writeErr(w, 503, "Laboratorio no disponible")
		return
	}
	writeJSON(w, 200, service.Status())
}
func (s *Server) labTask(w http.ResponseWriter, r *http.Request) (*level12.Service, int64, bool) {
	service := s.laboratory()
	if service == nil {
		writeErr(w, 503, "Laboratorio no disponible")
		return nil, 0, false
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, "Identificador de tarea inválido")
		return nil, 0, false
	}
	task, err := s.m.pg.GetTask(id)
	if err != nil {
		labError(w, err)
		return nil, 0, false
	}
	if task == nil {
		writeErr(w, 404, "Tarea no encontrada")
		return nil, 0, false
	}
	return service, id, true
}
func (s *Server) labRuns(w http.ResponseWriter, r *http.Request) {
	service, id, ok := s.labTask(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var err error
		limit, err = strconv.Atoi(raw)
		if err != nil {
			labError(w, level12.ErrInvalid)
			return
		}
	}
	runs, err := service.List(r.Context(), id, limit)
	if err != nil {
		labError(w, err)
		return
	}
	writeJSON(w, 200, runs)
}
func (s *Server) labRun(w http.ResponseWriter, r *http.Request) {
	service, id, ok := s.labTask(w, r)
	if !ok {
		return
	}
	run, err := service.Get(r.Context(), id, r.PathValue("runID"))
	if err != nil {
		labError(w, err)
		return
	}
	writeJSON(w, 200, run)
}
func (s *Server) labPredictions(w http.ResponseWriter, r *http.Request) {
	service, id, ok := s.labTask(w, r)
	if !ok {
		return
	}
	predictions, err := service.Predictions(r.Context(), id)
	if err != nil {
		labError(w, err)
		return
	}
	writeJSON(w, 200, predictions)
}
func decodeLabRequest(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err == nil {
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			err = level12.ErrInvalid
		}
	}
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeErr(w, 413, "Solicitud demasiado grande")
		} else {
			writeErr(w, 400, "Solicitud inválida")
		}
		return false
	}
	return true
}
func (s *Server) labAnalyze(w http.ResponseWriter, r *http.Request) {
	service, id, ok := s.labTask(w, r)
	if !ok {
		return
	}
	var request level12.Request
	if !decodeLabRequest(w, r, &request) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	run, err := service.Analyze(ctx, id, request)
	if err != nil {
		labError(w, err)
		return
	}
	writeJSON(w, 201, run)
}
func (s *Server) labFeedback(w http.ResponseWriter, r *http.Request) {
	service, id, ok := s.labTask(w, r)
	if !ok {
		return
	}
	var request struct {
		Class      string `json:"class"`
		EvidenceID string `json:"evidence_id"`
		Confirmed  *bool  `json:"confirmed"`
	}
	if !decodeLabRequest(w, r, &request) {
		return
	}
	if request.Confirmed == nil {
		labError(w, level12.ErrInvalid)
		return
	}
	err := service.Feedback(r.Context(), id, r.PathValue("runID"), level12.Feedback{Class: request.Class, EvidenceID: request.EvidenceID, Confirmed: *request.Confirmed})
	if err != nil {
		labError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"recorded": true})
}
func labError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, level12.ErrUnavailable):
		writeErr(w, 503, "Faltan los controles de autorización del laboratorio")
	case errors.Is(err, level12.ErrDenied):
		writeErr(w, 403, "Operación fuera del ámbito autorizado del laboratorio")
	case errors.Is(err, level12.ErrInvalid):
		writeErr(w, 400, "Solicitud o evidencia inválida")
	case errors.Is(err, level12.ErrNotFound):
		writeErr(w, 404, "Análisis no encontrado")
	case errors.Is(err, level12.ErrConflict):
		writeErr(w, 409, "La evidencia o la evaluación ya está registrada")
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		writeErr(w, 408, "Análisis interrumpido")
	default:
		log.Printf("[laboratory] operation failed: %v", err)
		writeErr(w, 500, "No se pudo completar la operación del laboratorio")
	}
}

type recordedLabSource struct{ m *Manager }

func (s *recordedLabSource) Load(ctx context.Context, taskID int64, target string, ids []string) ([]blue.Observation, error) {
	// Only literal addresses already attached to this task are eligible. No DNS or requests.
	hosts, err := s.m.assets.HostsByTask(taskID)
	if err != nil {
		return nil, err
	}
	address, err := netip.ParseAddr(target)
	if err != nil {
		return nil, level12.ErrInvalid
	}
	allowed := false
	for _, host := range hosts {
		a, e := netip.ParseAddr(host)
		if e == nil && a == address {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, level12.ErrDenied
	}
	if s.m.traffic == nil {
		return nil, level12.ErrUnavailable
	}
	observations := make([]blue.Observation, 0, len(ids))
	for _, id := range ids {
		var rawURL, head string
		var ts int64
		var status int
		err = s.m.traffic.DB().QueryRowContext(ctx, `SELECT e.url,e.ts,e.status,b.resp_head FROM exchanges e JOIN exchange_bodies b ON b.id=e.id WHERE e.id=?`, id).Scan(&rawURL, &ts, &status, &head)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, level12.ErrInvalid
		}
		if err != nil {
			return nil, err
		}
		u, e := url.Parse(rawURL)
		if e != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, level12.ErrInvalid
		}
		captured, e := netip.ParseAddr(u.Hostname())
		if e != nil || captured != address || len(head) > 64<<10 || ts <= 0 {
			return nil, level12.ErrInvalid
		}
		headers, e := labResponseHeaders(head)
		if e != nil {
			return nil, level12.ErrInvalid
		}
		observations = append(observations, blue.Observation{ID: id, Target: target, Scheme: u.Scheme, Status: status, Headers: headers, ObservedAt: time.Unix(ts, 0).UTC()})
	}
	return observations, nil
}
func labResponseHeaders(raw string) (map[string][]string, error) {
	reader := bufio.NewReader(strings.NewReader(raw))
	// Captured headers include the response status line. Missing headers are not
	// evidence of absent protections: reject incomplete captures instead.
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(line)
	if len(parts) < 2 || (parts[0] != "HTTP" && parts[0] != "HTTP/1.0" && parts[0] != "HTTP/1.1" && parts[0] != "HTTP/2.0") {
		return nil, level12.ErrInvalid
	}
	status, err := strconv.Atoi(parts[1])
	if err != nil || status < 100 || status > 599 {
		return nil, level12.ErrInvalid
	}
	// The recorder stores "HTTP <status>\n" followed by header lines without
	// a terminating blank line; imports may instead contain an HTTP status line.
	rest, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	normalized := strings.TrimRight(string(rest), "\r\n") + "\r\n\r\n"
	headers, err := textproto.NewReader(bufio.NewReader(strings.NewReader(normalized))).ReadMIMEHeader()
	return map[string][]string(headers), err
}

// ConfigureLevel12Security binds the host's existing security services to the
// passive runtime. It never creates substitute OPSEC or ethical evaluators.
func (s *Server) ConfigureLevel12Security(scope network.Scope, cfg security.Config, agentIDs map[swarm.Role]string) error {
	adapter, err := security.New(cfg)
	if err != nil {
		return err
	}
	gate, err := adapter.Gate(scope)
	if err != nil {
		return err
	}
	return s.ConfigureLevel12(level12.Config{Gate: gate, AgentIDs: agentIDs})
}
