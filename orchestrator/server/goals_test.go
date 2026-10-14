// Nexus IA v6.0 — Absorbido de ARTEX/server
// Upstream: github.com/mhtsec/ARTEX/server
// Licencia: AGPL-3.0
// ⚠️ WIP: depende de agents/red/ (reescritura pendiente)

package server

import (
	"context"
	"testing"
)

func TestCreateGoalsNilTask(t *testing.T) {
	if got := (&Server{}).createGoals(context.Background(), nil, nil); len(got) != 0 {
		t.Fatalf("nil task produced goals: %+v", got)
	}
}
