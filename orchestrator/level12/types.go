// Package level12 connects passive laboratory analysis, eight workers and feedback.
// It performs no network requests and provides no permissive authorization default.
package level12

import (
	"context"
	"errors"
	"time"

	"github.com/nexus-ia/nexus/agents/blue"
	"github.com/nexus-ia/nexus/agents/prediction"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/swarm"
)

var (
	ErrUnavailable = errors.New("laboratory runtime is unavailable")
	ErrDenied      = errors.New("laboratory authorization denied")
	ErrInvalid     = errors.New("invalid laboratory request")
	ErrConflict    = errors.New("feedback already recorded")
	ErrNotFound    = errors.New("laboratory run not found")
)

const MaxEvidence = 64

func Classes() []string {
	return []string{"cleartext", "missing-hsts", "missing-csp", "missing-nosniff", "cookie-flags", "login-failures"}
}

type Request struct {
	Target      string   `json:"target"`
	EvidenceIDs []string `json:"evidence_ids"`
}
type WorkerResult struct {
	Role    swarm.Role `json:"role"`
	AgentID string     `json:"agent_id"`
	State   string     `json:"state"`
}
type Run struct {
	ID              string                  `json:"id"`
	TaskID          int64                   `json:"task_id"`
	Target          string                  `json:"target"`
	EvidenceIDs     []string                `json:"evidence_ids"`
	EvidenceDigest  string                  `json:"evidence_digest"`
	Findings        []blue.Finding          `json:"findings"`
	Predictions     []prediction.Prediction `json:"predictions"`
	Recommendations []string                `json:"recommendations"`
	Workers         []WorkerResult          `json:"workers"`
	StartedAt       time.Time               `json:"started_at"`
	FinishedAt      time.Time               `json:"finished_at"`
}
type Feedback struct {
	Class      string `json:"class"`
	EvidenceID string `json:"evidence_id"`
	Confirmed  bool   `json:"confirmed"`
}
type Outcome struct {
	RunID     string
	Class     string
	Confirmed bool
}
type Repository interface {
	SaveRun(context.Context, Run) error
	GetRun(context.Context, int64, string) (*Run, error)
	ListRuns(context.Context, int64, int) ([]Run, error)
	AddFeedback(context.Context, int64, string, Feedback) error
	Outcomes(context.Context, int64) ([]Outcome, error)
}
type Source interface {
	Load(context.Context, int64, string, []string) ([]blue.Observation, error)
}
type Config struct {
	Repository Repository
	Source     Source
	Gate       *lab.Gate
	AgentIDs   map[swarm.Role]string
}
type Status struct {
	Ready       bool         `json:"ready"`
	Blockers    []string     `json:"blockers"`
	Roles       []swarm.Role `json:"roles"`
	Classes     []string     `json:"classes"`
	PassiveOnly bool         `json:"passive_only"`
}
