// Package prediction ranks laboratory hypotheses from recorded outcomes.
// It performs no target interaction and never treats a prediction as evidence.
package prediction

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Outcome struct {
	SessionID string `json:"session_id"`
	Class     string `json:"class"`
	Success   bool   `json:"success"`
}
type Counts struct {
	Success uint64 `json:"success"`
	Failure uint64 `json:"failure"`
}
type Prediction struct {
	Class       string  `json:"class"`
	Probability float64 `json:"probability"`
	Samples     uint64  `json:"samples"`
	Lower       float64 `json:"lower"`
	Upper       float64 `json:"upper"`
	Ready       bool    `json:"ready"`
}
type Engine struct {
	mu       sync.RWMutex
	counts   map[string]Counts
	sessions map[string]map[string]bool
}
type snapshot struct {
	Version  int                        `json:"version"`
	Counts   map[string]Counts          `json:"counts"`
	Sessions map[string]map[string]bool `json:"sessions"`
}

const maxSessions = 100000

func New(classes ...string) (*Engine, error) {
	if len(classes) == 0 || len(classes) > 128 {
		return nil, fmt.Errorf("one to 128 prediction classes are required")
	}
	e := &Engine{counts: map[string]Counts{}, sessions: map[string]map[string]bool{}}
	for _, class := range classes {
		if class == "" || len(class) > 128 || strings.TrimSpace(class) != class {
			return nil, fmt.Errorf("invalid prediction class")
		}
		if _, exists := e.counts[class]; exists {
			return nil, fmt.Errorf("duplicate prediction class %q", class)
		}
		e.counts[class] = Counts{}
	}
	return e, nil
}

// Record validates an entire batch before updating it. One outcome per class
// per session prevents repeated feedback from inflating the apparent evidence.
func (e *Engine) Record(outcomes ...Outcome) error {
	if e == nil {
		return fmt.Errorf("prediction engine is not configured")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	pending := map[string]map[string]bool{}
	for _, o := range outcomes {
		if o.SessionID == "" || len(o.SessionID) > 256 {
			return fmt.Errorf("invalid session identity")
		}
		if _, ok := e.counts[o.Class]; !ok {
			return fmt.Errorf("unknown prediction class %q", o.Class)
		}
		if _, exists := e.sessions[o.SessionID][o.Class]; exists {
			return fmt.Errorf("duplicate session outcome")
		}
		if pending[o.SessionID] == nil {
			pending[o.SessionID] = map[string]bool{}
		}
		if _, exists := pending[o.SessionID][o.Class]; exists {
			return fmt.Errorf("duplicate outcome in batch")
		}
		pending[o.SessionID][o.Class] = o.Success
	}
	newSessions := 0
	for id := range pending {
		if e.sessions[id] == nil {
			newSessions++
		}
	}
	if len(e.sessions)+newSessions > maxSessions {
		return fmt.Errorf("session retention limit reached")
	}
	for id, classes := range pending {
		if e.sessions[id] == nil {
			e.sessions[id] = map[string]bool{}
		}
		for class, success := range classes {
			e.sessions[id][class] = success
			counts := e.counts[class]
			if success {
				counts.Success++
			} else {
				counts.Failure++
			}
			e.counts[class] = counts
		}
	}
	return nil
}

// Rank returns Beta(1,1) posterior means and Wilson 95% evidence intervals.
// Ready requires outcomes from at least 50 distinct sessions for that class.
func (e *Engine) Rank() []Prediction {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]Prediction, 0, len(e.counts))
	for class, c := range e.counts {
		samples := c.Success + c.Failure
		prediction := Prediction{Class: class, Probability: (float64(c.Success) + 1) / (float64(samples) + 2), Samples: samples, Lower: 0, Upper: 1, Ready: samples >= 50}
		if samples > 0 {
			n := float64(samples)
			p := float64(c.Success) / n
			z := 1.96
			denom := 1 + z*z/n
			center := (p + z*z/(2*n)) / denom
			spread := z * math.Sqrt(p*(1-p)/n+z*z/(4*n*n)) / denom
			prediction.Lower = math.Max(0, center-spread)
			prediction.Upper = math.Min(1, center+spread)
		}
		result = append(result, prediction)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Probability != result[j].Probability {
			return result[i].Probability > result[j].Probability
		}
		return result[i].Class < result[j].Class
	})
	return result
}

func (e *Engine) Save(path string) error {
	e.mu.RLock()
	data, err := json.Marshal(snapshot{1, e.counts, e.sessions})
	e.mu.RUnlock()
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".prediction-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Load reconstructs counts from sessions and rejects corrupt/inconsistent files.
func Load(path string) (*Engine, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var state snapshot
	decoder := json.NewDecoder(io.LimitReader(file, 64<<20))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&state); err != nil {
		return nil, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after prediction snapshot")
	}
	if state.Version != 1 || len(state.Sessions) > maxSessions {
		return nil, fmt.Errorf("unsupported prediction snapshot")
	}
	classes := make([]string, 0, len(state.Counts))
	for class := range state.Counts {
		classes = append(classes, class)
	}
	engine, err := New(classes...)
	if err != nil {
		return nil, err
	}
	outcomes := []Outcome{}
	for session, values := range state.Sessions {
		if len(values) == 0 {
			return nil, fmt.Errorf("empty session in prediction snapshot")
		}
		for class, success := range values {
			outcomes = append(outcomes, Outcome{session, class, success})
		}
	}
	if err = engine.Record(outcomes...); err != nil {
		return nil, err
	}
	for class, counts := range engine.counts {
		if counts != state.Counts[class] {
			return nil, fmt.Errorf("prediction snapshot counts do not match evidence")
		}
	}
	return engine, nil
}
