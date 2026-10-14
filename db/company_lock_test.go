// Nexus IA v6.0 — Absorbido de ARTEX/db
// Upstream: github.com/mhtsec/ARTEX/db
// Licencia: AGPL-3.0

package db

import "testing"

func TestCompanyScopeMutationLockDoesNotReuseInfrastructureLocks(t *testing.T) {
	reserved := map[string]int64{
		"schema migration":         7337741001,
		"cross-package test suite": 7337741002,
	}
	for name, key := range reserved {
		if companyScopeMutationLock == key {
			t.Fatalf("company scope mutation lock reuses the %s advisory lock key %d", name, key)
		}
	}
}
