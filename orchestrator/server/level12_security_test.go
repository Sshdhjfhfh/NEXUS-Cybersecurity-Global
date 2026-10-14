package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexus-ia/nexus/abm"
	"github.com/nexus-ia/nexus/ans"
	"github.com/nexus-ia/nexus/db"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/lab/network"
	"github.com/nexus-ia/nexus/lab/security"
	nhi "github.com/nexus-ia/nexus/nhi_manager"
	"github.com/nexus-ia/nexus/orchestrator/level12"
	"github.com/nexus-ia/nexus/swarm"
)

func TestLabSecureCompositionWithRealIdentities(t *testing.T) {
	dsn := os.Getenv("NEXUS_PG_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL is required")
	}
	s, f, request := trafficEvidenceServer(t)
	finding, err := s.m.pg.GetFinding(f.FindingID)
	if err != nil || finding.TaskID == nil {
		t.Fatal("missing task", err)
	}
	taskID := *finding.TaskID
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	identities, err := nhi.NewManager(t.Context(), pool, nhi.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	store := ans.NewStore(pool)
	if err := store.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	directory := ans.NewService(ans.DefaultConfig(), nil, store)
	authoritative, err := security.NewPersistentDirectory(store)
	if err != nil {
		t.Fatal(err)
	}
	currentIdentities, err := security.NewPersistentIdentities(identities)
	if err != nil {
		t.Fatal(err)
	}
	monitor, err := abm.NewMonitor(abm.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	audit, err := security.NewAuditRecorder(identities)
	if err != nil {
		t.Fatal(err)
	}
	cfg := security.Config{Audit: audit, Directory: authoritative, Identities: currentIdentities, Monitor: monitor, CACertPEM: identities.CACertPEM()}
	scope, err := network.NewScope("10.20.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigureLevel12Security(scope, cfg, nil); err == nil || s.laboratory().Status().Ready {
		t.Fatal("missing OPSEC and ethics accepted")
	}
	ids := map[swarm.Role]string{}
	for _, role := range swarm.Roles() {
		id := "lab-secure-" + uuid.NewString()
		ids[role] = id
		actions := []nhi.ActionType{nhi.ActionType("swarm." + string(role))}
		agentType := nhi.AgentTypeScout
		switch role {
		case swarm.Analyzer:
			actions = append(actions, "blue.analyze")
			agentType = nhi.AgentTypeBlue
		case swarm.Predictor:
			actions = append(actions, "prediction.feedback")
			agentType = nhi.AgentTypePrediction
		case swarm.Coordinator:
			actions = append(actions, "level12.persist")
			agentType = nhi.AgentTypeSwarmCoordinator
		}
		identity, err := identities.Register(t.Context(), nhi.RegisterRequest{AgentID: id, AgentType: agentType, AllowedActions: actions, AllowedAssets: []string{"10.20.0.3"}})
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(context.Background(), `DELETE FROM nhi_identities WHERE agent_id=$1`, id)
		defer pool.Exec(context.Background(), `DELETE FROM nhi_audit WHERE actor_id=$1`, id)
		grants := []string{}
		for _, action := range actions {
			grants = append(grants, string(action))
		}
		entry, err := directory.Register(t.Context(), ans.RegisterRequest{Name: id, AgentID: id, AgentType: string(agentType), DID: identity.DID.String(), CertX509PEM: identity.CertX509PEM, AllowedActions: grants, AllowedAssets: []string{"10.20.0.3"}})
		if err != nil {
			t.Fatal(err)
		}
		defer store.DeleteEntry(context.Background(), entry.Name)
	}
	// These two callbacks belong only to this isolated test fixture. Production
	// installs neither and must remain blocked until real evaluators are supplied.
	cfg.OPSEC = func(context.Context, lab.Action) error { return nil }
	cfg.Ethics = func(context.Context, lab.Action) error { return nil }
	if err := s.ConfigureLevel12Security(scope, cfg, ids); err != nil {
		t.Fatal(err)
	}
	_, err = s.m.pg.Assets().UpsertIP(db.UpsertIPReq{IP: "10.20.0.3", TaskID: taskID})
	if err != nil {
		t.Fatal(err)
	}
	seedServerEvidenceFlow(t, s, "secure-lab", nil)
	_, err = s.m.traffic.DB().Exec(`UPDATE exchanges SET url='http://10.20.0.3/',host='10.20.0.3' WHERE id='secure-lab'`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.m.traffic.DB().Exec(`UPDATE exchange_bodies SET resp_head=? WHERE id='secure-lab'`, "HTTP 200\nContent-Type: text/html\n")
	if err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/lab/tasks/%d/runs", taskID)
	body := `{"target":"10.20.0.3","evidence_ids":["secure-lab"]}`
	response := request("POST", base, body)
	if response.Code != 201 {
		adapter, _ := security.New(cfg)
		gate, _ := adapter.Gate(scope)
		detail := gate.Authorize(t.Context(), lab.Action{AgentID: ids[swarm.Recon], TaskID: fmt.Sprint(taskID), Name: "swarm.recon", Target: "10.20.0.3"})
		t.Fatalf("secure pipeline: %d %s detail=%v", response.Code, response.Body.String(), detail)
	}
	var run level12.Run
	if err := json.Unmarshal(response.Body.Bytes(), &run); err != nil || len(run.Workers) != 8 {
		t.Fatal("workers did not complete", err)
	}
	identity, err := identities.GetIdentity(t.Context(), ids[swarm.Recon])
	if err != nil {
		t.Fatal(err)
	}
	revoked := *identity
	revoked.Revoke("test persisted revocation")
	if err := identities.Store().SaveIdentity(t.Context(), &revoked); err != nil {
		t.Fatal(err)
	}
	if response := request("POST", base, body); response.Code != 403 {
		t.Fatalf("revoked worker: %d %s", response.Code, response.Body.String())
	}
	var approvals, denials int
	actorIDs := []string{}
	for _, id := range ids {
		actorIDs = append(actorIDs, id)
	}
	if err := pool.QueryRow(t.Context(), `SELECT COUNT(*) FILTER(WHERE result='allowed'),COUNT(*) FILTER(WHERE result='denied') FROM nhi_audit WHERE actor_id=ANY($1) AND metadata->>'phase'='authorization'`, actorIDs).Scan(&approvals, &denials); err != nil {
		t.Fatal(err)
	}
	if approvals != 10 || denials != 1 {
		t.Fatalf("missing authorization audit: allowed=%d denied=%d", approvals, denials)
	}
	history, err := s.laboratory().List(t.Context(), taskID, 20)
	if err != nil || len(history) != 1 {
		t.Fatal("denied run was persisted", err)
	}
}
