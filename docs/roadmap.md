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
  - **También las herramientas de archivos:** `resolvePath` acepta rutas absolutas sin filtro, así que Read puede abrir `/app/keys/jwt.key` o los datos de otra tarea sin usar Bash.
  - **Diseño y plan por pasos:** `docs/security/tool-runner.md`. El paso 2 (confinar las herramientas de archivos al directorio de la tarea) no depende del contenedor y va primero.
  - **Hecho (código), 2026-11-07:**
    - Paso 1 (commit 9aaafa4): indirección `Launcher` en `core/tool`; Bash fg/bg y sesiones de shell pasan por `launch()`.
    - Paso 3 (commits f281bce, f078063): paquete `core/runner` — servidor en socket Unix con protocolo enmarcado, modo PTY, autenticación `SO_PEERCRED`, confinamiento de directorio (rechaza fugas por `../` y symlink), muerte del grupo al desconectar; `MountIsolate` con namespaces PID+montaje+usuario (`/proc` fresco → un comando no ve `/proc/<pid>/environ` del orquestador, rutas sensibles enmascaradas, `no_new_privs`); subcomandos `runner-init`/`runner-exec`/`runner-serve` y `SetLauncher` cableados en `cmd/nexus` (con build tags). Tests `-race`; verificado end-to-end con el binario real (shell = PID 1 en su namespace, `/proc` del orquestador oculto, clave enmascarada ilegible, escape de directorio rechazado).
  - **Pendiente:** despliegue del contenedor `tool-runner` + red `nexus_tools` + pasarela de salida (paso 5) con la **prueba negativa real** (sin `/app/keys`, sin ruta a Postgres) — **bloqueado por la integración Docker–WSL**; paso 2 (confinar Read/Write/Edit/Grep/Glob a la raíz de la tarea); paso 4 (customtool/MCP stdio por el runner); paso 6 (rechazar `Local` cuando `NEXUS_PORTAL_BASE_URL` es `https://`).
  - **Bloquea los encargos contra clientes reales** (la frontera fuerte es el contenedor, aún pendiente).
- [x] **P2** Higiene de tests: 38 `defer X.Close()` en 17 archivos pasan a `t.Cleanup`, registrado antes de los borrados de filas. El residuo por ejecución de `db`, `agents/red` y `orchestrator/server` baja a 1 tarea.
- [x] **P2** `TestFindingTrafficArchiveV3RoundTripAndRetry` y `TestNotifyDeepLinkUsesPublicBaseURL` fallaban a veces. **Causa:** la suite local corría contra la BD `nexus` del backend de desarrollo, cuyos workers reclamaban los trabajos de los tests: la cola de archivado es global y el notificador reparte los eventos.
  - No era un fallo del código: la CI ya usa una BD propia (`nexus_test`) y `docs/engineering/IMPLEMENTATION_20261026.md` pide una base dedicada.
  - **Arreglo:** la suite local también usa `nexus_test` (misma instancia, otra base). Con ella, el par pasó 10/10 y la suite completa dos veces seguidas (2060 tests, 0 fallos). La BD de desarrollo no cambia: 1117 tareas antes y después, así que ya no aparecen tareas de prueba en el panel.
- [x] **P2** R8: `shadcn` 4.23.0 pasa a devDependencies, `zod` sube a 4.6.7, `next` a 16.4.4, más `npm audit fix`.
  - osv-scanner: de 20 paquetes y 84 avisos a **2 paquetes y 2 avisos medios, ambos de desarrollo** (`deepmerge` y `minimatch`, vía shadcn).
  - `npm audit --omit=dev`: 0.
  - `tsc` 0 errores; `build:static` correcto (68 páginas).
- [x] **P2** SBOM CycloneDX en la CI (`security.yml`, job `sbom`).
  - **Go:** `cyclonedx-gomod` v1.12.0 genera el SBOM de los módulos de `cmd/nexus` y el del binario compilado.
  - **Web:** `@cyclonedx/cyclonedx-npm` 6.0.1 genera el SBOM del árbol de producción.
  - Se suben como artefacto `nexus-sbom`, y la CI falla si alguno está vacío.
  - Probado en local: 50 módulos Go en el binario y 363 paquetes web de producción (CycloneDX 1.6).
- [x] **P2** F6: errores de Biome y Biome en la CI.
  - **Medición:** con `biome check` sobre el árbol commiteado y el mismo método antes y después, se pasa de **253 a 219 errores**.
  - **Archivos sin cambios locales del usuario:** quedan **0 errores de lint**. Se corrigieron promesas sin manejar, bloques vacíos y `forEach` que devolvían valor. Además, el `Checkbox` de Radix se declara como control de etiqueta, porque renderiza un `<button role="checkbox">`, que es etiquetable.
  - **Lo que queda (219):**
    - 151 errores en 11 archivos con cambios locales sin commitear del usuario, que no se tocaron.
    - 68 avisos previos de formato y orden de imports en 41 archivos. No se reformatearon para no mezclar un diff grande sin cambio funcional.
  - **CI:** el job `web` de `ci.yml` ejecuta `tsc` y `scripts/biome_ratchet.py`, un trinquete por archivo contra `web/biome-baseline.json`. Falla si un archivo supera su cuenta o si un archivo nuevo tiene errores; la base solo puede bajar.
  - **Al commitear los cambios locales del usuario:** el trinquete puede marcar esos 11 archivos. Hay que corregirlos o, si se acepta la deuda, regenerar la base con `--update`.
- [x] **P2** `GET /api/tasks`: con ~1100 tareas ya medía **310 ms en p50 y 505 ms en p95** (en vivo).
  - **Causa (perfil de CPU):** el estado de cada tarea iniciada llamaba a `engine.ReadyFor`, que lee de la BD las vinculaciones LLM de `planner` y `worker`. Eran ~2 consultas por tarea (N+1). Ordenar las comprobaciones no servía, porque los bucles del motor marcan como iniciadas casi todas las tareas.
  - **Arreglo:** `listReadiness` aplica la misma regla que el resolvedor del motor, pero lee cada vinculación una sola vez por petición. Lo usan `GET /api/tasks`, la herramienta `list_tasks` y los grupos de hallazgos. No se añadió una caché global, porque una caché caducada podría enviar datos de un cliente al proveedor LLM equivocado.
  - **Resultado en proceso** (1116 tareas, misma BD): de 301,8 ms a **14,3 ms** por petición.
  - **Resultado en vivo** (backend de desarrollo recompilado, ~1117 tareas, 100 peticiones, todas 200): p50 de 310 a **33,3 ms**, p95 de 505 a **64,7 ms** y p99 de 120,5 ms. La batería E2E pasó entera con ese binario (597/597).
  - **Test:** `TestListReadinessMatchesEngineAndReadsBindingsOnce` comprueba que el estado coincide con `engine.ReadyFor` en todas las tareas y que cada vinculación se lee una vez.
- [x] **P2** Guardia en la CI: `TestWebFetchProxyFallback -count=20` (job `go` de `ci.yml`). En local: 20/20 en 0,66 s.
- [x] **P2** La CI (`ci.yml`) ahora también corre en las ramas `implementation/**`. Antes solo corría en `main` y `fase2-saneamiento`, así que la rama de trabajo no pasaba por ella.
- [x] **P2** Tests para `governance/report`: la cobertura pasa de 0 % a **97,4 %**. Es el código de las exportaciones a Markdown, CSV y ZIP (un archivo por hallazgo) que se entregan al cliente. Los tests encontraron tres fallos, ya corregidos:
  - **Evidencia que escapaba de su bloque de código.** Si la evidencia o la PoC contenía ```` ``` ````, el objetivo (que puede ser hostil) escribía texto propio en el informe del cliente, por ejemplo "Sin hallazgos". Ahora la valla es más larga que cualquier secuencia de backticks del contenido (`codeBlock`).
  - **Inyección de fórmulas en el CSV.** Las celdas que empiezan con `=`, `+`, `-`, `@`, tabulador o CR se ejecutaban como fórmula al abrir el CSV en Excel. Ahora llevan un apóstrofo delante (`csvText`, recomendación de OWASP).
  - **Nombres inválidos en el ZIP.** El nombre de archivo se cortaba a 120 bytes en mitad de un carácter UTF-8 (títulos en chino o con tildes). Ahora se corta en un límite de carácter.
  - **Sigue abierto:** el campo `Report` (informe detallado en Markdown escrito por el agente) se inserta tal cual, por diseño. Si algún visor lo convierte a HTML, debe sanearlo.
- [x] **P2** Las exportaciones del operador (`governance/report`: página de hallazgos en Markdown, ZIP y CSV, y `/api/report`) salían **en chino**, heredado de ARTEX. Los informes PDF que recibe el cliente en el portal (`portal_report_pdf.go`) ya estaban en español e inglés.
  - Ahora se generan en español por defecto, en inglés o en chino. Elige el parámetro `lang`, que la web envía con el idioma del panel; sin él se usa `Accept-Language`.
  - Los estados y las severidades también se traducen (por ejemplo "Riesgo aceptado" o "[CRÍTICA]").
  - Tests: `TestParseLang`, `TestCatalogComplete` (ningún texto vacío en ningún idioma), `TestExportLang` y `TestFindingsExportFollowsLang`.

## Privacidad y protección de datos (servidor grande en producción)

Análisis completo: [security/privacy-readiness.md](security/privacy-readiness.md).
NEXUS procesa datos muy sensibles de clientes (hallazgos, evidencia, cuentas),
así que esto bloquea cobrar a clientes reales. Ya existe: cifrado de secretos
2FA/SSO, export+borrado GDPR, auditoría encadenada, TLS+HSTS, scrub de secretos
en comandos de agente, aislamiento multi-tenant. Falta:

- [ ] **P0** Controlar y minimizar el **egress a LLM**: bandera `ai_local_only` por
  organización, redacción de PII/secretos antes del prompt, procedencia de qué se
  envió y a quién, y acuerdo de subencargo con cada proveedor LLM.
- [ ] **P0** **Cifrado en reposo** de la base de datos y del almacén de evidencia
  (cifrado de volumen/disco o TDE; clave fuera del volumen).
- [ ] **P0** Cerrar **R13** (ver Fase 1): sin el contenedor, un agente alcanza la BD
  y archivos del host — también es fuga de privacidad entre clientes.
- [ ] **P1** **DPA** (acuerdo de tratamiento de datos) + política de privacidad
  real (no borrador) + lista de subencargados.
- [ ] **P1** **Retención y eliminación automatizadas** por encargo (no solo el
  export/erasure manual que ya existe).
- [ ] **P1** Gestión de claves con **KMS/gestor de secretos** y rotación (hoy todo
  deriva de un único `jwtKey` en un archivo).
- [ ] **P1** **Residencia de datos**: decidir y documentar dónde vive el servidor.
- [ ] **P2** Redacción de PII en hallazgos/evidencia antes de mostrar/exportar/LLM;
  emergency stop del cliente; política de divulgación publicada.

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
- [~] **P2** Detección y bloqueo **automático** de mal uso por un miembro ya autenticado. **Hecho en modo sombra (commit 08119dd):** `portal_abuse.go` puntúa señales por usuario en ventana deslizante de 15 min (pesos: `authz.forbidden` 2, `scope.blocked` 3, `auth.ip_blocked` 1, `export.excessive` 3; umbral 6); al cruzarlo, con `NEXUS_ABUSE_AUTOSUSPEND` activo desactiva al miembro + revoca sus sesiones + audita + alerta a `NEXUS_OPS_EMAIL`; por defecto (sombra) solo audita `security.abuse_detected` y alerta, sin suspender, para afinar umbrales. Nunca suspende a un owner; nunca por fallos de login (evita DoS a la víctima). Cableado en `portalRequireRole` y `enforceOrgPolicy`. Núcleo `abuseRecord` puro y testeado. **Pendiente:** hook de `scope.blocked` desde el motor (hoy solo señales a nivel portal) y `export.excessive`; panel para el operador; afinar pesos con tráfico real antes de activar la suspensión.
- [ ] **P2** HA (2 instancias más la BD gestionada) y `docs/operations/runbooks.md`.
- [ ] `docs/security/threat-model.md` (STRIDE por componente).

## Fase 8: preparación comercial

- [ ] `docs/release/readiness.md` con criterios GO/NO-GO medibles.
- [ ] `docs/architecture/target-state.md`.
