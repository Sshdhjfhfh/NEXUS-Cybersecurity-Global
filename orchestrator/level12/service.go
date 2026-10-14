package level12

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-ia/nexus/agents/blue"
	"github.com/nexus-ia/nexus/agents/prediction"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/swarm"
)

type Service struct {
	cfg     Config
	runSlot chan struct{}
}

func New(cfg Config) (*Service, error) {
	if cfg.Repository == nil {
		return nil, fmt.Errorf("laboratory repository is required")
	}
	ids := map[swarm.Role]string{}
	seen := map[string]bool{}
	if cfg.Gate != nil && cfg.Gate.Ready() {
		if len(cfg.AgentIDs) != 8 {
			return nil, fmt.Errorf("all eight laboratory identities are required")
		}
		for _, role := range swarm.Roles() {
			id := cfg.AgentIDs[role]
			if id == "" || seen[id] {
				return nil, fmt.Errorf("laboratory identities must be distinct")
			}
			ids[role] = id
			seen[id] = true
		}
	}
	cfg.AgentIDs = ids
	return &Service{cfg: cfg, runSlot: make(chan struct{}, 1)}, nil
}
func (s *Service) Status() Status {
	blockers := []string{}
	if s.cfg.Gate == nil || !s.cfg.Gate.Ready() {
		blockers = append(blockers, "authorization")
	}
	if s.cfg.Source == nil {
		blockers = append(blockers, "evidence_source")
	}
	return Status{Ready: len(blockers) == 0, Blockers: blockers, Roles: swarm.Roles(), Classes: Classes(), PassiveOnly: true}
}
func (s *Service) List(ctx context.Context, taskID int64, limit int) ([]Run, error) {
	if taskID <= 0 || limit < 1 || limit > 100 {
		return nil, ErrInvalid
	}
	return s.cfg.Repository.ListRuns(ctx, taskID, limit)
}
func (s *Service) Get(ctx context.Context, taskID int64, id string) (*Run, error) {
	if taskID <= 0 {
		return nil, ErrInvalid
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalid
	}
	return s.cfg.Repository.GetRun(ctx, taskID, id)
}
func (s *Service) Predictions(ctx context.Context, taskID int64) ([]prediction.Prediction, error) {
	if taskID <= 0 {
		return nil, ErrInvalid
	}
	outcomes, err := s.cfg.Repository.Outcomes(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if len(outcomes) > 100000 {
		return nil, fmt.Errorf("prediction feedback limit exceeded")
	}
	model, _ := prediction.New(Classes()...)
	batch := make([]prediction.Outcome, 0, len(outcomes))
	for _, outcome := range outcomes {
		batch = append(batch, prediction.Outcome{SessionID: outcome.RunID, Class: outcome.Class, Success: outcome.Confirmed})
	}
	if err = model.Record(batch...); err != nil {
		return nil, fmt.Errorf("rebuild laboratory predictions: %w", err)
	}
	return model.Rank(), nil
}

var evidenceID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func validate(request Request) error {
	addr, err := netip.ParseAddr(request.Target)
	if err != nil || addr.Is4In6() || addr.Zone() != "" || !addr.IsPrivate() {
		return ErrInvalid
	}
	if len(request.EvidenceIDs) == 0 || len(request.EvidenceIDs) > MaxEvidence {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range request.EvidenceIDs {
		if !evidenceID.MatchString(id) || seen[id] {
			return ErrInvalid
		}
		seen[id] = true
	}
	return nil
}
func (s *Service) Analyze(ctx context.Context, taskID int64, request Request) (*Run, error) {
	if taskID <= 0 {
		return nil, ErrInvalid
	}
	if err := validate(request); err != nil {
		return nil, err
	}
	if !s.Status().Ready {
		return nil, ErrUnavailable
	}
	// Serialize runs as well as workers: identities cannot overlap across schedulers.
	select {
	case s.runSlot <- struct{}{}:
		defer func() { <-s.runSlot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	// Copy caller-owned request slices before scheduling concurrent workers.
	request.EvidenceIDs = append([]string(nil), request.EvidenceIDs...)
	state := &analysis{service: s, taskID: taskID, request: request}
	workers := map[swarm.Role]swarm.Worker{}
	for _, role := range swarm.Roles() {
		workers[role] = &worker{id: s.cfg.AgentIDs[role], role: role, state: state}
	}
	scheduler, err := swarm.New(denialGate{s.cfg.Gate}, workers, 4)
	if err != nil {
		return nil, err
	}
	jobs := []swarm.Job{}
	deps := map[swarm.Role][]string{
		swarm.Recon: nil, swarm.Mapper: {string(swarm.Recon)}, swarm.Analyzer: {string(swarm.Mapper)},
		swarm.Validator: {string(swarm.Analyzer)}, swarm.Defender: {string(swarm.Validator)},
		swarm.Predictor: {string(swarm.Validator)}, swarm.Reporter: {string(swarm.Defender), string(swarm.Predictor)},
		swarm.Coordinator: {string(swarm.Reporter)},
	}
	for _, role := range swarm.Roles() {
		jobs = append(jobs, swarm.Job{ID: string(role), TaskID: strconv.FormatInt(taskID, 10), Role: role, Target: request.Target, Dependencies: deps[role]})
	}
	state.run = Run{ID: uuid.NewString(), TaskID: taskID, Target: request.Target, EvidenceIDs: request.EvidenceIDs,
		StartedAt: time.Now().UTC(), Recommendations: []string{}, Findings: []blue.Finding{}, Workers: []WorkerResult{}}
	results, err := scheduler.Run(ctx, jobs)
	if err != nil {
		return nil, err
	}
	for _, result := range results {
		role := swarm.Role(result.JobID)
		state.run.Workers = append(state.run.Workers, WorkerResult{Role: role, AgentID: s.cfg.AgentIDs[role], State: "completed"})
	}
	state.run.FinishedAt = time.Now().UTC()
	raw, err := json.Marshal(state.run)
	if err != nil {
		return nil, err
	}
	if len(raw) > 4<<20 {
		return nil, fmt.Errorf("laboratory report limit exceeded")
	}
	if err = s.cfg.Gate.Authorize(ctx, labAction(s.cfg.AgentIDs[swarm.Coordinator], taskID, "level12.persist", request.Target)); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDenied, err)
	}
	if err = s.cfg.Repository.SaveRun(ctx, state.run); err != nil {
		return nil, err
	}
	return &state.run, nil
}
func (s *Service) Feedback(ctx context.Context, taskID int64, id string, feedback Feedback) error {
	if !s.Status().Ready {
		return ErrUnavailable
	}
	run, err := s.Get(ctx, taskID, id)
	if err != nil {
		return err
	}
	if run == nil {
		return ErrNotFound
	}
	valid := false
	for _, finding := range run.Findings {
		if finding.Rule == feedback.Class && finding.EvidenceID == feedback.EvidenceID {
			valid = true
			break
		}
	}
	if !valid {
		return ErrInvalid
	}
	if err = s.cfg.Gate.Authorize(ctx, labAction(s.cfg.AgentIDs[swarm.Predictor], taskID, "prediction.feedback", run.Target)); err != nil {
		return fmt.Errorf("%w: %v", ErrDenied, err)
	}
	return s.cfg.Repository.AddFeedback(ctx, taskID, id, feedback)
}

type analysis struct {
	service      *Service
	taskID       int64
	request      Request
	observations []blue.Observation
	run          Run
}
type worker struct {
	id    string
	role  swarm.Role
	state *analysis
}

func (w *worker) ID() string { return w.id }
func (w *worker) Execute(ctx context.Context, job swarm.Job) ([]byte, error) {
	a := w.state
	switch w.role {
	case swarm.Recon:
		observations, err := a.service.cfg.Source.Load(ctx, a.taskID, a.request.Target, a.request.EvidenceIDs)
		if err != nil {
			return nil, err
		}
		if len(observations) != len(a.request.EvidenceIDs) {
			return nil, fmt.Errorf("%w: incomplete evidence", ErrInvalid)
		}
		expected := map[string]bool{}
		for _, id := range a.request.EvidenceIDs {
			expected[id] = true
		}
		for _, o := range observations {
			if !expected[o.ID] || o.Target != a.request.Target {
				return nil, fmt.Errorf("%w: evidence does not match the requested target", ErrInvalid)
			}
			delete(expected, o.ID)
		}
		sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
		a.observations = observations
		raw, err := json.Marshal(observations)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(raw)
		a.run.EvidenceDigest = hex.EncodeToString(digest[:])
	case swarm.Mapper:
		for _, o := range a.observations {
			if o.Target != job.Target {
				return nil, ErrInvalid
			}
		}
	case swarm.Analyzer:
		sentinel, err := blue.New(w.id, denialGate{a.service.cfg.Gate})
		if err != nil {
			return nil, err
		}
		findings, err := sentinel.Analyze(ctx, job.TaskID, a.observations)
		if err != nil {
			return nil, err
		}
		if len(findings) > 4096 {
			return nil, fmt.Errorf("finding limit exceeded")
		}
		a.run.Findings = findings
	case swarm.Validator:
		for _, f := range a.run.Findings {
			valid := false
			for _, o := range a.observations {
				if f.EvidenceID == o.ID && f.Target == o.Target {
					valid = true
					break
				}
			}
			if !valid {
				return nil, fmt.Errorf("finding has no matching evidence")
			}
		}
	case swarm.Defender:
		seen := map[string]bool{}
		for _, f := range a.run.Findings {
			seen[f.Recommendation] = true
		}
		for recommendation := range seen {
			a.run.Recommendations = append(a.run.Recommendations, recommendation)
		}
		sort.Strings(a.run.Recommendations)
	case swarm.Predictor:
		predictions, err := a.service.Predictions(ctx, a.taskID)
		if err != nil {
			return nil, err
		}
		a.run.Predictions = predictions
	case swarm.Reporter:
		if a.run.EvidenceDigest == "" || a.run.Predictions == nil {
			return nil, fmt.Errorf("report is incomplete")
		}
	case swarm.Coordinator:
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported laboratory worker")
	}
	return []byte(`{"status":"completed"}`), nil
}

func labAction(agentID string, taskID int64, name, target string) lab.Action {
	return lab.Action{AgentID: agentID, TaskID: strconv.FormatInt(taskID, 10), Name: name, Target: target}
}

// Preserve denial classification through scheduler error aggregation.
type denialGate struct{ gate *lab.Gate }

func (g denialGate) Authorize(ctx context.Context, action lab.Action) error {
	if err := g.gate.Authorize(ctx, action); err != nil {
		return fmt.Errorf("%w: %w", ErrDenied, err)
	}
	return nil
}
