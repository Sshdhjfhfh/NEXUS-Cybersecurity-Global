# NEXUS IA v6.0 — Especificación Técnica

> **Documento de referencia.** Todo el código de Nexus debe ser coherente con esta spec.
> Basado en el PDF `NEXUS_IA_v6.pdf` + auditoría del fork ARTEX v0.3.15.

---

## 1. Identidad del proyecto

| Campo | Valor |
|---|---|
| Nombre | NEXUS IA v6.0 |
| Tipo | Sistema de Investigación Adversarial Autónomo |
| Uso | **SOLO laboratorio controlado** |
| Licencia | AGPL-3.0 (heredada de ARTEX) |
| Base | Fork de [ARTEX](https://github.com/mhtsec/ARTEX) + SDK [norma](https://github.com/Autumn-27/norma) v0.4.3 |
| Tamaño objetivo | ~250.000 LOC |
| Idiomas | 10 (en zh hi es ar fr bn pt id ru) |
| Fases | 6 |
| Módulos | 47 |
| Niveles | 5 |

---

## 2. Estructura de niveles

### Nivel 0 — Interfaz
- `i18n/` — 10 idiomas + RTL árabe
- `llm_gateway/` — multi-proveedor + failover <1s + TUI + Web
- `web/` — Dashboard Next.js

### Nivel 1-2 — Base
- `phantom/` — 10 capas de anonimato
- `agents/red/` — RED agent (fork ARTEX)
- `agents/blue/` — BLUE agent (SENTINEL)
- `agents/prediction/` — prediction engine
- `swarm/` — 8 agentes especializados
- `watchdog/` — resiliencia continua
- `lab/` — DVWA + Metasploitable + VulnHub + AD + red virtual

### Nivel 3 — Auto-mejora
- `evolve/` — Self-play + Metacognición + Hyperagent
- `memory/` — ChromaDB
- `knowledge_bus/` — Redis Streams cifrado

### Nivel 4 — Colectivo + Frontera científica
- `graph_rag/` — extiende doble graph de ARTEX
- `vckg/` — Vulnerability Causal Knowledge Graph
- `zero_day/` — AFL++ guiado por LLM
- `symbolic_ai/` — Z3 + Prolog
- `ai_scientist/` — papers LaTeX
- `cti_feed/` — NVD + MITRE + CISA
- `digital_twin/` — gemelo digital
- `xai/` — Explainability + STIX 2.1
- `federated/` — PySyft + differential privacy

### Nivel 5 — Seguridad y Gobernanza (NUEVO v6)
- `nhi_manager/` — Non-Human Identity Manager
- `abm/` — Agent Behavior Monitor (INMUTABLE)
- `ctf_engine/` — CTF autónomo
- `zero_trust/` — Framework MAESTRO
- `rsi_detector/` — Recursive Self-Improvement Detector
- `ans/` — Agent Name Service
- `adversarial_rt/` — Adversarial Robustness Tester
- `governance/` — Reporte semanal firmado

### Módulos inmutables
`opsec`, `circuit_breaker`, `watchdog`, `ethical_boundary`, `llmjacking_guard`, `oracle_defense`, `abm`, `zero_trust`

---

## 3. Restricciones absolutas

1. Toda acción: `opsec.Verify()` + `ethical_boundary.Check()` + `abm.Monitor()`
2. Todo tráfico: `phantom/proxy` → `onion` → `mixnet` (nunca directo)
3. 100% offline posible con modelos locales Ollama
4. Logs internos **SIEMPRE en inglés**
5. `ethical_boundary`: SOLO `qwen3:8b` local, nunca nube, sin excepciones
6. `ethical_boundary` STOP: ningún módulo puede anularlo
7. `EVOLVE` variantes: sandbox Docker antes de producción
8. Zero-day: validación humana antes de uso fuera del lab
9. API keys: solo `.env` + Docker secrets, nunca en código
10. NHI: cada agente autenticado antes de cada acción vía `ans/`
11. `adversarial_rt`: sus ataques nunca salen de la red interna NEXUS
12. `governance/` reporte: firmado Dilithium, inmutable, semanal

---

## 4. Herencia de ARTEX + norma

### De `norma` (SDK de agentes)
Absorber en `core/`:
- `agentcore` → `core/agent/` (reescrito)
- `tool` → `core/tool/`
- `harness` → `core/harness/`
- `compaction` → `core/compaction/`
- `skill` → `core/skill/` (extendido)
- `memory` → `core/memory/` (reescrito con ChromaDB)
- `transcript` → `core/transcript/`
- `mcp` → `core/mcp/`
- `subagent` → `core/subagent/` (extendido)
- `coordinator` → `core/coordinator/` (reescrito para swarm)
- `permission` → `core/permission/` (extendido con zero_trust)
- `hook` → `core/hook/`
- `plan` → `core/plan/` (reescrito para swarm)

Descartar: `llm/`, `noa/`, `noaadapter/`, `cmd/`, `examples/`.

### De ARTEX
Absorber:
- `server/` → `orchestrator/`
- `db/` → `graph_rag/` + `vckg/` (extendido)
- `llmpool/` → `llm_gateway/pool/`
- `llmrec/` → `observability/llmrec/`
- `intercept/` → `ethical_boundary/intercept/`
- `guard/` → `ethical_boundary/guard/`
- `report/` → `governance/report/`
- `notify/` → `governance/notify/`
- `traffic/` → `lab/proxy/`
- `enrich/` → `lab/enrich/`
- `evidence/` → `evidence/`
- `mcphttp/` → `core/mcp/http/`
- `selfupdate/` → `selfimprove/update/`
- `config/` → `config/`
- `skills/` → `skills/`
- `web/` → `web/`

Reescribir:
- `agent/` (planner, worker, tools) → `agents/red/` + `swarm/`

Descartar:
- `cmd/artex/` → crear `cmd/nexus/` propio

---

## 5. Hardware y tiers

| Tier | VRAM | RAM | Modelos | Uso |
|---|---|---|---|---|
| nano | 0-2 GB | 4-8 GB | qwen3:1.7b | Mínimo |
| micro | 2-4 GB | 8-12 GB | qwen3:4b | **Dev (Saul)** |
| standard | 4-8 GB | 12-16 GB | qwen3:8b | PC prod |
| pro | 8-16 GB | 16-32 GB | qwen3:14b | Workstation |
| ultra | 16-32 GB | 32-64 GB | qwen3:32b | Server |

**Regla inmutable**: `ethical_boundary` siempre usa el modelo más pequeño que funcione — nunca nube.

---

## 6. Fases de construcción (PDF)

| Fase | Pasos | Contenido |
|---|---|---|
| FASE 1 | 1-8 | Interfaz + Infra (proto, docker-compose, i18n, llm_gateway, phantom, watchdog, ans, lab) |
| FASE 2 | 9-14 | Seguridad + Grafos (nhi_manager, zero_trust, abm, graph_rag, vckg, memory, knowledge_bus) |
| FASE 3 | 15-20 | Agentes Core (red, blue, prediction, swarm, digital_twin, ctf_engine) |
| FASE 4 | 21-30 | Auto-mejora + Ciencia (evolve, evoexploit, selfimprove, tom, rsi_detector, zero_day, symbolic_ai, cti_feed, ai_scientist, nexus_bench) |
| FASE 5 | 31-40 | Protección + Frontera (llmjacking_guard, oracle_defense, adversarial_rt, xai, federated, governance, research, ethical_boundary, web) |
| FASE 6 | 41-46 | Calidad + Baseline (tests, chaos, i18n, nexus_bench baseline) |

---

## 7. Métricas de éxito (PDF)

| Métrica | Objetivo |
|---|---|
| i18n idiomas completos | 10/10 al 100% |
| Cambio de idioma en caliente | < 1 segundo |
| LLM Gateway failover | < 1 segundo |
| NHI autenticación | 100% acciones con cert |
| ABM detección rogue | < 2 segundos |
| CTF percentil humano | > percentil 25 en 30 días |
| Zero Trust acciones sin auth | = 0 |
| RSI monitoreo | Alerta antes de RSI real |
| Adversarial RT debilidades | 0 sin patch |
| Governance reportes | 1 semanal firmado |
| Red agent éxito DVWA | > 80% |
| Swarm vs agente individual | > 40% más rápido |
| Prediction accuracy | > 60% tras 50 sesiones |
| Mejora autónoma sin humano | > 15% en 20 sesiones |
| Digital twin fidelidad N2 | > 70% |
| Zero-day pipeline | 1 crash / 10h fuzzing |
| AI Scientist papers | 1 borrador / 100 sesiones |
| Recuperación módulo | < 30 segundos |
| Recuperación sistema | < 60 segundos |
| Latencia proxy phantom | < 200 ms |
| Credenciales en código | 0 |
| Salidas del lab | 0 |
| NEXUS-Bench baseline | v1.0 establecido |

---

## 8. Licencia y créditos

- **Licencia**: AGPL-3.0 (heredada de ARTEX).
- **Créditos**: ARTEX por mhtsec, norma por Autumn-27.
- **Uso**: SOLO laboratorio controlado. Prohibido uso en producción o contra sistemas reales.
- **Compliance**: Cumple con AGPL-3.0 (publicación de fuente si se ofrece como servicio).
