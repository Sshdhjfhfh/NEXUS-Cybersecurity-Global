# Auditoría técnica de NEXUS

Fecha: 2026-10-26. Base: `21ac386100218b2ab0b8f907e6aa52e9e8906698`.
Repositorio confirmado por el usuario: `/home/saul/nexus-real`; no se modificó
`/home/saul/nexus`. Rama: `research/nexus-evidence-20261026`.

## Alcance y método

Revisión estática transversal y pruebas locales. Se leyeron README, arquitectura,
especificación, estados previos, `go.mod`, Makefile, CI, Dockerfile y composición.
Se inspeccionaron los módulos solicitados y sus contratos. No es una auditoría
exhaustiva de cada línea, una prueba de intrusión ni una certificación empresarial.
Las pruebas históricas citadas por otros documentos no se consideran ejecuciones
de esta sesión. Los resultados actuales están en [VALIDATION_MATRIX.md](VALIDATION_MATRIX.md)
y sus evidencias adjuntas.

Inventario de archivos versionados antes del cambio: 1.029 archivos, 662 Go,
290 archivos de pruebas Go. No se cuentan respaldos locales como producto.
Go 1.26.3; backend Go, UI Next.js/React, PostgreSQL y SQLite para usos específicos.
`go list -m -json all` enumera 183 módulos externos, más que los `require` directos:
incluye el grafo transitivo seleccionado. El [SBOM](evidence/SBOM.cdx.json)
contiene 1.021 componentes Go/npm de los manifiestos examinados.

## Capacidades y límites observados

| Área | Evidencia de implementación | Límite real |
|---|---|---|
| RED | `agents/red/{mainagent,planner,worker}.go`, herramientas, findings y retests; `agents/red/scan/scan.go` y CLI `cmd/nexus-scan` | Adaptadores httpx/nuclei/katana; la prueba real de httpx se omite si falta el binario. No se certificó detección real de CVEs. |
| BLUE | `agents/blue/sentinel.go`: TLS, cabeceras, cookies y fallos de login sobre observaciones aportadas | Reglas deterministas de higiene HTTP; no equivalen a un SOC o detección general de intrusiones. |
| Sigma | `agents/blue/sigma/{sigma,match}.go`, parser y matcher con pruebas | Subconjunto, no conformidad completa. `N of` con N>1 se aproxima a OR, modificadores desconocidos se ignoran y `Match` no selecciona por `logsource`. |
| Predicción | `agents/prediction/engine.go`: Beta/Wilson, deduplicación y snapshots | Umbral de 50 sesiones no garantiza independencia, calibración ni precisión en clientes. |
| Swarm | `swarm/swarm.go`: ocho roles, DAG, autorización por trabajo, límites, cancelación y aislamiento de panic | Memoria del scheduler local; no demuestra recuperación distribuida o ejecución exactamente una vez. |
| Orquestación | `orchestrator/server/engine.go`, `db`: intents, claims, estados CAS, eventos, blackboard y trabajadores | Existen dos modelos de coordinación: runtime heredado y DAG de laboratorio. No hay prueba de equivalencia de controles entre ambos. |
| Identidad/laboratorio | `lab/{gate,audit}.go`, `lab/security`, ANS/NHI/ABM y `orchestrator/level12` | Composición de producción pendiente de OPSEC/ética reales. Cierre por defecto preservado. |
| Memoria/LLM | `core/memory`, `agents/red/memory.go`, NOA; `core/llm`, `llm_gateway/pool` | Notas por tarea/chat y compresión; sin memoria vectorial global demostrada. Failover evita repetir streams ya emitidos; costes adaptativos pendientes. |
| Evidencias/gobierno | `evidence/store.go`, `governance/{report,notify}`, auditoría de autorizaciones | Hashes y persistencia existen; no demuestran cadena de custodia externa, retención legal ni aislamiento completo entre empresas. |
| Evolución | `evolve`, `evolve/recall`: selección, holdout, promoción, límites y reversión | Optimiza estrategias registradas, principalmente búsqueda de títulos. No entrena pesos ni demuestra mejora recursiva general. |
| Detección experimental | `agents/blue/detectbench` conectado a `evolve` | Fixtures train/holdout repiten eventos de la misma familia; sus etiquetas y aciertos no prueban generalización. Digest no incluye todos los eventos/reglas. |
| `adversarial_rt` | Motor y almacén, con pruebas | `executeAttempt` es un stub y `applyPatch` solo marca campos. No prueba ataque bloqueado ni parche aplicado a un componente real. |
| `ctf_engine` | Registro y evaluación de intentos | `Start` solo espera cancelación; comparaciones humanas codificadas no son un benchmark humano medido. |
| `zero_day` | Python y AFL++ sobre parser C sintético, contenedor sin red, reproducción y guía local | Ya no está vacío, a diferencia de estados antiguos. No demuestra descubrimiento de vulnerabilidades inéditas en software real. |
| Academia/RSI | `lab/academy`, `lab/benchmark`, `rsi_detector` | Academia exporta/evalúa material sintético; no entrena. Tendencias RSI son heurísticas, no prueba causal de mejora recursiva. |
| Actualizador | `selfimprove/update`: descarga, SHA256SUMS, staging y bootstrap | Actualización de binario no es autoaprendizaje. Un checksum del mismo origen no autentica por sí solo al editor. |

La búsqueda de imports muestra que Sigma alimenta `detectbench`; no se encontró
su integración con el pipeline de telemetría del servidor. `scan` está usado por
la CLI. Los paquetes `adversarial_rt`, `ctf_engine` y `rsi_detector` no aparecen
conectados por imports en los ejecutables inspeccionados.

No hay archivos Go versionados en `opsec`, `circuit_breaker`, `watchdog`,
`llmjacking_guard`, `oracle_defense`, `graph_rag`, `vckg`, `knowledge_bus`,
`symbolic_ai`, `ai_scientist`, `cti_feed`, `digital_twin`, `xai` ni `federated`.
`ethical_boundary` sí contiene código heredado; su presencia no aporta el
evaluador obligatorio que requiere el laboratorio. No se sustituyeron controles.

## Riesgos priorizados

| ID | Prioridad | Evidencia y consecuencia | Decisión propuesta |
|---|---|---|---|
| A01 | P0 | `ModeBypass` en agentes principal/planificador/worker; restricciones de prompt no son un perímetro de red | Verificar cada ruta de herramientas con autorización y aislamiento externos al LLM, antes de autonomía productiva. No retirar controles existentes. |
| A02 | P0 | Scanners autorizan entradas, pero procesos externos manejan navegación, DNS, redirects y plantillas | Probar confinamiento de cada conexión, fijar plantillas/hashes y egreso por alcance. No asumir que una plantilla de detección es inocua. |
| A03 | P0 comercial | LICENSE y cabeceras ARTEX/norma AGPL; arquitectura habla de independencia pese a código derivado | Inventario de procedencia y revisión del modelo comercial; no relicenciar por cambiar imports. |
| A04 | P1 | OSV identifica coincidencias de versión en 29 componentes | Triage por alcanzabilidad/uso; ver [licencias y dependencias](LICENSE_COMPLIANCE.md). No son 29 explotaciones confirmadas. |
| A05 | P1 | Semántica Sigma aproximada y corpus real no ejecutado | Rechazar construcciones no soportadas o implementar con oráculo independiente antes de desplegar reglas externas. |
| A06 | P1 | Fixtures de detectbench casi triviales y eventos repetidos entre splits | Separación por escenario/tiempo/organización, hashes completos y evaluación ciega externa. |
| A07 | P1 | Falta de pruebas de pérdida de proceso, redistribución e idempotencia distribuida | Outbox, leases y fencing solo tras pruebas de fallo sobre contratos actuales. |
| A08 | P1 | Compose publica API en `0.0.0.0`; imágenes por tag y herramientas amplias en runtime | Revisar diseño de despliegue, autenticación/TLS, egreso y perfiles de ejecución; sin cambiar configuración actual. |
| A09 | P2 | `make lint` depende de `fmt`, que usa `gofmt -w` | Añadir comprobación sin escritura en un cambio separado; no usar ese target en una auditoría de lectura. |
| A10 | P1 | Suite completa presenta un timeout en `TestWebFetchProxyFallback` antes del arreglo Sigma | Reproducir y diagnosticar; no esconderlo ni considerarlo causado por este cambio. |

## Primera mejora realizada

Se eligió propagar errores de recorrido de `LoadDir` por su retorno `[]error`
existente. La ruta ausente y los subdirectorios inaccesibles dejaban un resultado
indistinguible de un árbol vacío. Se añadieron cuatro pruebas y siete líneas netas
de lógica/comentario, sin modificar coincidencias, permisos ni rutas productivas.
Es una mejora de visibilidad, no una garantía de carga completa: el consumidor
debe inspeccionar los errores. Detalles, regresión previa y reversión en la matriz.

## Conclusión

NEXUS tiene una base funcional considerable, pero no hay evidencia suficiente de
SOC autónomo, cobertura ATT&CK validada, ejecución ofensiva confinada de extremo
a extremo ni mejora recursiva. La inversión inicial debe ir a control verificable
de acciones, semántica de detección, evidencias y aislamiento entre organizaciones.
La [hoja de ruta](TECHNICAL_ROADMAP_12_MONTHS.md) responde las quince preguntas y
ordena las fases por dependencias.

Fuentes locales: [README](../../README.md), [arquitectura](../ARCHITECTURE.md),
[estado anterior](../PROJECT_STATUS.md), [límites](../AUDITORIA_LIMITES_2026-10-19.md),
[evolución](../EVOLVE.md). Referencia de proceso:
[NIST SSDF 800-218](https://csrc.nist.gov/pubs/sp/800/218/final).
