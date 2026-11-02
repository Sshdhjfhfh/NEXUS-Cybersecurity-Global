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

- [ ] **P1** F2: los tests de `db` limpian después de `Close()`. Registrar el cierre con `t.Cleanup` para que corra al final (5 archivos). Tests repetibles con `-count=3`.
- [ ] **P1** F1: el esquema no debe tomar locks exclusivos en cada arranque. Saltar `schema.sql` si su hash ya está aplicado (tabla `schema_meta`) y reintentar ante deadlock. Probarlo con escrituras concurrentes.
- [ ] **P2** F3: `release_smoke.py` compatible con Python 3.10 (query vacía).
- [ ] **P1** R4: subir la cobertura de `ethical_boundary/{guard,intercept}` con casos negativos.
- [ ] **P2** R8: mover `shadcn` a devDependencies, revisar `zod` y actualizar next/geist/postcss. Volver a ejecutar osv-scanner.
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
