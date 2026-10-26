# Equipo de mejora de Nexus

El trabajo conjunto usa dos revisores: Codex implementa y comprueba cambios;
Claude Code revisa de forma independiente el problema, el cambio y sus pruebas.
Las observaciones del segundo revisor se contrastan con el código y con casos
reproducibles antes de aceptarlas. Este documento define el procedimiento;
no instala un servicio permanente ni programa ejecuciones en segundo plano.

## Revisión conjunta del 18 de octubre de 2026

Claude Code recibió los cuatro archivos de la corrección y entregó una revisión
independiente de lectura. Consideró coherentes ambas correcciones y explicó cómo
las pruebas cubren los fallos originales. Las pruebas las ejecutó Codex; Claude
no afirmó haberlas ejecutado. La revisión general con herramientas agotó su
plazo de 180 segundos; la revisión acotada, con el código proporcionado
directamente y sin herramientas, sí terminó.

La revisión cubre recuperación de memoria y su integración, no constituye una
auditoría completa de todos los módulos de Nexus.

| Observación de Claude | Contraste con el código | Decisión |
|---|---|---|
| El informe puede pertenecer al corpus anterior durante la espera de reevaluación | Confirmado: se reinician `active` y `evaluated`, pero `last` conserva el informe anterior | Pendiente definir estado de evaluación vigente frente a informe histórico; no cambiar la semántica de `Report()` en esta corrección |
| El límite de título debería tener una constante propia | Confirmado: 256 bytes de título es distinto de 256 documentos | Mejora menor pendiente, sin efecto en la corrección actual |
| Se ignora el error de `filepath.Abs` al elegir el bloqueo | Confirmado; un fallo puede aumentar la contención, no habilita una promoción inválida | Pendiente diagnóstico del fallo; no se ha reproducido en estas pruebas |
| La adaptación solo beneficia a búsquedas por título | Confirmado y ya documentado | Ampliar primero el evaluador antes de ampliar su aplicación |

## Ciclo de trabajo

1. Elegir una deficiencia observable y definir el resultado esperado.
2. Registrar la situación inicial con una prueba o un caso reproducible.
3. Preparar un cambio pequeño, conservando el trabajo previo del repositorio.
4. Pedir a Claude Code una revisión de los archivos relevantes, en modo de
   lectura, sin credenciales ni datos privados. Si no responde, registrar la
   revisión como pendiente; nunca atribuirle conclusiones que no entregó.
5. Verificar sus observaciones, corregir lo necesario y ejecutar las pruebas
   afectadas. No sumar votos de modelos como sustituto de evidencia.
6. Guardar resultados, limitaciones y pendientes. Separar el cambio local de
   su despliegue y de la validación del sistema completo.

## Prioridades para agentes con más iniciativa

| Prioridad | Trabajo propuesto | Evidencia para aceptarlo |
|---|---|---|
| 1 | Diagnosticar herramientas realmente disponibles al iniciar una tarea: ejecutables, herramientas asignadas, MCP conectados y esquemas descubribles | Prueba local que detecte dependencias ausentes y evite repetir llamadas imposibles |
| 2 | Medir el cierre de tareas completas, con criterios observables además de los informes del agente | Caso de laboratorio que termine con objetivo verificado, evidencias y estado final coherentes |
| 3 | Distinguir un fallo transitorio de una dependencia ausente y de una denegación de permisos | Recuperación acotada de fallos transitorios; bloqueo explícito cuando falta una dependencia o permiso |
| 4 | Comparar cambios de planificación e instrucciones con resultados independientes | Casos reservados, coste y duración medidos, rechazo de regresiones y reversión comprobada |
| 5 | Hacer visible qué mejora se aprobó, qué se rechazó y por qué | Informe consultable con versión, conjunto de evaluación y resultados reproducibles |

Usar herramientas con iniciativa significa seleccionar y comprobar las adecuadas
para el objetivo. Las herramientas disponibles dependen del ensamblado y de las
asignaciones del agente (`agents/red/assembly.go`); habilitar todo sin comprobar
su disponibilidad no garantiza un mejor resultado.

El trabajador y el planificador ya admiten `MaxTurns = 0` para turnos sin tope.
Los plazos, la cancelación y el alcance autorizado siguen siendo controles
independientes. Una ejecución más larga necesita detectar progreso y guardar
resultados; los turnos ilimitados por sí solos no demuestran autonomía útil.

## Mejora continua existente

`evolve` ya compara estrategias de recuperación de memoria por título,
evalúa un candidato sobre casos reservados y conserva decisiones con reversión.
Se activa al recuperar memoria, no como entrenamiento permanente. Todavía no
demuestra una mejora general del modelo ni de tareas completas. Véase
[EVOLVE.md](EVOLVE.md).

En esta revisión se reprodujeron y corrigieron dos defectos:

- El método de respaldo podía cambiar el orden de resultados empatados sin
  que se hubiese aprobado una estrategia nueva. Ahora conserva exactamente
  el resultado del método original para la entrada recibida.
- La aplicación de una estrategia incluía títulos que el evaluador excluía
  por longitud o por ausencia de términos. Ahora comparten el criterio de
  elegibilidad y esas consultas conservan el método original.

Las pruebas nuevas están en `evolve/recall/rollout_test.go`. Incluyen una consulta
excluida después de promover una estrategia para otras consultas.

Comprobaciones locales:

```sh
go test -race ./evolve/... ./core/memory -count=1
NEXUS_PG_DSN='' NEXUS_CONFIG=/tmp/nexus-review-no-config.yaml go test ./core/agent ./agents/red -count=1 -timeout=180s
go build ./...
go run ./cmd/nexus-evolve -demo
```

El ejemplo sintético mantiene 11/29 aciertos reservados con el método original
y 29/29 con `phrase-v1`. Pasaron las pruebas de memoria y concurrencia, los
paquetes `core/agent` y `agents/red`, y la compilación completa del proyecto.
Tras añadir el caso de consulta excluida con estrategia promovida, se repitió
`go test -race ./evolve/recall -count=1`, también satisfactorio.

Las pruebas de agentes requirieron permiso para abrir servidores HTTP locales:
el primer intento falló por la restricción de sockets del entorno. Se repitieron
con configuración aislada y sin DSN de PostgreSQL. Las pruebas que necesitan
PostgreSQL se omiten y no cuentan como validación de esa integración. La
compilación se repitió con permiso de escritura de caché de Go.

No se ejecutaron aquí la suite completa de pruebas, un despliegue ni una tarea
con infraestructura de producción. Los cambios no activan herramientas ni
destinos nuevos. No hay evidencia suficiente para prometer ausencia absoluta
de regresiones en todos los módulos.

## Alcance exacto de esta entrega

| Archivo | Cambio de esta revisión |
|---|---|
| `evolve/recall/adaptive.go` | Conservar el ranking original cuando no hay estrategia activa; comentario explicativo |
| `evolve/recall/index.go` | Compartir el criterio de título evaluable y aplicarlo en `Covers` |
| `evolve/recall/benchmark.go` | Utilizar ese mismo criterio al construir los casos |
| `evolve/recall/rollout_test.go` | Tres pruebas nuevas, con dos subcasos de títulos excluidos |
| `docs/EQUIPO_AGENTES.md` | Procedimiento del equipo, resultados, pendientes y alcance |

El repositorio ya tenía numerosos cambios al empezar, incluidos los archivos
de `evolve` sin seguimiento de Git. El diff contra HEAD no representa por sí
solo esta revisión. Se conservaron esos cambios y no se modificaron los módulos
inmutables, permisos, esquemas de base de datos ni configuración de despliegue.
