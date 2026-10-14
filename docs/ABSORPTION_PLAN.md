
# Plan de Absorción de ARTEX + norma

> **Objetivo**: tener Nexus v6.0-alpha funcionando en 4-6 semanas, sin depender de ARTEX ni norma como módulos externos.

---

## Números reales (auditoría 2026-10-14)

### normA SDK
| Subpaquete | LOC | Acción |
|---|---:|---|
| tool/ | 6.565 | Absorber → core/tool/ |
| harness/ | 2.845 | Absorber → core/harness/ |
| compaction/ | 2.280 | Absorber → core/compaction/ |
| agentcore/ | 1.411 | Absorber + reescribir → core/agent/ |
| skill/ | 502 | Absorber → core/skill/ |
| memory/ | 476 | Reescribir con ChromaDB → core/memory/ |
| transcript/ | 451 | Absorber → core/transcript/ |
| mcp/ | 414 | Absorber → core/mcp/ |
| subagent/ | 252 | Absorber + extender → core/subagent/ |
| coordinator/ | 222 | Reescribir para swarm → core/coordinator/ |
| permission/ | 209 | Absorber + extender → core/permission/ |
| hook/ | 206 | Absorber → core/hook/ |
| plan/ | 193 | Reescribir para swarm → core/plan/ |
| llm/ | 4.039 | **DESCARTAR** |
| noa/ | 9.850 | **DESCARTAR** |
| noaadapter/ | 6.527 | **DESCARTAR** |
| cmd/ | 196 | **DESCARTAR** |
| examples/ | 588 | **DESCARTAR** |
| **TOTAL ÚTIL** | **~16.000** | Absorber |
| **TOTAL DESCARTAR** | **~21.200** | Fuera |

### ARTEX (código propio)
| Paquete | LOC | Acción |
|---|---:|---|
| server/ | 34.755 | Absorber → orchestrator/ |
| db/ | 30.195 | Absorber → graph_rag/ + vckg/ |
| agent/ | 10.391 | Reescribir → agents/red/ + swarm/ |
| notify/ | 5.055 | Absorber → governance/notify/ |
| traffic/ | 4.158 | Absorber → lab/proxy/ |
| intercept/ | 1.572 | Absorber → ethical_boundary/intercept/ |
| selfupdate/ | 1.411 | Absorber → selfimprove/update/ |
| llmpool/ | 920 | Absorber → llm_gateway/pool/ |
| evidence/ | 787 | Absorber → evidence/ |
| llmrec/ | 778 | Absorber → observability/llmrec/ |
| mcphttp/ | 685 | Absorber → core/mcp/http/ |
| report/ | 299 | Absorber → governance/report/ |
| enrich/ | 292 | Absorber → lab/enrich/ |
| guard/ | 236 | Absorber → ethical_boundary/guard/ |
| config/ | 234 | Absorber → config/ |
| cmd/ | 179 | **DESCARTAR** |
| **TOTAL Go** | **93.282** | 321 archivos |
| **Web Next.js** | **56.088** | Absorber → web/ |

---

## Cronograma de absorción (6 semanas)

### Semana 1 — Preparación
- [ ] Crear `~/nexus/core/` con estructura completa
- [ ] Crear `~/nexus/go.mod` propio (`module github.com/nexus-ia/nexus`)
- [ ] Copiar `norma/` a `~/nexus/vendor/norma/` (referencia)
- [ ] Copiar ARTEX a `~/nexus/vendor/artex/` (referencia)
- [ ] Configurar `.gitignore` para excluir `vendor/`

### Semana 2 — Absorber core/ (paquetes pequeños)
- [ ] `hook/` → `core/hook/` (206 LOC)
- [ ] `permission/` → `core/permission/` (209)
- [ ] `plan/` → `core/plan/` (193, reescribir)
- [ ] `coordinator/` → `core/coordinator/` (222, reescribir)
- [ ] `subagent/` → `core/subagent/` (252)
- [ ] `transcript/` → `core/transcript/` (451)
- [ ] `mcp/` → `core/mcp/` (414)
- [ ] Verificación: `go build ./core/...`

### Semana 3 — Absorber core/ (paquetes medianos)
- [ ] `skill/` → `core/skill/` (502, extender)
- [ ] `memory/` → `core/memory/` (reescribir con ChromaDB)
- [ ] `agentcore/` → `core/agent/` (1.411, reescribir)

### Semana 4 — Absorber core/ (paquetes grandes)
- [ ] `tool/` → `core/tool/` (6.565)
- [ ] `compaction/` → `core/compaction/` (2.280)
- [ ] `harness/` → `core/harness/` (2.845)

### Semana 5 — Absorber ARTEX (base)
- [ ] `server/` → `orchestrator/` (34.755)
- [ ] `db/` → `graph_rag/` + `vckg/` (30.195)
- [ ] `config/` → `config/` (234)
- [ ] `llmpool/` → `llm_gateway/pool/` (920)
- [ ] `llmrec/` → `observability/llmrec/` (778)

### Semana 6 — Absorber ARTEX (resto)
- [ ] `agent/` → `agents/red/` + `swarm/` (10.391, reescribir)
- [ ] `notify/` → `governance/notify/` (5.055)
- [ ] `traffic/` → `lab/proxy/` (4.158)
- [ ] `intercept/` + `guard/` → `ethical_boundary/` (1.808)
- [ ] `report/` → `governance/report/` (299)
- [ ] `evidence/` → `evidence/` (787)
- [ ] `mcphttp/` → `core/mcp/http/` (685)
- [ ] `enrich/` → `lab/enrich/` (292)
- [ ] `selfupdate/` → `selfimprove/update/` (1.411)
- [ ] `skills/` → `skills/`
- [ ] `web/` → `web/` (56.088)
- [ ] Eliminar `vendor/` después de absorber
- [ ] `go build ./...` verde

---

## Estrategia de absorción (por archivo)

Para cada paquete:
1. **Copiar** carpeta a destino.
2. **Reemplazar imports**:
   - `github.com/Autumn-27/norma/X` → `github.com/nexus-ia/nexus/core/X`
   - `github.com/Autumn-27/artex/X` → `github.com/nexus-ia/nexus/X`
3. **Añadir header** a cada archivo `.go`:
   ```go
   // Nexus IA v6.0 — Absorbido de ARTEX/norma
   // Upstream: github.com/Autumn-27/norma/<pkg>
   // Licencia: AGPL-3.0