# Continuación de la verificación de agentes — 16 de octubre de 2026

La verificación local pasa en los nueve paquetes seleccionados. La evidencia
guardada de una ejecución posterior a la observación con Qwen demuestra trabajo
HTTP, escritura de informes, revisión entre trabajadores y recuperación de
memoria dentro de una tarea. No demuestra cierre autónomo de la tarea ni una
ejecución nueva del contenedor completo.

## Pruebas ejecutadas en esta continuación

Go 1.26.3, sin caché de resultados (`-count=1`), con límite de 180 segundos
por paquete. Se cuentan pruebas principales; los subtests y semillas de fuzz
no se suman de nuevo.

| Paquete | Aprobadas | Omitidas |
|---|---:|---:|
| `agents/blue` | 2 | 0 |
| `agents/prediction` | 2 | 0 |
| `agents/red` | 55 | 14 |
| `core/agent` | 23 | 0 |
| `core/subagent` | 3 | 0 |
| `core/memory` | 8 | 0 |
| `core/noa` | 235 | 2 |
| `core/noaadapter` | 138 | 0 |
| `swarm` | 4 | 0 |
| **Total** | **470** | **16** |

También pasan los tres controles de `lab/benchmark/test_target_advanced.py`.
Estos validan la lógica del objetivo sintético; no son una auditoría realizada
por los agentes.

La primera ejecución dentro del sandbox bloqueó los sockets de `httptest` en
`agents/red` y `core/agent`. Además, una prueba que requiere PostgreSQL falló al
recibir una DSN deliberadamente inaccesible. Ambos paquetes se repitieron fuera
del sandbox con `NEXUS_PG_DSN=''` y `NEXUS_CONFIG` apuntando a un archivo
inexistente: pasaron, omitiendo las pruebas que necesitan base de datos. No se
conectó esta verificación a la base de datos de producción.

Registros de esta sesión:

- `/tmp/nexus-agent-verification/go-tests.jsonl`: primera ejecución.
- `/tmp/nexus-agent-verification/go-tests-retry.jsonl`: repetición de los dos
  paquetes afectados por el entorno.

No se ejecutó el detector de carreras ni la suite completa del repositorio.
Las pruebas omitidas no cuentan como capacidades validadas.

## Evidencia histórica contrastada

Fuente: `lab/benchmark/artifacts/session-continuacion-20261016/`.
Se verificaron las 97 entradas de `queue-manifest.json` contra sus archivos
mediante SHA-256: ninguna discrepancia. Esto comprueba consistencia con el
manifiesto local; no constituye una certificación externa de procedencia.

Esta ejecución usó `codex-session-relay`: las respuestas del proveedor fueron
entregadas por un operador de sesión, y Nexus ejecutó las herramientas.
Sus resultados deben mantenerse separados de la observación con `qwen3:4b`
documentada en [la prueba anterior](LAB_OBSERVATION_2026-10-16.md).

| Capacidad | Evidencia conservada | Límite |
|---|---|---|
| Configurar alcance | `add_task_scope` añadió `target`; `set_constraints` registró los límites | La extracción inicial automática del host volvió a fallar |
| Consultar el objetivo | Seis GET, todos con respuesta 200, en el registro del objetivo | El registro del objetivo no certifica por sí solo ausencia de tráfico a otros destinos |
| Registrar hallazgos | Dos registros: configuración expuesta y acceso sin autenticación a informes privados sintéticos | Ambos conservan estado `pending`; no son hallazgos aprobados |
| Producir informe | `Write` confirmó `auditoria-continuacion.md`, de 4.406 bytes | No se volvió a leer el volumen Docker en esta sesión |
| Revisar entre trabajadores | `work#3` leyó el informe, consultó nodos e historial y escribió `revision-continuacion.md`, de 1.884 bytes | Revisión asistida por el mismo proveedor de sesión |
| Compartir memoria | `work#2` guardó la ausencia de Bash; la respuesta de `RecallMemory` para `work#3` contiene esa nota completa | No prueba memoria entre tareas ni persistencia tras reinicio |
| Ejecutar terminal | `Bash` y `shell_open` fallaron por ausencia del ejecutable `bash` | El Dockerfile actual incluye Bash, pero no se verificó su imagen construida |
| Cerrar la tarea | Las intenciones 2 y 7 terminaron en `done` | El objetivo seguía `open` y la tarea terminó `paused` |

Las dos últimas solicitudes al proveedor, con identificadores que comienzan
por `f35ecfd2` y `20db2114`, figuran como `expired`. El historial termina en una
ronda de planificación interrumpida por la pausa. Esto identifica un fallo de
atención del proveedor de sesión en esa ventana; no permite atribuir toda la
espera a la lógica de planificación ni demostrar un cierre satisfactorio.

## Verificación pendiente

Docker no está disponible desde esta sesión WSL: el comando devuelve el aviso
de integración de Docker Desktop no habilitada. Tampoco se encontraron
ejecutables locales `psql` o `postgres`. Por ello no se inició una ejecución
nueva ni se ejecutaron las pruebas de integración con PostgreSQL.

Con el entorno disponible, el siguiente paso es construir la imagen completa,
ejecutar `lab/benchmark/preflight.py` dentro de ella y usar un directorio nuevo
de evidencias para la tarea de capacidad. La prueba debe mantener disponible
su proveedor durante toda la ejecución y comprobar el estado terminal, además
de los informes y resultados de herramientas. El procedimiento está en
[el README del laboratorio](../lab/benchmark/README.md).

En esta continuación no se modificó el código de los agentes.
