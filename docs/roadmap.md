# NEXUS: hoja de ruta y lista de trabajo persistente

Fuente de verdad del trabajo pendiente. Se actualiza al cerrar cada tarea con el
commit y la evidencia. Solo se marca `[x]` lo que está implementado **y**
verificado. Prioridades: P0 bloquea la venta, P1 bloquea la producción,
P2 importante, P3 mejora.

Estado al 2026-11-02: base `d218b3b`. Diagnóstico en
[architecture/current-state.md](architecture/current-state.md) y riesgos en
[security/security-audit.md](security/security-audit.md).

## Decisiones que corresponden al titular (no técnicas)

- [ ] **P0** Licencia AGPL-3.0 en el SaaS: publicar el código, licencia comercial upstream o reescritura (con abogado).
- [ ] **P1** Módulo `phantom` (anonimato y evasión): ¿excluirlo del repositorio del producto o mantenerlo como investigación restringida y documentada?
- [ ] **P1** Datos legales de la empresa, dominio, SMTP real (SPF/DKIM/DMARC) y buzón de operaciones (`NEXUS_OPS_EMAIL`).
- [ ] **P1** Obligaciones SUNAT (facturación electrónica) antes de cobrar.
- [ ] **P2** Borrar las reglas portproxy de Windows en :8787 y :8789 (PowerShell como administrador).

## Fase 0: diagnóstico ✅ (2026-11-02)

- [x] Inventario, mapa de dependencias, línea base de tests, cobertura, OSV, licencias y latencia.

## Fase 1: estabilización

- [x] **P1** F2: los tests de `db` borraban sus filas después de `Close()`. Ahora el cierre va en `t.Cleanup` registrado primero (`config_test.go`, `llmretry_test.go`; los otros 3 archivos ya lo hacían). Verificado: `go test ./db` 3 veces seguidas en verde y 0 filas `t-%` residuales.
- [x] **P1** F1: `db.Open` salta `schema.sql` si su SHA-256 ya consta en `nexus_schema_state`. Un `schema.sql` nuevo se aplica una vez, bajo el lock de migración; `NEXUS_SCHEMA_FORCE=1` fuerza la reaplicación. Test: `TestOpenSkipsUnchangedSchema`.
  - Suite Go completa (`-p 4`) 2 veces seguidas: 1.654 aprobados, 0 fallidos, 6 omitidos. E2E 591/591.
  - `go test ./db` en A/B dentro de la misma ventana: 35 s antes, 28–29 s después.
- [x] **P2** F3: `release_smoke.py` compatible con Python 3.10. `lab/benchmark` 16/16.
- [x] **P1** R4: cobertura de `ethical_boundary` con casos negativos.
  - `guard` sube de 33,8 % a 71,0 % e `intercept` de 34,0 % a 62,3 %; el nuevo paquete `scope` tiene 96,9 %.
  - Los tests de `intercept` crean reglas etiquetadas y las borran. Verificado: 0 residuales.
- [x] **P0** R12: el alcance de un encargo de cliente no se aplicaba a las herramientas y los agentes podían ampliarlo.
  - **Bloqueo al promover:** al promover una solicitud del portal, la tarea queda con el alcance bloqueado (`task_scope_locks`). Desde ese momento:
    - `add_task_scope` del agente devuelve `ErrScopeLocked`;
    - el auto-alcance por activos insertados no hace nada;
    - solo el operador puede ampliarlo (filas `manual`).
  - **Endpoints:** `PUT` y `DELETE /api/tasks/{id}/scope/lock`. Desbloquear exige un motivo de al menos 10 caracteres y queda en el log.
  - **Puerta de alcance en `PreToolUse`:** cubre Bash, shell, WebFetch y MCP en el worker, el planner y el agente principal. El planner y el agente principal antes corrían **sin hooks**. Si la consulta del alcance falla, la llamada se bloquea.
  - **Tests:** `TestTaskScopeLock`, `TestScopeGuardBlocksOutOfScope`, `TestScopeGuardUnlockedAndFailClosed` y `scope_test.go`.
  - **Verificación del conjunto de la Fase 1, parte 2:**
    - Suite Go completa (`go test ./...`), 2 ejecuciones. La 1.ª: 2.042 aprobados, 0 fallidos, 8 omitidos. La 2.ª: 2.040 aprobados y 2 fallidos (los dos tests de cola compartida de abajo). Ahora se cuentan también los subtests, por eso no es comparable con los 1.654 anteriores.
    - E2E: 597/597 (13 suites). Incluye 6 comprobaciones nuevas en `admin_e2e`: alcance bloqueado al promover, el token del portal no puede desbloquearlo, desbloqueo sin motivo → 400, y desbloqueo y rebloqueo del operador.
  - **Límite honesto:** la puerta inspecciona el texto de la llamada. No ve hosts construidos dentro de scripts, codificados o sin punto. Es defensa en profundidad, no una frontera de red (ver R13 abajo).
- [x] **P1** R13 (parcial): los comandos que lanzan los agentes ya no heredan los secretos del orquestador.
  - **Aplica a:** Bash en primer y segundo plano, shell interactivo, herramientas personalizadas y servidores MCP stdio.
  - **Se eliminan:** `NEXUS_*`, `ARTEX_*`, `PG*`, `POSTGRES*`, `DATABASE_*`, `SMTP*`, `AWS_*`, `AZURE_*` y cualquier variable que contenga `SECRET`, `PASSWORD`, `TOKEN`, `CREDENTIAL` o `PRIVATE`, o que termine en `_KEY`, `_PASS` o `_DSN`.
  - **Se conservan** las entradas explícitas: proxy, CA y el `env` declarado de cada MCP.
  - **Tests:** `TestBashDoesNotInheritSecrets`, `TestSensitiveEnvKey` y `TestChildEnvKeepsExplicitSecrets`.
- [ ] **P0** R13: ejecutar las herramientas de los agentes en un contenedor separado (tool-runner).
  - **Requisitos:** sin red hacia la BD, sin secretos montados, sin acceso a `/proc` del orquestador y con la salida limitada a una allowlist derivada del alcance del encargo.
  - **Hoy:** Bash corre dentro del contenedor del orquestador como usuario `node`. Puede leer `/proc/<ppid>/environ` y `/app/keys` y alcanzar la BD.
  - **Bloquea los encargos contra clientes reales.**
- [x] **P2** Higiene de tests: 38 `defer X.Close()` en 17 archivos pasan a `t.Cleanup`, registrado antes de los borrados de filas. El residuo por ejecución de `db`, `agents/red` y `orchestrator/server` baja a 1 tarea.
- [ ] **P2** Tests que comparten colas globales con el backend de desarrollo: `TestFindingTrafficArchiveV3RoundTripAndRetry` y `TestNotifyDeepLinkUsesPublicBaseURL` fallaron 1 vez en 2 ejecuciones de la suite completa. Aislados pasan 5/5. Causa probable: otro proceso de la misma BD reclama el trabajo o el evento. Arreglo: filtrar por los IDs del test o usar una BD por ejecución en la CI.
- [x] **P2** R8: `shadcn` 4.23.0 pasa a devDependencies, `zod` sube a 4.6.7, `next` a 16.4.4, más `npm audit fix`.
  - osv-scanner: de 20 paquetes y 84 avisos a **2 paquetes y 2 avisos medios, ambos de desarrollo** (`deepmerge` y `minimatch`, vía shadcn).
  - `npm audit --omit=dev`: 0.
  - `tsc` 0 errores; `build:static` correcto (68 páginas).
- [ ] **P2** SBOM CycloneDX generado en la CI.
- [ ] **P2** F6: corregir los errores de Biome en archivos commiteados (sin tocar los cambios locales del usuario) y añadir Biome a la CI.
- [ ] **P2** `GET /api/tasks`: 128 ms en p50 con 670 tareas. Perfilar y paginar o indexar.
- [ ] **P2** Guardia en la CI: `TestWebFetchProxyFallback -count=20`.
- [ ] **P2** Tests para `governance/report` (0 %).

## Fase 2: Cyber Range, benchmarks y evaluador independiente

- [ ] **P1** Cyber Range local aislado (red interna sin salida, objetivos vulnerables conocidos, reinicio reproducible).
- [ ] **P1** Prueba negativa de alcance: el motor RED no debe tocar hosts fuera de alcance (tráfico capturado como evidencia).
- [ ] **P2** Evaluador independiente del motor (oráculo con flags y hallazgos esperados, sin acceso al estado interno).
- [ ] **P2** `docs/benchmarks/methodology.md` y `results.md`, solo con resultados medidos.
- [ ] **P3** `adversarial_rt` y `ctf_engine`: implementar o rotular como investigación.

## Fase 3: laboratorio V8

- [ ] **P3** Solo builds autorizados en un entorno aislado, sin cadenas de explotación ni escapes de sandbox. Divulgación coordinada. `docs/research/v8-lab.md`.

## Fase 4: agentes de inteligencia

- [ ] **P3** Conectar fuentes reales (KEV, EPSS, NVD) con procedencia y fecha; nada de datos inventados.

## Fase 5: SaaS, facturación y API

- [ ] **P1** Interfaz `PaymentProvider` con Culqi y Mercado Pago, solo en sandbox hasta tener autorización.
- [ ] **P1** Planes y suscripciones por organización, límites de uso y webhooks firmados e idempotentes.
- [ ] **P2** API pública con claves por organización (con hash y scopes) y webhooks salientes. `docs/api/openapi.yaml`.
- [ ] **P2** Modo "IA solo local" por organización, emergency stop del cliente y política de retención.
- [ ] `docs/billing/architecture.md`.

## Fase 6: diseño y UX

- [ ] **P2** Sistema de diseño documentado (`docs/design/design-system.md`) y auditoría WCAG 2.1 AA.

## Fase 7: resiliencia y seguridad

- [ ] **P1** Backups cifrados off-site, ensayo de restauración y RPO/RTO definidos.
- [ ] **P1** Pentest externo independiente.
- [ ] **P2** HA (2 instancias más la BD gestionada) y `docs/operations/runbooks.md`.
- [ ] `docs/security/threat-model.md` (STRIDE por componente).

## Fase 8: preparación comercial

- [ ] `docs/release/readiness.md` con criterios GO/NO-GO medibles.
- [ ] `docs/architecture/target-state.md`.
