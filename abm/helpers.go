// Nexus IA v6.0 — Helpers internos del ABM
package abm

import (
	"fmt"
	"os"
)

// openAppend abre un archivo en modo append (crea si no existe).
func openAppend(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open append %s: %w", path, err)
	}
	return f, nil
}
