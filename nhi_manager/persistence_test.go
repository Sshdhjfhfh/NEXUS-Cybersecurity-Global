package nhi_manager

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestManagerRestartPreservesTrustAuditAndRevocation(t *testing.T) {
	dsn := os.Getenv("NEXUS_PG_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL is required")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	cfg := DefaultConfig()
	cfg.KeyDir = filepath.Join(t.TempDir(), "secrets")
	first, err := NewManager(t.Context(), pool, cfg)
	if err != nil {
		t.Fatal(err)
	}
	id := "restart-" + uuid.NewString()
	identity, err := first.Register(t.Context(), RegisterRequest{AgentID: id, AgentType: AgentTypeBlue, AllowedActions: []ActionType{"blue.analyze"}, AllowedAssets: []string{"10.20.0.3"}})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(t.Context(), `DELETE FROM nhi_identities WHERE agent_id=$1`, id)
	defer pool.Exec(t.Context(), `DELETE FROM nhi_audit WHERE actor_id=$1`, id)
	signed, err := first.Audit(t.Context(), identity, "blue.analyze", "10.20.0.3", []byte("evidence digest"), "ok", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewManager(t.Context(), pool, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.CACertPEM(), second.CACertPEM()) || first.SystemKeyID() != second.SystemKeyID() {
		t.Fatal("trust changed at restart")
	}
	if _, err := second.VerifyJWT(t.Context(), identity.JWTToken); err != nil {
		t.Fatal("existing token rejected after restart", err)
	}
	restored, err := second.RefreshIdentity(t.Context(), id)
	if err != nil || restored.JWTToken == "" {
		t.Fatal("credential refresh failed", err)
	}
	entries, err := second.Store().ListAudit(t.Context(), id, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.ID == signed.ID {
			found = true
			public, err := second.Store().GetSigningKey(t.Context(), entry.SigningKeyID)
			if err != nil {
				t.Fatal(err)
			}
			key, err := DilithiumPublicKeyFromBytes(public)
			if err != nil {
				t.Fatal(err)
			}
			valid, err := entry.Verify(&DilithiumKeyPair{PublicKey: key})
			if err != nil || !valid {
				t.Fatal("stored audit signature failed after restart", err)
			}
		}
	}
	if !found {
		t.Fatal("signed decision was not retained")
	}
	other, err := GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Store().SaveSigningKey(t.Context(), second.SystemKeyID(), other.PublicKeyBytes()); err == nil {
		t.Fatal("historical signing key overwritten")
	}
	if err := second.Revoke(t.Context(), id, RevocationReason("test"), "test revocation", "test"); err != nil {
		t.Fatal(err)
	}
	third, err := NewManager(t.Context(), pool, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := third.VerifyJWT(t.Context(), identity.JWTToken); err == nil {
		t.Fatal("revoked token valid after restart")
	}
}
