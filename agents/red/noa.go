// Nexus IA v6.0 — Absorbido de ARTEX/agent
// Upstream: github.com/mhtsec/ARTEX/agent
// Licencia: AGPL-3.0
//
// ⚠️ REESCRITO: La versión original dependía de norma/noaadapter
// (feature experimental de compresión de contexto). Nexus IA
// implementará su propio sistema de compresión en el módulo
// `core/compaction/` cuando sea necesario. Por ahora este módulo
// es un stub inerte que no interfiere con el sistema.

package agent

import (
	"log"
)

// noaWarn returns a diagnostics sink tagging non-fatal noa messages.
// Mantenido para compatibilidad de API con el código absorbido.
func noaWarn(session string) func(string) {
	return func(msg string) {
		log.Printf("[noa-stub] %s: %s", session, msg)
	}
}

// enableNoa es un stub inerte. La funcionalidad de NOA (compresión de
// contexto dirigida por modelo) está deshabilitada en Nexus IA v6.0.
//
// En el futuro, cuando implementemos nuestro propio sistema de
// compresión avanzada, esta función lo conectará. Por ahora, no hace
// nada y deja que el sistema de compaction estándar (`core/compaction/`)
// siga funcionando.
//
// Parámetros (ignorados en el stub):
//   - opts: opciones del agentcore
//   - enabled: función que indica si NOA está habilitado
//   - archiveRoot: directorio raíz para archivos comprimidos
//   - sessionID: ID único de la sesión
//   - onWarn: callback para warnings
func enableNoa(
	opts any,
	enabled func() bool,
	archiveRoot string,
	sessionID string,
	onWarn func(string),
) {
	if enabled != nil && enabled() {
		// NOA está habilitado pero no implementado en Nexus IA
		if onWarn != nil {
			onWarn("noa: feature no implementada en Nexus IA v6.0, usando compresión estándar")
		}
	}
	// No hace nada — deja que core/compaction/ haga su trabajo
}
