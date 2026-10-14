package server

import (
	"context"
	"testing"
)

// newManagedTestServer ensures API fixtures do not leave schedulers, delivery
// loops or agent runtimes running against the next test's database settings.
func newManagedTestServer(t *testing.T, m *Manager, skillDir, dataDir, keyDir string) *Server {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	s := New(ctx, m, skillDir, dataDir, keyDir)
	t.Cleanup(func() {
		cancel()
		s.engine.runtimeMu.Lock()
		ids := make([]string, 0, len(s.engine.runtimes))
		for id := range s.engine.runtimes {
			ids = append(ids, id)
		}
		s.engine.runtimeMu.Unlock()
		for _, id := range ids {
			s.engine.StopTask(id)
		}
		s.archiveWG.Wait()
		<-s.side.done
	})
	return s
}
