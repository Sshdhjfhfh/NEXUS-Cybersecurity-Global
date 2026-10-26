// Nexus IA — Adaptado de norma v0.4.3 (ARTEX a9d11cb).
// Se conservan los avisos y condiciones de los autores originales.

package noa

// CompressToolDescription is what the model reads next to the tool's schema.
//
// The brake in the second half is load-bearing. Tool schemas ship with every
// request whether or not a nudge fired, so the model can always see that
// Compress exists. Without an explicit instruction to wait, it may call the
// tool on its own initiative — and a summary written without the rules is an
// irreversible loss, because the range it replaced is gone from the view.
const CompressToolDescription = `Sustituye rangos antiguos de la conversación por resúmenes, por defecto en español. Primero se archivan los originales en disco. ` +
	`No la invoques por iniciativa propia: espera a que el gestor de contexto lo solicite y sigue sus reglas. ` +
	`Conserva literalmente evidencias, rutas, identificadores y restricciones; resumir sin esas reglas puede perder información esencial.`

// CompressToolSchema is the tool's JSON Schema.
//
// Note what is NOT here: "content" carries no "type": "array".
//
// Providers without strict tool support (vLLM, some Qwen deployments) stringify
// nested array arguments. A schema that insists on an array rejects that shape
// before ParseCompressArgs ever sees it — and a turn's only compression attempt
// is lost to a validation error rather than being repaired. Omitting the type
// lets anything through to the lenient parser, which knows how to recover an
// array from a string.
func CompressToolSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"topic": map[string]any{
				"type":        "string",
				"description": "Título breve opcional para los rangos que no tengan uno propio.",
			},
			"content": map[string]any{
				"description": "Uno o más rangos que resumir en bloques separados. No deben solaparse.",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"topic": map[string]any{
							"type":        "string",
							"description": "Título de 3 a 5 palabras para este rango.",
						},
						"startId": map[string]any{
							"type":        "string",
							"description": "Referencia de mensaje mNNNNN o identificador de bloque bN al inicio del rango.",
						},
						"endId": map[string]any{
							"type":        "string",
							"description": "Referencia de mensaje mNNNNN o identificador de bloque bN al final del rango, incluido.",
						},
						"summary": map[string]any{
							"type":        "string",
							"description": "Resumen autónomo en español que sustituye el rango en el contexto y conserva las evidencias literales.",
						},
					},
					"required": []any{"startId", "endId", "summary"},
				},
			},
			"summaryMaxChars": map[string]any{
				"type":        "number",
				"description": "Amplía el límite predeterminado de 20000 caracteres por resumen cuando haga falta más detalle. No recortes información crítica para ajustarte al límite.",
			},
		},
		"required": []any{"content"},
	}
}
