// Nexus IA v6.0 — Helpers internos del NHI Manager
package nhi_manager

import (
	"crypto/rand"
	"fmt"
)

// readRandom llena b con bytes aleatorios criptográficamente seguros.
func readRandom(b []byte) (int, error) {
	n, err := rand.Read(b)
	if err != nil {
		return n, fmt.Errorf("read random: %w", err)
	}
	return n, nil
}
