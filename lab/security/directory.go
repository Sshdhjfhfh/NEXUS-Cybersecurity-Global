package security

import (
	"context"
	"fmt"
	"github.com/nexus-ia/nexus/ans"
)

// PersistentDirectory reads the authoritative ANS store for every action.
// It avoids cached grants surviving an update made by another process and works
// after restart without first rebuilding the in-memory agent-ID index.
type PersistentDirectory struct{ store *ans.Store }

func NewPersistentDirectory(store *ans.Store) (*PersistentDirectory, error) {
	if store == nil {
		return nil, fmt.Errorf("ANS store is required")
	}
	return &PersistentDirectory{store: store}, nil
}
func (d *PersistentDirectory) LookupByAgentID(ctx context.Context, id string) (*ans.LookupResult, error) {
	entry, err := d.store.GetEntryByAgentID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ans.LookupResult{Found: entry != nil, Entry: entry}, nil
}

var _ Directory = (*PersistentDirectory)(nil)
