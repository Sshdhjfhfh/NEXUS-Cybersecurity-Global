package security

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-ia/nexus/lab"
	nhi "github.com/nexus-ia/nexus/nhi_manager"
)

// NewAuditRecorder signs decisions with the manager's system key. Unknown actors
// are explicitly recorded without a verified DID; they are never registered.
func NewAuditRecorder(manager *nhi.Manager) (lab.Audit, error) {
	if manager == nil {
		return nil, fmt.Errorf("NHI audit manager is required")
	}
	return func(ctx context.Context, action lab.Action, decision error) error {
		if len(action.AgentID) > 128 || len(action.Name) > 128 || len(action.Target) > 128 || len(action.TaskID) > 32 {
			return fmt.Errorf("audit action limit exceeded")
		}
		actor, err := manager.Store().GetIdentity(ctx, action.AgentID)
		if err != nil {
			return fmt.Errorf("cannot resolve audit actor")
		}
		entry := &nhi.AuditEntry{ID: uuid.NewString(), ActorID: action.AgentID, AgentType: "UNKNOWN", Action: action.Name, Target: action.Target, Timestamp: time.Now().UTC().Truncate(time.Microsecond), Metadata: map[string]string{"task_id": action.TaskID, "phase": "authorization"}, Result: "allowed", Reason: "authorization checks passed"}
		if actor != nil {
			entry.ActorDID = actor.DID.String()
			entry.AgentType = actor.AgentType
		}
		if decision != nil {
			entry.Result = "denied"
			entry.Reason = "authorization checks denied"
		}
		payload, err := json.Marshal(action)
		if err != nil {
			return err
		}
		entry.PayloadHash = nhi.HashSHA256(payload)
		return manager.RecordAudit(ctx, entry)
	}, nil
}
