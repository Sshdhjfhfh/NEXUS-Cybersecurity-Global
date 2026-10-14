// Package swarm schedules eight specialized laboratory workers in a bounded DAG.
// Workers receive a job only after the shared laboratory gate authorizes it.
package swarm

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/nexus-ia/nexus/lab"
)

type Role string

const (
	Recon       Role = "recon"
	Mapper      Role = "mapper"
	Analyzer    Role = "analyzer"
	Validator   Role = "validator"
	Defender    Role = "defender"
	Predictor   Role = "predictor"
	Reporter    Role = "reporter"
	Coordinator Role = "coordinator"
)

func Roles() []Role {
	return []Role{Recon, Mapper, Analyzer, Validator, Defender, Predictor, Reporter, Coordinator}
}

type Job struct {
	ID           string   `json:"id"`
	TaskID       string   `json:"task_id"`
	Role         Role     `json:"role"`
	Target       string   `json:"target"`
	Dependencies []string `json:"dependencies"`
	Payload      []byte   `json:"payload"`
}
type Worker interface {
	ID() string
	Execute(context.Context, Job) ([]byte, error)
}
type Result struct {
	JobID   string
	Output  []byte
	Err     error
	Skipped bool
}
type Swarm struct {
	gate     lab.Authorizer
	workers  map[Role]Worker
	parallel int
	slots    map[Role]chan struct{}
}

func New(gate lab.Authorizer, workers map[Role]Worker, parallel int) (*Swarm, error) {
	if gate == nil || parallel < 1 || parallel > 8 {
		return nil, fmt.Errorf("authorization and concurrency between one and eight are required")
	}
	registered := map[Role]Worker{}
	ids := map[string]bool{}
	if len(workers) != 8 {
		return nil, fmt.Errorf("all eight specialized workers are required")
	}
	for _, role := range Roles() {
		worker := workers[role]
		if worker == nil || worker.ID() == "" || ids[worker.ID()] {
			return nil, fmt.Errorf("role %s requires a distinct authenticated worker", role)
		}
		ids[worker.ID()] = true
		registered[role] = worker
	}
	slots := map[Role]chan struct{}{}
	for _, role := range Roles() {
		slots[role] = make(chan struct{}, 1)
	}
	return &Swarm{gate: gate, workers: registered, parallel: parallel, slots: slots}, nil
}

func validate(jobs []Job) ([][]int, error) {
	if len(jobs) > 1024 {
		return nil, fmt.Errorf("job limit exceeded")
	}
	ids := map[string]int{}
	for i, job := range jobs {
		if job.ID == "" || job.TaskID == "" || job.Target == "" {
			return nil, fmt.Errorf("job, task and target are required")
		}
		if _, exists := ids[job.ID]; exists {
			return nil, fmt.Errorf("duplicate job %s", job.ID)
		}
		ids[job.ID] = i
	}
	remaining := make([]int, len(jobs))
	dependents := make([][]int, len(jobs))
	seenEdges := map[string]bool{}
	for i, job := range jobs {
		for _, dep := range job.Dependencies {
			parent, exists := ids[dep]
			if !exists || parent == i || jobs[parent].TaskID != job.TaskID {
				return nil, fmt.Errorf("invalid dependency %s", dep)
			}
			key := job.ID + "\x00" + dep
			if seenEdges[key] {
				return nil, fmt.Errorf("duplicate dependency")
			}
			seenEdges[key] = true
			remaining[i]++
			dependents[parent] = append(dependents[parent], i)
		}
	}
	layers := [][]int{}
	processed := 0
	for {
		layer := []int{}
		for i, count := range remaining {
			if count == 0 {
				layer = append(layer, i)
			}
		}
		if len(layer) == 0 {
			break
		}
		for _, i := range layer {
			remaining[i] = -1
			processed++
			for _, child := range dependents[i] {
				remaining[child]--
			}
		}
		layers = append(layers, layer)
	}
	if processed != len(jobs) {
		return nil, fmt.Errorf("job dependency cycle detected")
	}
	return layers, nil
}

// Run preserves input ordering, executes independent jobs concurrently, and
// skips descendants of failed jobs. Cancellation never schedules a new worker.
// Workers must honor their context; Go cannot safely terminate arbitrary code.
func (s *Swarm) Run(ctx context.Context, jobs []Job) ([]Result, error) {
	if s == nil || s.gate == nil {
		return nil, fmt.Errorf("swarm is not configured")
	}
	// Snapshot the input so workers cannot alter the coordinator's dependency graph.
	copied := make([]Job, len(jobs))
	for i, j := range jobs {
		copied[i] = j
		copied[i].Dependencies = append([]string(nil), j.Dependencies...)
		copied[i].Payload = append([]byte(nil), j.Payload...)
	}
	jobs = copied
	layers, err := validate(jobs)
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if s.workers[job.Role] == nil {
			return nil, fmt.Errorf("unregistered worker role %s", job.Role)
		}
	}
	results := make([]Result, len(jobs))
	indices := map[string]int{}
	for i, job := range jobs {
		results[i].JobID = job.ID
		indices[job.ID] = i
	}
	for _, layer := range layers {
		var wg sync.WaitGroup
		sem := make(chan struct{}, s.parallel)
		for _, i := range layer {
			job := jobs[i]
			blocked := false
			for _, dep := range job.Dependencies {
				if results[indices[dep]].Err != nil {
					blocked = true
					break
				}
			}
			if blocked {
				results[i].Err = fmt.Errorf("dependency failed")
				results[i].Skipped = true
				continue
			}
			if err := ctx.Err(); err != nil {
				results[i].Err = err
				results[i].Skipped = true
				continue
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[i].Err = ctx.Err()
				results[i].Skipped = true
				continue
			}
			wg.Add(1)
			go func(i int, job Job) {
				defer wg.Done()
				defer func() { <-sem }()
				defer func() {
					if value := recover(); value != nil {
						results[i].Err = fmt.Errorf("worker panic: %v", value)
					}
				}()
				worker := s.workers[job.Role]
				select {
				case s.slots[job.Role] <- struct{}{}:
				case <-ctx.Done():
					results[i].Err = ctx.Err()
					return
				}
				defer func() { <-s.slots[job.Role] }()
				if err := s.gate.Authorize(ctx, lab.Action{AgentID: worker.ID(), TaskID: job.TaskID, Name: "swarm." + string(job.Role), Target: job.Target}); err != nil {
					results[i].Err = err
					return
				}
				if err := ctx.Err(); err != nil {
					results[i].Err = err
					return
				}
				output, err := worker.Execute(ctx, job)
				results[i].Output = append([]byte(nil), output...)
				results[i].Err = err
			}(i, job)
		}
		wg.Wait()
	}
	failed := []error{}
	for _, result := range results {
		if result.Err != nil {
			failed = append(failed, fmt.Errorf("job %s: %w", result.JobID, result.Err))
		}
	}
	return results, errors.Join(failed...)
}

// Summaries returns completed job identifiers deterministically for reporting.
func Summaries(results []Result) []string {
	ids := []string{}
	for _, r := range results {
		if r.Err == nil {
			ids = append(ids, r.JobID)
		}
	}
	sort.Strings(ids)
	return ids
}
