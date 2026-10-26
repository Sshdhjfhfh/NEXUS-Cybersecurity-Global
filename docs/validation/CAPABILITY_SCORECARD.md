# NEXUS — Scorecard de capacidades reales

**Fecha:** 2026-10-26 · **Rama:** `implementation/nexus-reliability-20261026`
· **Working copy:** `/home/saul/nexus-real` (copia; el original no se toca)
· **Base verificada:** `go build ./...` y `go vet ./...` salida 0; suite previa
1.957 tests aprobados / 7 omitidos / 0 fallos (ver
[docs/engineering/IMPLEMENTATION_20261026.md](../engineering/IMPLEMENTATION_20261026.md)
y [evidencia](../engineering/evidence/validation.json)).

## Escala

| Nivel | Significado |
|---|---|
| 0 | No implementado |
| 1 | Interfaz/estructura sin funcionamiento demostrado |
| 2 | Implementación parcial |
| 3 | Funciona en pruebas unitarias |
| 4 | Funciona en integración/laboratorio |
| 5 | Validado bajo carga, fallos y escenarios representativos |
| 6 | Producción con controles operativos y evidencia suficiente |

Los niveles se asignan por **evidencia ejecutable**, no por nombres de archivo.
No hay puntuación global: cada dimensión se evalúa por separado.

## Resumen por dimensión (directiva §3)

| Dim | Capacidad | Nivel | Evidencia |
|---|---|---|---|
| A | Inteligencia / razonamiento (loop LLM) | **1–2** | `core/agent`, `core/llm` existen; el loop plan→herramienta→verifica **no está demostrado de punta a punta** porque no hay proveedor LLM habilitado. Bloqueante abajo. |
| B | Autonomía | **2** | `core/coordinator`, `core/subagent` con cancelación previa al arranque probada ([cancellation-after.log](../engineering/evidence/cancellation-after.log)); ejecución autónoma completa depende de A. |
| C | Ofensiva autorizada | **4** | `agents/red/scan` + `cmd/nexus-scan`: recon (katana) → web (httpx/nuclei) → **SCA (osv) + secrets (gitleaks)**, todo tras el gate de alcance. Demo real en lab: 5 hallazgos web + 2 secretos + 63 dependencias vulnerables. Pendiente: verificación/PoC de hallazgo. |
| D | Defensiva | **4** | `agents/blue/sigma` + `agents/blue/telemetry` + **`cmd/nexus-detect`** (CLI usable): corre detección pasiva sobre eventos JSONL, autoriza cada activo contra el alcance y emite alertas con tag ATT&CK y agregación temporal. Demo lab: 3 eventos→1 alerta HIGH (T1059.001, 2 evidencias, ventana 30s); activo fuera de alcance denegado; corpus real SigmaHQ carga 1000 reglas. Métricas FP/recall formales: pendientes. |
| E | Purple team | **1–2** | Piezas sueltas (simulación/telemetría) existen; **no hay ciclo cerrado** ATT&CK→simulación→brecha→regla→validación. No construido. |
| F | Mejora continua | **3–4** | `agents/blue/detectbench` sobre `evolve`: promoción verificada y **rechazo de regresión** probados (sign-test, train/holdout). |
| G | Mejora recursiva | **2** | `evolve/experiment.go` (generaciones con presupuesto + evaluador independiente, [generations.json](../engineering/evidence/generations.json)). Mejorar el **propio generador** y la transferencia a tareas nuevas: no demostrado. |
| H | Escalabilidad | **1–2** | Primitivas de concurrencia existen; **sin pruebas de carga** (latencia/colas/CPU/RAM). Sin evidencia. |
| I | Seguridad empresarial | **3** | Gate de alcance fuera del LLM (probado dentro/fuera de scope), auditoría firmada, autz por evento en telemetry, fix de proxy/SSRF en `core/tool`. Aislamiento multi-tenant bajo carga: no probado. |
| J | SaaS internacional | **1** | `web/` existe; multi-tenancy, portal, facturación, i18n: no demostrados. |

## Detalle de las capacidades DEMOSTRADAS (Nivel ≥3)

### C — Ofensiva autorizada (Nivel 4)
- **Archivos:** [agents/red/scan/scan.go](../../agents/red/scan/scan.go),
  [cmd/nexus-scan/main.go](../../cmd/nexus-scan/main.go),
  [lab/engagement/engagement.go](../../lab/engagement/engagement.go).
- **Qué hace:** descubre superficie (katana), prueba endpoints (httpx), detecta
  vulnerabilidades/exposiciones (nuclei), halla secretos filtrados (gitleaks) y
  dependencias vulnerables/CVE (osv-scanner). Cada acción se autoriza contra el
  alcance firmado **antes** de ejecutar el proceso; un objetivo/ruta fuera de
  alcance aborta.
- **Evidencia (demo lab):** recon descubrió 4 URLs → 5 hallazgos web
  (.env, .git/config); SCA halló 63 dependencias vulnerables reales
  (Django 2.2.0, urllib3, requests…) tras colapsar alias (190→63); gitleaks
  halló 2 secretos (redactados en el informe). Prueba de alcance: `/etc`
  rechazado.
- **Tests:** `agents/red/scan` y `lab/engagement` (parsers, redacción de
  secretos, dedupe por grupo, autorización de host y de ruta) — verdes.
- **Limitación / próximo:** no hay verificación controlada ni PoC del hallazgo;
  no hay priorización por explotabilidad real.

### D — Defensiva (Nivel 3–4)
- **Archivos:** [agents/blue/sigma/compile.go](../../agents/blue/sigma/compile.go),
  [agents/blue/telemetry/engine.go](../../agents/blue/telemetry/engine.go).
- **Qué hace:** compila un subconjunto acotado de Sigma (reutilizable y
  concurrente) y aplica detección **pasiva** sobre eventos normalizados, con
  autorización por evento y agregación temporal de alertas. Nunca abre
  conexiones ni aplica contención. **`cmd/nexus-detect`** lo expone como CLI:
  carga reglas de un directorio, lee eventos JSONL y emite alertas con ATT&CK.
- **Evidencia:** parseo de 3.161 reglas reales; fuzz y `-race` verdes
  ([final-sigma-fuzz.log](../engineering/evidence/final-sigma-fuzz.log),
  [sigma-race.log](../engineering/evidence/sigma-race.log)). Demo `nexus-detect`:
  3 eventos→1 alerta HIGH (T1059.001, 2 evidencias, ventana 30s); activo fuera
  de alcance denegado; SigmaHQ real carga 1000 reglas (tope), 2021 omitidas.
  Tests de los helpers del CLI verdes (`cmd/nexus-detect`).
- **Limitación / próximo:** faltan métricas formales de precisión/recall/FP
  contra un corpus etiquetado independiente.

### F — Mejora continua (Nivel 3–4)
- **Archivos:** [agents/blue/detectbench/detectbench.go](../../agents/blue/detectbench/detectbench.go),
  [evolve/engine.go](../../evolve/engine.go).
- **Qué hace:** evalúa variantes de reglas en train/holdout, promueve solo si la
  mejora es estadísticamente sólida (sign-test) y **rechaza regresiones**.
- **Evidencia:** test demuestra promoción verificada y rechazo de ruleset
  "noisy" — verde.

## Capacidades NO demostradas (honestidad, directiva §21-B)
- Razonamiento autónomo de punta a punta (bloqueado por falta de LLM).
- Purple-team como ciclo cerrado.
- Escalabilidad bajo carga; aislamiento multi-tenant bajo carga.
- Plataforma SaaS (portal, multi-tenancy, facturación, i18n).
- Mejora del propio generador de estrategias y transferencia a tareas nuevas.

## Bloqueante principal
**A/B dependen de habilitar un proveedor LLM.** Sin la clave, el "cerebro" que
planifica el pentest y adapta la defensa queda inerte; hoy lo conduce un
operador vía `nexus-scan`. Esto no se puede resolver desde el código: requiere
una decisión y una credencial del dueño del proyecto.
