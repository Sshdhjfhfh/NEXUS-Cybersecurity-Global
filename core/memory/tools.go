// Nexus IA v6.0 — Absorbido de norma/memory
// Upstream: github.com/Autumn-27/norma/memory
// Licencia: AGPL-3.0

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nexus-ia/nexus/core/permission"
	"github.com/nexus-ia/nexus/core/tool"
)

// Tools returns the RecallMemory and RecordMemory tools bound to a store.
func Tools(s *Store) []tool.CoreTool {
	return []tool.CoreTool{RecallTool(s), RecordTool(s)}
}

// RecallTool lets the agent read memories relevant to a query.
func RecallTool(s *Store) tool.CoreTool {
	return tool.Build(tool.Spec{
		Name:        "RecallMemory",
		Description: "Busca notas relevantes en la memoria de esta tarea o conversación. Comprueba la vigencia de las evidencias antes de usar las conclusiones.",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"query": map[string]any{"type": "string", "description": "Qué deseas recordar."}},
			"required":   []any{"query"},
		},
		ReadOnly:   func(json.RawMessage) bool { return true },
		Concurrent: func(json.RawMessage) bool { return true },
		Permissions: func(context.Context, json.RawMessage, permission.Context) permission.Decision {
			return permission.Allowed()
		},
		Run: func(ctx context.Context, input json.RawMessage, _ *tool.ToolContext) (tool.Result, error) {
			var in struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return tool.Result{}, err
			}
			mems := s.RelevantContext(ctx, in.Query, 5)
			if err := ctx.Err(); err != nil {
				return tool.Result{}, err
			}
			if len(mems) == 0 {
				return tool.Text("No se encontraron memorias relevantes."), nil
			}
			return tool.Text(RenderForInjection(mems, time.Now())), nil
		},
	})
}

// RecordTool lets the agent persist a durable fact to memory.
func RecordTool(s *Store) tool.CoreTool {
	return tool.Build(tool.Spec{
		Name:        "RecordMemory",
		Description: "Guarda una lección verificada, una preferencia o una decisión para futuras sesiones de esta tarea o conversación. Incluye su evidencia; no guardes secretos ni hipótesis como hechos.",
		Prompt:      "Tipos: user (preferencias), feedback (lecciones de trabajo), project (decisiones y restricciones), reference (referencias). Usa un nombre corto y una descripción de una línea.",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "description": "Título breve de la memoria."},
				"description": map[string]any{"type": "string", "description": "Resumen de una línea."},
				"type":        map[string]any{"type": "string", "description": "user | feedback | project | reference"},
				"content":     map[string]any{"type": "string", "description": "Hecho o lección que recordar, con evidencia (Markdown)."},
			},
			"required": []any{"name", "content"},
		},
		ReadOnly:   func(json.RawMessage) bool { return false },
		Concurrent: func(json.RawMessage) bool { return false },
		Permissions: func(context.Context, json.RawMessage, permission.Context) permission.Decision {
			return permission.Allowed()
		},
		Run: func(_ context.Context, input json.RawMessage, _ *tool.ToolContext) (tool.Result, error) {
			var in struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Type        string `json:"type"`
				Content     string `json:"content"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return tool.Result{}, err
			}
			if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Content) == "" {
				return tool.Errorf("Se requieren nombre y contenido."), nil
			}
			m := Memory{Name: in.Name, Description: in.Description, Type: Type(in.Type), Content: in.Content}
			if err := s.Save(m); err != nil {
				return tool.Errorf("No se pudo guardar la memoria: " + err.Error()), nil
			}
			return tool.Text(fmt.Sprintf("Memoria %q guardada.", in.Name)), nil
		},
	})
}
