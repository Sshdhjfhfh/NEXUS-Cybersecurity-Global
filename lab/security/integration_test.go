package security

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexus-ia/nexus/abm"
	"github.com/nexus-ia/nexus/ans"
	"github.com/nexus-ia/nexus/lab"
	nhi "github.com/nexus-ia/nexus/nhi_manager"
	"os"
	"testing"
)

func TestRealServicesRejectRevocationPersistedOutsideCache(t *testing.T) {
	dsn := os.Getenv("NEXUS_PG_DSN")
	if dsn == "" {
		t.Skip("NEXUS_PG_DSN is required for isolated PostgreSQL integration")
	}
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
	id := "lab-test-" + uuid.NewString()
	identity, err := identities.Register(t.Context(), nhi.RegisterRequest{AgentID: id, AgentType: nhi.AgentTypeBlue, AllowedActions: []nhi.ActionType{"blue.analyze"}, AllowedAssets: []string{"10.20.0.0/24"}})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(t.Context(), `DELETE FROM nhi_identities WHERE agent_id=$1`, id)
	entry, err := directory.Register(t.Context(), ans.RegisterRequest{Name: id, AgentID: id, AgentType: string(identity.AgentType), DID: identity.DID.String(), CertX509PEM: identity.CertX509PEM, AllowedActions: []string{"blue.analyze"}, AllowedAssets: []string{"10.20.0.3"}})
	if err != nil {
		t.Fatal(err)
	}
	defer store.DeleteEntry(t.Context(), entry.Name)
	monitor, err := abm.NewMonitor(abm.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	authoritative, err := NewPersistentDirectory(store)
	if err != nil {
		t.Fatal(err)
	}
	persistentIdentities, err := NewPersistentIdentities(identities)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := New(Config{Directory: authoritative, Identities: persistentIdentities, Monitor: monitor, CACertPEM: identities.CACertPEM()})
	if err != nil {
		t.Fatal(err)
	}
	action := lab.Action{AgentID: id, TaskID: "7", Name: "blue.analyze", Target: "10.20.0.3"}
	if err := adapter.CheckIdentity(t.Context(), action); err != nil {
		t.Fatal(err)
	}
	// Current database grants narrow a still-valid cached token immediately.
	narrowed := *identity
	narrowed.AllowedActions = nil
	if err := identities.Store().SaveIdentity(t.Context(), &narrowed); err != nil {
		t.Fatal(err)
	}
	if err := adapter.CheckIdentity(t.Context(), action); err == nil {
		t.Fatal("persisted permission withdrawal was ignored")
	}
	if err := identities.Store().SaveIdentity(t.Context(), identity); err != nil {
		t.Fatal(err)
	}
	// Directory changes made outside its in-memory registry are authoritative.
	changed := *entry
	changed.Status = ans.StatusQuarantined
	if err := store.SaveEntry(t.Context(), &changed); err != nil {
		t.Fatal(err)
	}
	if err := adapter.CheckIdentity(t.Context(), action); err == nil {
		t.Fatal("persisted ANS quarantine was ignored")
	}
	if err := store.SaveEntry(t.Context(), entry); err != nil {
		t.Fatal(err)
	}
	if err := adapter.CheckIdentity(t.Context(), action); err != nil {
		t.Fatal(err)
	}
	// Persist revocation without touching the manager cache. JWT verification must
	// still consult the database before the next action is authorized.
	revoked := *identity
	revoked.Revoke("test revocation from another process")
	if err := identities.Store().SaveIdentity(t.Context(), &revoked); err != nil {
		t.Fatal(err)
	}
	if err := adapter.CheckIdentity(t.Context(), action); err == nil {
		t.Fatal("database revocation was ignored")
	}
	// A stale writer must never undo an already persisted revocation.
	if err := identities.Store().SaveIdentity(t.Context(), identity); err != nil {
		t.Fatal(err)
	}
	persisted, err := identities.Store().GetIdentity(t.Context(), id)
	if err != nil || persisted == nil || !persisted.IsRevoked() || persisted.RevokeReason != revoked.RevokeReason {
		t.Fatal("stale save cleared revocation", err)
	}
	if _, err := identities.VerifyJWT(t.Context(), identity.JWTToken); err == nil {
		t.Fatal("persisted revocation did not invalidate token")
	}

}
