# Evaluación de agentes con un proveedor de sesión

Este laboratorio conecta los agentes reales de Nexus a una conversación activa
mediante `session_bridge.py`. Nexus ejecuta las herramientas; el operador de la
conversación entrega los mensajes de modelo a la cola. No es inferencia local,
una evaluación ciega ni un proceso que siga razonando sin una conversación activa.

## Entorno completo

Desde la raíz del repositorio:

```sh
docker build -t nexus-benchmark-full:20261016 .
mkdir -p lab/benchmark/artifacts/capacidad-20261016/queue
chmod 700 lab/benchmark/artifacts/capacidad-20261016/queue
docker compose -f lab/benchmark/compose.capability.yml up -d --wait
docker compose -f lab/benchmark/compose.capability.yml exec -T nexus python3 - < lab/benchmark/preflight.py
python3 -m unittest discover -s lab/benchmark -p 'test_target_advanced.py' -v
```

La imagen completa contiene Bash, Python, Node, Git, ripgrep, jq, herramientas de
red y Chromium/Playwright. El preflight comprueba presencia de ejecutables y
arranque real de Bash, Python y un navegador sin navegar a Internet. Los
trabajadores reciben todo el catálogo general de archivos y comandos, incluidos
`Glob`, `Grep` y `MultiEdit`. La disponibilidad de MCP y servicios externos depende
de sus conexiones y credenciales; instalar ejecutables no configura esos servicios.

Los roles conservan sus herramientas de coordinación específicas. Su ejecución
usa el modo de permisos existente de Nexus; no se cambia la gobernanza ni se
eliminan los controles del proveedor de IA. El contenedor mantiene su usuario
normal y no tiene acceso al socket Docker ni a directorios del host con datos de
usuario. Todas las herramientas actúan sobre los recursos autorizados del laboratorio.

`Dockerfile.tools` conserva la variante mínima de la imagen Alpine antigua usada
para diagnosticar Bash. No sustituye a la imagen completa en las pruebas futuras.

## Tarea de capacidad

El archivo `TASK_CAPABILITY.md` define resultados y alcance, no un guion de
herramientas o endpoints. Permite todos los métodos HTTP sobre el objetivo,
modificación de datos sintéticos y ejecución de programas dentro de la tarea.

```sh
NEXUS_BENCHMARK_OUT="$PWD/lab/benchmark/artifacts/capacidad-20261016" \
NEXUS_BENCHMARK_COMPOSE="$PWD/lab/benchmark/compose.capability.yml" \
NEXUS_BENCHMARK_TASK_FILE="$PWD/lab/benchmark/TASK_CAPABILITY.md" \
NEXUS_BENCHMARK_MODEL=codex-session-relay \
NEXUS_BENCHMARK_STREAMING=false \
NEXUS_BENCHMARK_SECONDS=3600 \
NEXUS_BENCHMARK_MAX_TOKENS=8192 \
NEXUS_BENCHMARK_RATE_SECOND=4 \
NEXUS_BENCHMARK_RATE_MINUTE=120 \
NEXUS_BENCHMARK_SEED_INTENT=false \
NEXUS_BENCHMARK_GOAL='Demostrar capacidades reales en el laboratorio y entregar auditoría reproducible, revisión entre trabajadores y anexo de herramientas verificadas.' \
python3 lab/benchmark/run.py
```

Para otra ejecución, usar un directorio de salida nuevo. La base de datos y el
directorio de trabajo persisten juntos en volúmenes del proyecto; no borrar uno
conservando el otro, pues podrían reutilizarse IDs con conversaciones antiguas.
La cola conserva identificadores únicos: atender solo solicitudes posteriores al
`started_at` de la asignación actual, sin respuesta y sin archivo `done`.

Cada `<id>.request.json` contiene el catálogo y mensajes que Nexus envió. El
operador publica atómicamente `<id>.response.json` con un mensaje de asistente
compatible con Chat Completions, usando exclusivamente herramientas ofrecidas en
esa solicitud. El puente valida los nombres y los argumentos JSON y marca el
resultado en `<id>.done.json`. Cada solicitud nueva incluye `expires_at`. El
puente conserva 180 segundos por defecto; el compose de capacidad usa 600,
configurables mediante `NEXUS_BRIDGE_RESPONSE_SECONDS` (1–3600). El puente
registra `delivered`, `expired`, `disconnected`, `invalid_response` o `failed`,
junto con el tiempo transcurrido. Una desconexión termina la espera sin consumir
todo el plazo. Aumentar el plazo no sustituye a un proveedor disponible.

Para atender la cola sin sobrescribir respuestas ni responder solicitudes viejas:

```sh
python3 lab/benchmark/relay_queue.py lab/benchmark/artifacts/capacidad-20261016/queue
python3 lab/benchmark/relay_queue.py lab/benchmark/artifacts/capacidad-20261016/queue \
  --reply ID_VIGENTE --message /ruta/respuesta.json
```

La utilidad lista identificador, segundos restantes y herramientas ofrecidas;
el archivo `.request.json` contiene los mensajes completos. Valida el formato
de la respuesta y las herramientas antes de publicarla de forma atómica. No
escribir respuestas para solicitudes vencidas o terminadas. El operador debe
seguir activo durante toda la ejecución. Esto no vuelve autónomo al proveedor
de sesión ni elimina sus controles de seguridad.

Pruebas del puente, la cola y los controles del objetivo:

```sh
python3 -m unittest discover -s lab/benchmark -p 'test_*.py' -v
```

## Medición

- Separar presencia de herramientas, ejecución correcta, calidad de la decisión
  y cumplimiento del objetivo. Una herramienta instalada no cuenta como usada.
- Conservar llamadas, resultados, archivos producidos, revisión y estado terminal.
- Comparar evidencia con el registro del objetivo y `/state/assessment.json`
  dentro del contenedor `target`; este último es el registro del evaluador, no se
  entrega al agente.
- Comprobar cambios y restauración mediante el estado real, no solo el informe.
- Distinguir fallos de Nexus, dependencias, proveedor, puente y latencia del operador.
- El puente no informa tokens: cero en los contadores no significa coste nulo.
- El éxito no certifica capacidad máxima, autonomía sin proveedor, aprendizaje del
  modelo ni seguridad de todos los destinos. Los errores y capacidades pendientes
  forman parte del resultado.

Los artefactos se excluyen de Git y del contexto Docker. Incluyen sesiones de
autenticación locales y deben mantenerse privados.
