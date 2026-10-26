# Revisión de límites de agentes y herramientas — 2026-10-19

Revisión estática del código disponible. No se consultó la configuración activa
de PostgreSQL ni se ejecutaron agentes contra objetivos externos. Los valores
descritos son valores del código, no una certificación del despliegue. Esta revisión
no es exhaustiva y no modifica permisos, presupuestos ni controles de seguridad.

## Límites técnicos identificados

| Componente | Comportamiento observado | Efecto sobre una evaluación |
|---|---|---|
| Turnos del agente | `core/agent/agentcore.go` recibe `MaxTurns`; `db/config.go` almacena el valor por agente. | El presupuesto configurado puede terminar una ejecución antes de completar la tarea. Revisar el valor efectivo y la razón de terminación. |
| Duración | `MaxDuration`, el presupuesto del worker y el plazo de la tarea son mecanismos distintos (`agents/red/worker.go`, `agents/red/taskclock.go`). | Un cambio de presupuesto individual no elimina el plazo de toda la tarea. |
| Bash | `core/tool/bash.go`: espera predeterminada de 120 s, máximo de 600 s. Con gestor de tareas, los comandos elegibles pueden continuar en segundo plano; sin él, el timeout cancela el comando. | No interpretar siempre un timeout de espera como pérdida de capacidad o fin del proceso. |
| Salida de herramientas | `core/tool/helpers.go`: umbral predeterminado de 30.000 bytes, aunque el campo se llama `MaxOutputChars`. Con directorio de salida y escritura exitosa, conserva la salida completa en archivo; en otro caso trunca. | Verificar que los resultados grandes se preserven y puedan recuperarse. Aumentar contexto sin límite no sustituye esa persistencia. |
| Read | `core/tool/files.go`: 2.000 líneas por defecto, paginación y recorte de cada línea a 2.000 bytes. | La paginación permite continuar el archivo, pero no recupera por sí misma la parte recortada de una línea larga. |
| Shell interactiva | `core/tool/shell_session.go`: historial en memoria de 256 KiB, lectura predeterminada de 200 líneas/8.192 bytes y valores de espera e inactividad. | Distinguir historial retenido, tamaño de lectura y duración de sesión al evaluar pérdida de resultados. |
| WebFetch | `core/tool/webfetch.go`: HTTP/HTTPS, petición GET con plazo de 30 s y rechazo de extensiones de recursos estáticos. | Es una herramienta de consulta con un contrato limitado; no equivale a un cliente HTTP general. |
| Modelo | `db/config.go` contiene ventana de contexto, tokens y tasas configuradas por perfil. | La capacidad también depende del proveedor y del modelo. No se verificaron sus límites externos. |

## Disponibilidad de herramientas

- `agents/red/assembly.go` compone herramientas base, skills y MCP visibles.
- `agents/red/toolcatalog.go` define un resolvedor que puede filtrar herramientas
  deshabilitadas o no asignadas al agente. La lista efectiva depende del host y
  de la configuración almacenada; no se verificó esa configuración en esta revisión.
- Hay interruptores globales para tareas en segundo plano y shell interactiva,
  además de opciones por sesión/agente. Su presencia no demuestra que estén
  desactivadas en el despliegue.
- `agents/red/wrapup.go` puede ocultar herramientas durante el cierre de una
  ejecución para guardar resultados. Ese estado no representa la disponibilidad
  de herramientas durante toda la tarea.

## Controles e integraciones

`lab/network/scope.go` limita el alcance de laboratorio a redes privadas explícitas.
`lab/security/security.go` valida identidad, permisos, monitor y controles del host.
`core/tool/bash.go` contiene un bloqueo de ciertos comandos destructivos. Estos
mecanismos cumplen funciones distintas de los presupuestos de tiempo y salida.

El estado del proyecto en `docs/PROJECT_STATUS.md` también declara integraciones
pendientes. Eliminar un bloqueo no implementa esas integraciones ni demuestra
que las capacidades anunciadas funcionen.

## Evaluación recomendada

En un entorno autorizado, registrar las herramientas realmente disponibles,
presupuestos efectivos, motivo de terminación, resultados persistidos, errores de
integración y consumo de recursos. Comparar tareas reproducibles variando un
presupuesto por vez. Esto permite atribuir resultados a límites técnicos,
integraciones o desempeño del modelo sin confundirlos con falta de permisos.
