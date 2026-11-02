# NEXUS: estado actual (Fase 0, diagnóstico)

Fecha: 2026-11-02. Commit: `d218b3bbdb5db4e9339f3bc13b94999d89e4eded`.
Rama: `implementation/nexus-reliability-20261026`. Entorno: WSL Ubuntu 22.04,
`go1.26.10 linux/amd64`, PostgreSQL local (127.0.0.1:55432) compartido con el
backend de desarrollo, Python 3.10, Node/Next.js 16.4.1.

Todo lo que sigue se midió en esta sesión. Las cifras de documentos anteriores
(59/59; 1.920 aprobados / 7 omitidos / 1 timeout; SBOM con 1.021 componentes;
29 componentes con avisos OSV) son **históricas** y solo se usan para comparar.
La evidencia en bruto está en [evidence/phase0/](evidence/phase0/).

## 1. Inventario

72 paquetes Go en el build (`go list ./...`). `_reference/` (16 MB de código
upstream) no se compila ni forma parte del producto.

| Directorio | Archivos Go | Archivos de test | Líneas (sin tests) | Función |
|---|---:|---:|---:|---|
| orchestrator | 94 | 70 | 36.479 | Servidor HTTP, portal de clientes, consola, motor de tareas |
| db | 64 | 50 | 23.259 | Persistencia PostgreSQL y esquema |
| core | 101 | 99 | 20.258 | Agente, LLM, herramientas, permisos, memoria, MCP |
| agents | 40 | 36 | 9.757 | RED (planner/worker/scan), BLUE (sentinel, Sigma, detectbench), predicción |
| phantom | 25 | 4 | 8.066 | Red de anonimato (onion, garlic, mixnet, obfs, stego, VPN). No la importa ningún binario del producto |
| governance | 22 | 12 | 4.121 | Informes, notificaciones, autorizaciones |
| lab | 13 | 13 | 3.506 | Puerta de laboratorio, proxy, red, engagement, seguridad |
| abm, ans, nhi_manager, zero_trust | 27 | 6 | 5.866 | Identidad de agentes y confianza cero |
| evolve, selfimprove, rsi_detector | 13 | 6 | 2.866 | Evolución de estrategias, detector RSI, actualizaciones |
| adversarial_rt, ctf_engine | 6 | 2 | 2.228 | Red team adversarial y CTF (parcialmente stubs, ver §5) |
| resto (cmd, config, evidence, i18n, llm_gateway, mcphttp, observability, swarm, ethical_boundary) | 21 | 15 | 5.113 | |

- **Web:** 255 archivos ts/tsx (Next.js 16 + React 19). Specs de Playwright:
  `cookies`, `demo-form`, `i18n`, `nav`, `notfound`, `smoke`.
- **Python:** 46 archivos (lab/academy, lab/benchmark, zero_day, scripts,
  terminal, phantom/runtime).
- **CI:** `ci.yml` y `security.yml` (govulncheck, osv-scanner, gitleaks,
  npm audit). La CI **no** ejecuta Biome.

### Componentes que pide la directiva

| Componente | Estado |
|---|---|
| `agents/red`, `agents/blue`, `swarm`, `orchestrator/server` | Presentes |
| `db/companies.go`, `db/findings.go`, `evidence`, `governance/report` | Presentes |
| `core/permission`, `evolve`, `rsi_detector`, `web` | Presentes |
| `lab/cyberrange` | **Ausente** (`lab/dvwa` y `lab/active_directory` son directorios vacíos) |
| `benchmarks` (evaluador independiente) | **Ausente**. Hay un arnés Python en `lab/benchmark`, pero no es un evaluador independiente |
| Laboratorio V8 | **Ausente** |
| Facturación (Culqi / Mercado Pago) | **Ausente**: no hay código de cobros, suscripciones ni facturas |
| API pública con claves y webhooks salientes por organización | **Ausente**. Solo existen canales de notificación internos (consola) |

## 2. Mapa de dependencias internas

Las importaciones entre directorios de primer nivel salen de `go list`.

```
orchestrator     -> agents, core, db, ethical_boundary, evidence, governance, lab, llm_gateway, mcphttp, observability, selfimprove, swarm
agents           -> core, db, ethical_boundary, evolve, lab, observability, orchestrator
cmd              -> agents, config, core, evolve, lab, orchestrator, selfimprove
lab              -> abm, ans, core, db, nhi_manager
db               -> config, governance, orchestrator
ethical_boundary -> core, db
evidence         -> db, lab
observability    -> core, db
governance       -> db
swarm            -> lab
evolve, llm_gateway, mcphttp -> core
```

Hay ciclos entre directorios: `agents ↔ orchestrator` y `db ↔ orchestrator`.
Go los permite porque los subpaquetes son distintos, pero indican capas mezcladas.
`phantom` no aparece en el mapa: ningún binario del producto lo importa.

## 3. Línea base de pruebas (medida hoy)

| Suite | Resultado | Evidencia |
|---|---|---|
| Go `go test ./... -count=1 -cover -p 4` (112 s) | **1.652 aprobados, 1 fallido, 6 omitidos** (tests de primer nivel). 71 paquetes aprobados, 1 fallido (`db`) | `gotest-summary.txt` |
| `go test ./db` aislado, 3 repeticiones | 1 aprobada, **2 fallidas** (3 tests) | §4, F2 |
| `TestWebFetchProxyFallback` ×20 | 20/20 aprobados (0,76 s) | §4, F4 |
| Python (unittest, 6 suites) | **58 aprobados, 1 error** (`lab/benchmark`) | `python.txt` |
| Batería E2E del portal (13 suites contra el backend real) | 591/591 (ejecutada sobre `d218b3b` en la sesión anterior) | alt_run |
| Web `tsc --noEmit` | 0 errores | |
| Web Biome (`biome check src`) | **153 errores**, 167 avisos y 1.177 avisos informativos. 93 errores en 37 archivos ya commiteados y 60 en 11 archivos con cambios locales del usuario | |
| Playwright | No ejecutado en esta fase | |

**Tests omitidos (6), todos por dependencias externas ausentes:**
- `TestLoadRealSigmaRules`: necesita `NEXUS_SIGMA_DIR`.
- `TestHTTPXAuthorizesThenProbes`: httpx no está instalado.
- `TestHowToCompressRulesMatchesUpstream` y `TestTierRulesAreVerbatim`: falta el checkout de acp-kernel.
- `TestGrepViaRipgrepIfAvailable`: rg no está instalado.
- `TestLiveContextReview`: necesita un modelo externo privado.

### Comparación con las cifras históricas

| Métrica | Histórico (2026-10-26) | Hoy | Nota |
|---|---|---|---|
| Tests Go | 1.920 aprobados / 1 fallido / 7 omitidos (`-p 1`, contaba subtests) | 1.652 / 1 / 6 (solo tests de primer nivel) | **No son comparables**: el método de conteo es distinto |
| Paquetes Go | 55 aprobados / 1 fallido / 14 sin tests | 71 aprobados / 1 fallido | Hoy, 14 paquetes tienen 0 % de cobertura |
| Fallo conocido | `TestWebFetchProxyFallback` (timeout) | Aprueba 21 de 21 veces | No hubo cambio de código desde `c9b8814`. El timeout histórico parece ambiental (sandbox sin red). Queda sin reproducir, no "corregido" |
| SBOM | 1.021 componentes | 188 módulos Go + 725 entradas npm (669 en el árbol de producción) | Hoy no se regeneró el SBOM CycloneDX (pendiente de Fase 1) |
| OSV | 29 componentes con avisos | 21 paquetes (1 Go, 20 npm), 85 avisos | Ver [security-audit.md](../security/security-audit.md) |

## 4. Fallos encontrados

**F1. Deadlock entre la aplicación del esquema y las escrituras normales.**
- *Qué pasa:* `db.Open` ejecuta `schema.sql` completo en cada arranque. Ese script incluye `DROP TRIGGER IF EXISTS` y `CREATE OR REPLACE`, que toman locks `AccessExclusive`.
- *Evidencia:* el log de PostgreSQL a las 10:24:39 muestra `INSERT INTO activity` esperando por `RowShareLock` mientras la migración de otro proceso espera por `AccessExclusiveLock`. Eso hizo fallar `TestActivityPageSessions`.
- *Por qué importa en producción:* el advisory lock 7337741001 solo serializa migraciones entre sí, no frente al tráfico. Arrancar una segunda instancia (rolling deploy o un reinicio con tráfico) puede abortar transacciones de usuarios o quedar en reintentos.
- *Prioridad:* P1.

**F2. Los tests de `db` dejan filas tras de sí y fallan al repetirse.**
- *Qué pasa:* `t.Cleanup(func() { d.Exec(DELETE …) })` corre **después** de `defer d.Close()`, así que el DELETE se ejecuta sobre una conexión cerrada y falla en silencio.
- *Evidencia:* tras una ejecución quedan en la BD `t-pool-lo`, `t-pool-hi`, `t-pool-excluded`, `t-pool-nokey`, `t-retry-roundtrip` y `t-retry-clamp`. La siguiente ejecución falla con `duplicate key llm_profiles_name_key` en 3 tests.
- *Archivos afectados (5):* `config_test.go`, `config_max_tokens_test.go`, `llmretry_test.go`, `side_questions_test.go`, `company_scope_test.go`.
- La BD de desarrollo acumula además filas de otros tests de servidor (`lock-order-first-*`, `task-chain-*`, …): 42 perfiles LLM, 670 tareas, 90 empresas y 1.650 findings.
- *Prioridad:* P1. Hace que la suite no sea determinista.

**F3. `lab/benchmark/release_smoke.py` falla con Python 3.10.**
- *Qué pasa:* `parse_qs("", strict_parsing=True)` lanza `ValueError` en 3.10. Con un DSN sin query string, la validación rechaza una base de laboratorio válida.
- *Prioridad:* P2.

**F4. `TestWebFetchProxyFallback`.**
- Hoy no se reproduce (21/21). Se deja una guardia: ejecutarlo con `-count=20` en la CI de estabilización.

**F5. 403 intermitente (histórico).**
- `docs/engineering/IMPLEMENTATION_20261026.md` documenta que ya no se reprodujo en 20 repeticiones.
- Hoy no apareció ningún 403 inesperado en Go, en la batería E2E ni en la latencia.
- Sigue abierto como "no reproducido".

**F6. Biome sin control en la CI.**
- 93 errores en archivos commiteados: `noFloatingPromises` (20), `noLabelWithoutControl` (21), `noEmptyBlockStatements` (23), imports desordenados (32), a11y de clics sin teclado (10).
- *Prioridad:* P2. Varios son de accesibilidad (WCAG).

## 5. Estado funcional por módulo

| Módulo | Estado actual | Problema | Prioridad | Cambio implementado | Pruebas | Resultado | Pendiente |
|---|---|---|---|---|---|---|---|
| Portal multitenant (cuentas, roles, invitaciones, 2FA, SSO, sesiones, auditoría, políticas, términos) | Funcional, con aislamiento por `org_id` en el servidor | Sin pentest externo; SSO con `NEXUS_SSO_DEV_INSECURE` en dev | P1 | Fase 2 (commits previos) | 13 suites E2E | 591/591 | Pentest externo; revisión OIDC; facturación |
| Consola y motor (`orchestrator/server`) | Funcional | Cobertura del 40,8 %; `GET /api/tasks` tarda 128 ms en p50 con 670 tareas | P2 | — | Unit + E2E | Aprobado | Paginación o índice; subir cobertura |
| Persistencia (`db`) | Funcional | F1 y F2; esquema completo en cada arranque | **P1** | — | Unit | 1 fallido (deadlock) | Migraciones versionadas; arreglar limpieza |
| RED (`agents/red`, `scan`) | Funcional con adaptadores httpx/nuclei/katana | Cobertura 50 % / 46 %; falta demostrar el respeto al alcance de extremo a extremo | **P1** | — | Unit (httpx omitido) | Aprobado | Prueba de alcance negativa en el Cyber Range |
| BLUE / Sigma | Subconjunto de Sigma (78 %) | `N of` con N>1 ≈ OR; modificadores desconocidos ignorados; no filtra por `logsource` | P2 | — | Unit (reglas reales omitidas) | Aprobado | Conformidad Sigma contra el corpus oficial |
| Swarm | Scheduler local, DAG, límites | No es distribuido | P3 | — | Unit (88,8 %) | Aprobado | — |
| Evolve / RSI | Optimiza estrategias registradas | No demuestra mejora general | P3 | — | Unit (85 % / 60 %) | Aprobado | Benchmarks independientes (Fase 2) |
| `adversarial_rt` | Parcial | `executeAttempt` es un stub; `applyPatch` solo marca campos | P2 | — | Unit (42 %) | Aprobado | Implementar o rotular "investigación" |
| `ctf_engine` | Parcial | `Start` solo espera cancelación | P3 | — | Unit (59,6 %) | Aprobado | Fase 2 |
| Cyber Range / benchmarks / evaluador | **Ausente** | — | P1 (Fase 2) | — | — | — | Diseño y construcción |
| Laboratorio V8 | **Ausente** | — | P3 (Fase 3) | — | — | — | Builds autorizados y aislamiento |
| Agentes de inteligencia | Parcial (predicción 82,6 %, telemetría 77 %) | Sin fuentes reales conectadas | P3 (Fase 4) | — | Unit | Aprobado | — |
| Facturación y API pública | **Ausente** | — | P1 (Fase 5) | — | — | — | Interfaz de proveedor, Culqi/MP sandbox, SUNAT |
| `phantom` (anonimato y evasión) | Código presente, fuera del binario | Riesgo reputacional y legal para un proveedor B2B/gobierno; 8 paquetes con 0 % de cobertura | **P1 (decisión)** | — | 4 tests | Aprobado | Decidir: excluir del producto o restringir a investigación documentada |
| `governance/report` | Presente | 0 % de cobertura | P2 | — | Ninguna | — | Tests |
| Web | Compila, `tsc` limpio | Biome con 153 errores y fuera de la CI; Playwright no ejecutado | P2 | — | tsc | 0 errores | Biome en la CI; a11y |
| Operación (HTTPS, backups, watchdog, cabeceras) | Implementado (`aa65720`, `3af4188`, `d218b3b`) | Sin backups off-site, sin HA ni monitor externo | P1 | Commits citados | Unit + E2E | Aprobado | Off-site cifrado; runbooks |
| Cadena de suministro | Escaneada | 85 avisos npm (todos en el árbol de shadcn/CLI) y 1 Go no alcanzable | P1 | — | osv-scanner, govulncheck | Ver auditoría | Mover shadcn a devDependencies; actualizar |
| Licencia | AGPL-3.0 | Obliga a ofrecer el código fuente a los usuarios del SaaS | **P0 (decisión legal)** | — | — | — | Decisión del titular y su abogado |

## 6. Rendimiento base

Medición secuencial a unas 20 req/s (por debajo del límite de 50 req/s por IP):
150 peticiones por endpoint, loopback, backend de desarrollo en :8790 y BD de
desarrollo con datos de test acumulados. **No es una prueba de carga.**

| Endpoint | p50 | p95 | p99 |
|---|---:|---:|---:|
| `GET /api/health` | 2,2 ms | 3,0 ms | 4,8 ms |
| `GET /api/assets` | 6,1 ms | 7,7 ms | 9,9 ms |
| `GET /api/companies` | 39,6 ms | 61,7 ms | 75,7 ms |
| `GET /api/tasks` (670 tareas) | **128,1 ms** | 180,0 ms | 191,7 ms |
| `GET /api/admin/ops/status` | 2,2 ms | 2,8 ms | 3,7 ms |

## 7. Cobertura Go

Ver [evidence/phase0/coverage.txt](evidence/phase0/coverage.txt).

- **Con 0 %:**
  - `cmd/nexus`, `cmd/nexus-evolve`, `cmd/nexus-scan`
  - `governance/report`, `i18n`, `lab/enrich`, `lab/benchmark/recallfixture`
  - 7 paquetes de `phantom`
- **Entre 29 % y 50 %:** `cmd/nexus-detect`, `ethical_boundary/{guard,intercept}`, `selfimprove/update`, `ans`, `config/app`, `orchestrator/server`, `adversarial_rt`, `agents/red/scan`, `mcphttp`, `abm`.
- `ethical_boundary` (33,8 % / 34,0 %) es un control de seguridad con cobertura baja: es prioritario para la Fase 1.
