package security

import (
	"bytes"
	"context"
	"fmt"
	nhi "github.com/nexus-ia/nexus/nhi_manager"
)

// PersistentIdentities reads current grants and lifecycle from PostgreSQL.
// Tokens remain in the issuing manager; persisted identity records contain none.
// A restarted manager must renew credentials before it can authorize actions.
type PersistentIdentities struct{ manager *nhi.Manager }

func NewPersistentIdentities(manager *nhi.Manager) (*PersistentIdentities, error) {
	if manager == nil || manager.Store() == nil {
		return nil, fmt.Errorf("persistent NHI manager is required")
	}
	return &PersistentIdentities{manager: manager}, nil
}
func (p *PersistentIdentities) GetIdentity(ctx context.Context, id string) (*nhi.NHI, error) {
	current, err := p.manager.Store().GetIdentity(ctx, id)
	if err != nil || current == nil {
		return nil, err
	}
	credential, err := p.manager.GetIdentity(ctx, id)
	if err != nil {
		return nil, err
	}
	if credential == nil || credential.AgentID != current.AgentID || credential.DID != current.DID || !bytes.Equal(credential.CertX509PEM, current.CertX509PEM) {
		return nil, fmt.Errorf("current NHI credentials are unavailable")
	}
	current.JWTToken = credential.JWTToken
	return current, nil
}
func (p *PersistentIdentities) VerifyJWT(ctx context.Context, token string) (*nhi.ClaimsNexus, error) {
	return p.manager.VerifyJWT(ctx, token)
}

var _ Identities = (*PersistentIdentities)(nil)
