// Nexus IA v6.0 — Absorbido de ARTEX/agent
// Upstream: github.com/mhtsec/ARTEX/agent
// Licencia: AGPL-3.0

package agent

import (
	"context"
	"strconv"
	"strings"

	"github.com/nexus-ia/nexus/orchestrator/sidequestion"
	"github.com/nexus-ia/nexus/core/agent"
)

func attachSideCapture(ctx context.Context, opts *agentcore.Options) context.Context {
	ri := RunInfoFrom(ctx)
	p := sidequestion.Parent{TaskID: ri.TaskID, ExplorationID: ri.ExplorationID, IntentID: ri.IntentID}
	if strings.HasPrefix(ri.SessionID, "conv-") {
		p.ConversationID, _ = strconv.ParseInt(strings.TrimPrefix(ri.SessionID, "conv-"), 10, 64)
	}
	if p.ConversationID == 0 && (p.TaskID == 0 || p.ExplorationID == 0) {
		return ctx
	}
	ctx, opts.Deps = sidequestion.Attach(ctx, p, opts.Deps, opts.Provider)
	return ctx
}
