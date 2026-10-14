// Package security adapts existing ANS, NHI and ABM services to laboratory gates.
// It does not create identities, weaken immutable controls or supply fallback approvals.
package security

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"time"

	"github.com/nexus-ia/nexus/abm"
	"github.com/nexus-ia/nexus/ans"
	"github.com/nexus-ia/nexus/lab"
	"github.com/nexus-ia/nexus/lab/network"
	nhi "github.com/nexus-ia/nexus/nhi_manager"
)

type Directory interface {
	LookupByAgentID(context.Context, string) (*ans.LookupResult, error)
}
type Identities interface {
	GetIdentity(context.Context, string) (*nhi.NHI, error)
	VerifyJWT(context.Context, string) (*nhi.ClaimsNexus, error)
}
type Monitor interface {
	Observe(string, string, string, string, map[string]string) []*abm.Violation
	IsStopped(string) bool
	IsQuarantined(string) bool
	IsImmutable() bool
	Config() abm.Config
}

// Config requires an explicit trust anchor and both remaining host controls.
// Actions and assets must be granted independently in ANS, NHI and signed claims.
// Asset grants accept literal IPs and private CIDRs, never DNS or wildcard patterns.
type Config struct {
	Directory  Directory
	Identities Identities
	Monitor    Monitor
	CACertPEM  []byte
	OPSEC      lab.Check
	Ethics     lab.Check
	Audit      lab.Audit
}
type Adapter struct {
	cfg   Config
	roots *x509.CertPool
}

func New(cfg Config) (*Adapter, error) {
	if cfg.Directory == nil || cfg.Identities == nil || cfg.Monitor == nil {
		return nil, fmt.Errorf("ANS, NHI and ABM services are required")
	}
	roots := x509.NewCertPool()
	block, rest := pem.Decode(cfg.CACertPEM)
	if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, fmt.Errorf("exactly one trusted CA certificate is required")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !cert.IsCA || !cert.BasicConstraintsValid || cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		return nil, fmt.Errorf("invalid laboratory CA certificate")
	}
	roots.AddCert(cert)
	cfg.CACertPEM = nil
	if !monitorReady(cfg.Monitor) {
		return nil, fmt.Errorf("enabled immutable ABM with automatic stop and quarantine is required")
	}
	return &Adapter{cfg: cfg, roots: roots}, nil
}
func monitorReady(m Monitor) bool {
	cfg := m.Config()
	return cfg.Enabled && cfg.Immutable && m.IsImmutable() && cfg.AutoStopOnCritical && cfg.AutoQuarantineOnHigh
}

// Gate never invents OPSEC or ethical decisions when the corresponding services are absent.
func (a *Adapter) Gate(scope network.Scope) (*lab.Gate, error) {
	return lab.NewAuditedGate(scope, lab.Checks{Identity: a.CheckIdentity, Monitor: a.CheckMonitor, OPSEC: a.cfg.OPSEC, Ethics: a.cfg.Ethics}, a.cfg.Audit)
}
func validateAction(action lab.Action) error {
	task, err := strconv.ParseInt(action.TaskID, 10, 64)
	addr, addrErr := netip.ParseAddr(action.Target)
	if action.AgentID == "" || len(action.AgentID) > 128 || action.Name == "" || len(action.Name) > 128 || err != nil || task <= 0 || addrErr != nil || !addr.IsPrivate() || addr.Is4In6() || addr.Zone() != "" {
		return fmt.Errorf("invalid laboratory action")
	}
	return nil
}
func allowsAsset(grants []string, target string) bool {
	address, err := netip.ParseAddr(target)
	if err != nil {
		return false
	}
	for _, grant := range grants {
		if addr, err := netip.ParseAddr(grant); err == nil && addr == address {
			return true
		}
		if prefix, err := netip.ParsePrefix(grant); err == nil {
			scope, err := network.NewScope(prefix.String())
			if err == nil && scope.Allows(target) {
				return true
			}
		}
	}
	return false
}

// identity re-reads both services on every decision, including before persistence.
// No credential or verification error from a provider is included in returned errors.
func (a *Adapter) identity(ctx context.Context, action lab.Action) (*nhi.NHI, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateAction(action); err != nil {
		return nil, err
	}
	lookup, err := a.cfg.Directory.LookupByAgentID(ctx, action.AgentID)
	if err != nil || lookup == nil || !lookup.Found || lookup.Entry == nil {
		return nil, fmt.Errorf("agent is not registered in ANS")
	}
	entry := lookup.Entry
	now := time.Now()
	if entry.AgentID != action.AgentID || entry.Status != ans.StatusActive || !now.Before(entry.ExpiresAt) || !entry.CanAct(action.Name) || !allowsAsset(entry.AllowedAssets, action.Target) {
		return nil, fmt.Errorf("ANS identity or permission rejected")
	}
	identity, err := a.cfg.Identities.GetIdentity(ctx, action.AgentID)
	if err != nil || identity == nil {
		return nil, fmt.Errorf("NHI identity is unavailable")
	}
	if identity.AgentID != entry.AgentID || string(identity.AgentType) != entry.AgentType || identity.DID.String() == "" || identity.DID.String() != entry.DID || identity.IsRevoked() || !now.Before(identity.ExpiresAt) || identity.IssuedAt.After(now) || !identity.HasPermission(nhi.ActionType(action.Name)) || !allowsAsset(identity.AllowedAssets, action.Target) {
		return nil, fmt.Errorf("NHI identity or permission rejected")
	}
	block, rest := pem.Decode(identity.CertX509PEM)
	if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, fmt.Errorf("invalid agent certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil || cert.IsCA || cert.Subject.CommonName != identity.AgentID || !slices.Contains(cert.Subject.OrganizationalUnit, string(identity.AgentType)) || !slices.Contains(cert.DNSNames, identity.DID.String()) || !bytes.Equal(entry.CertX509PEM, identity.CertX509PEM) {
		return nil, fmt.Errorf("agent certificate binding rejected")
	}
	if _, err := cert.Verify(x509.VerifyOptions{Roots: a.roots, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return nil, fmt.Errorf("agent certificate trust rejected")
	}
	if identity.JWTToken == "" {
		return nil, fmt.Errorf("agent token is required")
	}
	claims, err := a.cfg.Identities.VerifyJWT(ctx, identity.JWTToken)
	if err != nil || claims == nil {
		return nil, fmt.Errorf("agent token rejected")
	}
	if claims.Issuer != "nexus-ia" || claims.Subject != identity.AgentID || claims.AgentID != identity.AgentID || claims.AgentType != identity.AgentType || claims.DID != identity.DID.String() || claims.ID != identity.DID.String() || !slices.Contains(claims.Audience, "nexus-ia") || claims.ExpiresAt == nil || !now.Before(claims.ExpiresAt.Time) || claims.IssuedAt == nil || claims.IssuedAt.Time.After(now) || claims.NotBefore == nil || claims.NotBefore.Time.After(now) || !claims.HasPermission(nhi.ActionType(action.Name)) || !allowsAsset(claims.AllowedAssets, action.Target) {
		return nil, fmt.Errorf("agent token binding or permission rejected")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return identity, nil
}
func (a *Adapter) CheckIdentity(ctx context.Context, action lab.Action) error {
	_, err := a.identity(ctx, action)
	return err
}

// CheckMonitor validates current identity again so revocation between gate steps
// cannot reach observation. The monitor is mandatory and cannot be disabled.
func (a *Adapter) CheckMonitor(ctx context.Context, action lab.Action) error {
	identity, err := a.identity(ctx, action)
	if err != nil {
		return err
	}
	monitor := a.cfg.Monitor
	if !monitorReady(monitor) {
		return fmt.Errorf("laboratory ABM protections are unavailable")
	}
	if monitor.IsStopped(action.AgentID) || monitor.IsQuarantined(action.AgentID) {
		return fmt.Errorf("agent is stopped or quarantined")
	}
	violations := monitor.Observe(action.AgentID, string(identity.AgentType), action.Name, action.Target, map[string]string{"task_id": action.TaskID, "source": "laboratory"})
	if err := ctx.Err(); err != nil {
		return err
	}
	if monitor.IsStopped(action.AgentID) || monitor.IsQuarantined(action.AgentID) {
		return fmt.Errorf("agent stopped or quarantined during observation")
	}
	for _, violation := range violations {
		if violation == nil || violation.Severity >= abm.SeverityHigh {
			return fmt.Errorf("laboratory ABM rejected the action")
		}
	}
	return nil
}

var _ Directory = (*ans.Service)(nil)
var _ Identities = (*nhi.Manager)(nil)
var _ Monitor = (*abm.DefaultMonitor)(nil)
