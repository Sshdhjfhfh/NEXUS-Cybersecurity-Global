# NEXUS IA v6.0

Sistema de investigación adversarial para laboratorios controlados. El backend Go
sirve la interfaz Next.js desde un único contenedor y utiliza PostgreSQL.

## Ejecutar

```sh
docker compose build orchestrator
docker compose up -d
docker compose ps
curl http://localhost:8787/api/health
```

Abrir <http://localhost:8787>. Los servicios `postgres` y `orchestrator` deben
mostrar `healthy`. La cuenta visible se llama **nexus**; el identificador interno
heredado de autenticación se conserva para mantener la compatibilidad.

## Traducciones al español

El catálogo canónico es `i18n/locales/es.json`; su copia pública está en
`web/public/locales/es.json`. El idioma predeterminado es español. Una preferencia
explícita guardada por el usuario tiene prioridad. El catálogo español se carga
también durante el build, evitando claves sin resolver en etiquetas estáticas.
Los demás nueve idiomas conservan sus catálogos existentes; esta entrega no
certifica su traducción completa.

```sh
cd web && npm ci && cd ..
python3 scripts/apply_translations.py --dry-run
python3 scripts/apply_translations.py --extract /tmp/chinese-strings.json
python3 scripts/apply_translations.py
python3 scripts/check_i18n.py
python3 -m unittest discover -s scripts -p 'test_apply_translations.py' -v
```

La migración usa el parser de TypeScript para distinguir texto JSX, atributos,
comentarios, tipos y valores internos. `--dry-run` no modifica ningún archivo;
`--extract` solo escribe el informe indicado. Los respaldos locales de la
traducción no forman parte del repositorio ni del contexto de Docker.

## Desarrollo y restricciones

```sh
go build ./...
go test ./...
```

Los logs internos nuevos deben escribirse en inglés. No modificar los módulos
inmutables: `opsec`, `circuit_breaker`, `watchdog`, `ethical_boundary`,
`llmjacking_guard`, `oracle_defense`, `abm` y `zero_trust`.

La funcionalidad de laboratorio debe permanecer aislada; ninguna implementación
nueva debe permitir acciones fuera de su alcance autorizado. Ver
[la especificación](docs/NEXUS_v6_SPEC.md) y
[la arquitectura](docs/ARCHITECTURE.md).

## Continuación de Nivel 1–2

Las APIs de BLUE, predicción, swarm y aislamiento de laboratorio, junto con los
pendientes de integración, se describen en [el estado de Nivel 1–2](docs/NIVEL12_STATUS.md).
La validación de la interfaz española está en [Nivel 0](docs/NIVEL0_VALIDATION.md).

El [estado real del proyecto](docs/PROJECT_STATUS.md) distingue lo implementado,
lo integrado y lo que falta para completar la especificación v6.0.

La [incorporación de ARTEX](docs/ARTEX_INTEGRATION.md) describe la memoria por tarea,
la compresión recuperable, su adaptación al español y las validaciones pendientes.

La [mejora continua](docs/EVOLVE.md) añade evaluación automática de estrategias de
recuperación de memoria, promoción con casos reservados y reversión ante regresiones.
Puede comprobarse sin un modelo ni red con `go run ./cmd/nexus-evolve -demo`.

El laboratorio defensivo está en `/function/lab`. Sus adaptadores de identidad y
supervisión están en `lab/security`; la composición del host usa
`Server.ConfigureLevel12Security`. El arranque permanece bloqueado hasta conectar
los controles OPSEC/éticos reales y resolver el ciclo de vida de credenciales y
la auditoría pendientes. No existe una opción web para omitir esos controles.
