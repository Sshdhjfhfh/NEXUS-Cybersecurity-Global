package security

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/nexus-ia/nexus/abm"
	"github.com/nexus-ia/nexus/ans"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/lab/network"
	nhi "github.com/nexus-ia/nexus/nhi_manager"
)

type testIdentities struct {
	identity *nhi.NHI
	key      nhi.JWTSigningKey
	failure  error
}

func (s *testIdentities) GetIdentity(context.Context, string) (*nhi.NHI, error) {
	return s.identity, s.failure
}
func (s *testIdentities) VerifyJWT(_ context.Context, token string) (*nhi.ClaimsNexus, error) {
	return nhi.VerifyJWT(s.key, token)
}

type fixture struct {
	registry   *ans.Registry
	adapter    *Adapter
	directory  *ans.Service
	identities *testIdentities
	entry      *ans.Entry
	monitor    *abm.DefaultMonitor
	action     lab.Action
}

func setup(t *testing.T) *fixture {
	t.Helper()
	now := time.Now()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test lab CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, root, root, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	rootPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	identity := &nhi.NHI{AgentID: "test-blue", AgentType: nhi.AgentTypeBlue, DID: nhi.NewDID(), IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour), AllowedActions: []nhi.ActionType{"blue.analyze"}, AllowedAssets: []string{"10.20.0.0/24"}}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: identity.AgentID, OrganizationalUnit: []string{string(identity.AgentType)}}, DNSNames: []string{identity.DID.String()}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, KeyUsage: x509.KeyUsageDigitalSignature}
	der, err = x509.CreateCertificate(rand.Reader, leaf, root, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	identity.CertX509PEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	key, err := nhi.GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	identity.JWTToken, err = nhi.IssueJWT(key, identity, "test-session", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registry := ans.NewRegistry()
	directory := ans.NewService(ans.DefaultConfig(), registry, nil)
	entry, err := directory.Register(t.Context(), ans.RegisterRequest{Name: "test-blue", AgentID: identity.AgentID, AgentType: string(identity.AgentType), DID: identity.DID.String(), CertX509PEM: identity.CertX509PEM, AllowedActions: []string{"blue.analyze"}, AllowedAssets: []string{"10.20.0.3"}})
	if err != nil {
		t.Fatal(err)
	}
	monitor, err := abm.NewMonitor(abm.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	identities := &testIdentities{identity: identity, key: key}
	adapter, err := New(Config{Directory: directory, Identities: identities, Monitor: monitor, CACertPEM: rootPEM})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{registry, adapter, directory, identities, entry, monitor, lab.Action{AgentID: identity.AgentID, TaskID: "7", Name: "blue.analyze", Target: "10.20.0.3"}}
}
func TestIdentityAcceptsBoundCertificateAndToken(t *testing.T) {
	f := setup(t)
	if err := f.adapter.CheckIdentity(t.Context(), f.action); err != nil {
		t.Fatal(err)
	}
	if err := f.adapter.CheckMonitor(t.Context(), f.action); err != nil {
		t.Fatal(err)
	}

	scope, _ := network.NewScope("10.20.0.0/24")
	if _, err := f.adapter.Gate(scope); err == nil {
		t.Fatal("missing OPSEC and ethics accepted")
	}
	denied := errors.New("OPSEC unavailable")
	f.adapter.cfg.Audit = func(context.Context, lab.Action, error) error { return nil }
	f.adapter.cfg.OPSEC = func(context.Context, lab.Action) error { return denied }
	f.adapter.cfg.Ethics = func(context.Context, lab.Action) error { return nil }
	gate, err := f.adapter.Gate(scope)
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.Authorize(t.Context(), f.action); !errors.Is(err, denied) {
		t.Fatalf("OPSEC was bypassed: %v", err)
	}
}
func TestIdentityRejectsEveryBindingAndPermissionFailure(t *testing.T) {
	cases := map[string]func(*fixture){
		"unregistered":          func(f *fixture) { f.action.AgentID = "missing" },
		"unknown status":        func(f *fixture) { f.entry.Status = "unknown" },
		"quarantined":           func(f *fixture) { f.entry.Status = ans.StatusQuarantined },
		"expired directory":     func(f *fixture) { f.entry.ExpiresAt = time.Now().Add(-time.Second) },
		"directory permission":  func(f *fixture) { f.entry.AllowedActions = nil },
		"directory scope":       func(f *fixture) { f.entry.AllowedAssets = []string{"*"} },
		"identity permission":   func(f *fixture) { f.identities.identity.AllowedActions = nil },
		"identity scope":        func(f *fixture) { f.identities.identity.AllowedAssets = []string{"10.21.0.0/24"} },
		"revoked":               func(f *fixture) { now := time.Now(); f.identities.identity.RevokedAt = &now },
		"expired identity":      func(f *fixture) { f.identities.identity.ExpiresAt = time.Now().Add(-time.Second) },
		"future identity":       func(f *fixture) { f.identities.identity.IssuedAt = time.Now().Add(time.Hour) },
		"wrong DID":             func(f *fixture) { f.entry.DID = nhi.NewDID().String() },
		"wrong type":            func(f *fixture) { f.entry.AgentType = "RED" },
		"wrong certificate":     func(f *fixture) { f.entry.CertX509PEM = []byte("untrusted") },
		"malformed certificate": func(f *fixture) { f.identities.identity.CertX509PEM = []byte("invalid") },
		"wrong root":            func(f *fixture) { f.adapter.roots = x509.NewCertPool() },
		"missing token":         func(f *fixture) { f.identities.identity.JWTToken = "" },
		"bad signature":         func(f *fixture) { f.identities.key = make([]byte, 32) },
		"outdated token permission": func(f *fixture) {
			id := *f.identities.identity
			id.AllowedActions = nil
			f.identities.identity.JWTToken, _ = nhi.IssueJWT(f.identities.key, &id, "", time.Hour)
		},
		"outdated token scope": func(f *fixture) {
			id := *f.identities.identity
			id.AllowedAssets = nil
			f.identities.identity.JWTToken, _ = nhi.IssueJWT(f.identities.key, &id, "", time.Hour)
		},
		"expired token": func(f *fixture) {
			f.identities.identity.JWTToken, _ = nhi.IssueJWT(f.identities.key, f.identities.identity, "", -time.Hour)
		},
		"provider failure": func(f *fixture) { f.identities.failure = errors.New("credential secret-token") },
		"public target":    func(f *fixture) { f.action.Target = "8.8.8.8" },
		"hostname":         func(f *fixture) { f.action.Target = "example.test" },
		"invalid task":     func(f *fixture) { f.action.TaskID = "-1" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := setup(t)
			mutate(f)
			if err := f.registry.Add(f.entry); err != nil {
				t.Fatal(err)
			}
			err := f.adapter.CheckIdentity(t.Context(), f.action)
			if err == nil {
				t.Fatal("invalid authorization accepted")
			}
			if strings.Contains(err.Error(), "secret-token") {
				t.Fatal("credential leaked")
			}
		})
	}
}
func TestRevocationAndStopAreRechecked(t *testing.T) {
	f := setup(t)
	if err := f.adapter.CheckIdentity(t.Context(), f.action); err != nil {
		t.Fatal(err)
	}
	f.monitor.Stop(f.action.AgentID, "test stop")
	if err := f.adapter.CheckMonitor(t.Context(), f.action); err == nil {
		t.Fatal("stopped agent accepted")
	}
	f = setup(t)
	now := time.Now()
	f.identities.identity.RevokedAt = &now
	if err := f.adapter.CheckMonitor(t.Context(), f.action); err == nil {
		t.Fatal("revocation between checks accepted")
	}
	f = setup(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := f.adapter.CheckIdentity(ctx, f.action); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestAssetGrantsDoNotAuthorizeWildcardsOrPublicPrefixes(t *testing.T) {
	for _, grant := range []string{"*", "10.20.*", "lab.test", "0.0.0.0/0", "10.0.0.0/1", "::/0", "::ffff:10.20.0.3"} {
		if allowsAsset([]string{grant}, "10.20.0.3") {
			t.Fatalf("unsafe grant %q accepted", grant)
		}
	}
	for _, grant := range []string{"10.20.0.3", "10.20.0.0/24"} {
		if !allowsAsset([]string{grant}, "10.20.0.3") {
			t.Fatal("valid grant rejected", grant)
		}
	}
}

type observedMonitor struct {
	*abm.DefaultMonitor
	calls      int
	violations []*abm.Violation
	config     *abm.Config
}

func (m *observedMonitor) Observe(agentID, agentType, action, target string, metadata map[string]string) []*abm.Violation {
	m.calls++
	m.DefaultMonitor.Observe(agentID, agentType, action, target, metadata)
	return m.violations
}
func (m *observedMonitor) Config() abm.Config {
	if m.config != nil {
		return *m.config
	}
	return m.DefaultMonitor.Config()
}
func TestMonitorCannotSilentlyDisableOrIgnoreHighViolations(t *testing.T) {
	f := setup(t)
	observed := &observedMonitor{DefaultMonitor: f.monitor}
	f.adapter.cfg.Monitor = observed
	if err := f.adapter.CheckMonitor(t.Context(), f.action); err != nil || observed.calls != 1 {
		t.Fatalf("ABM was not observed: %v", err)
	}
	observed.violations = []*abm.Violation{{Severity: abm.SeverityHigh}}
	if err := f.adapter.CheckMonitor(t.Context(), f.action); err == nil {
		t.Fatal("high violation accepted without quarantine")
	}
	unsafe := abm.DefaultConfig()
	unsafe.Enabled = false
	observed.config = &unsafe
	calls := observed.calls
	if err := f.adapter.CheckMonitor(t.Context(), f.action); err == nil || observed.calls != calls {
		t.Fatal("disabled ABM accepted")
	}
	if _, err := New(Config{Directory: f.directory, Identities: f.identities, Monitor: observed}); err == nil {
		t.Fatal("missing CA accepted")
	}
}
