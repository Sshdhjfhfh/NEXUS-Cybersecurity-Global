# Mejora continua: primera integración operativa

Esta entrega incorpora un motor de evaluación y adaptación en `evolve/` y lo
conecta con la memoria de los agentes de Nexus. Su primera capacidad es mejorar
la recuperación de notas por título. No reentrena los pesos del modelo ni genera
y ejecuta modificaciones de código.

## Cómo funciona

1. Al recuperar memoria, el agente detecta si cambió el corpus de su tarea o chat.
2. Construye casos de búsqueda cuyo resultado esperado es la nota con ese título.
   Es una etiqueta de recuperación comprobable; no certifica que el contenido de
   la nota sea verdadero.
3. Separa notas por un hash estable en selección y comprobación reservada. Una
   nota ocupa un solo grupo; los títulos ambiguos se excluyen. No se multiplican
   las muestras generando paráfrasis de la misma nota.
4. Compara la búsqueda original con dos estrategias registradas en código:
   coincidencia de palabras completas y preferencia por el título completo.
5. Solo el candidato ganador en selección llega a los casos reservados. Debe
   mejorar al menos 10 puntos porcentuales, superar la comparación pareada y
   respetar el presupuesto de contenido recuperado. Si no pasa, se conserva la
   estrategia activa; no se prueba otro candidato contra esos mismos casos
   reservados para buscar uno que apruebe.
6. La promoción se guarda antes de utilizarse. Si cambia el corpus, se vuelve
   temporalmente al método base y se evalúa de nuevo. Una regresión del candidato
   activo frente al método base provoca una reversión registrada.

Las promociones se aplican a consultas por un título único, incluidas diferencias
de mayúsculas y tildes. Las consultas generales conservan la búsqueda original:
la prueba de títulos no demuestra que una estrategia sea mejor para cualquier
pregunta en lenguaje natural.

## Integración y controles

Chat, agente principal, planificador y trabajador utilizan `scopedMemory`.
La adaptación se aplica tanto a `RecallMemory` como a la inyección automática
de recuerdos cuando la consulta pertenece a la familia evaluada.

- La memoria sigue separada por tarea y conversación. No existe intercambio
  global automático de contenido ni se activa `knowledge_bus`.
- Las decisiones se guardan en `<directorio de tarea o chat>/evolve/recall/state.json`.
  El archivo usa permisos `0600` y publicación mediante renombrado atómico.
- El historial conserva hasta 32 informes, acotados también por tamaño. Incluye
  huellas del corpus, evaluador, política y evidencias, resultados y motivos.
  No duplica el texto de las notas ni guarda consultas del usuario.
- Un estado restaurado se vuelve a evaluar antes de aplicar su estrategia.
  Los nombres fuera del catálogo, archivos corruptos, archivos públicos y enlaces
  simbólicos finales se rechazan. Los informes no son una firma criptográfica ni
  una protección frente a un administrador que pueda modificar el programa.
- No se habilitan herramientas, destinos ni permisos nuevos. El motor no tiene
  un comando que permita al agente declararse aprobado o cambiar el evaluador.
- Las evaluaciones respetan la cancelación de la ejecución del agente. Si faltan
  datos o falla el almacenamiento, sigue disponible el método base.

`NEXUS_EVOLVE_ENABLED=false` o `0` desactiva esta adaptación, conservando la memoria.
`NEXUS_MEMORY_ENABLED=false` o `0` desactiva toda la memoria. Sin estas opciones,
la adaptación está conectada a las nuevas ejecuciones del backend construido con
este código; esta entrega no reconstruye ni reemplaza el contenedor en ejecución.

## Límites de trabajo

| Límite | Valor inicial |
|---|---:|
| Notas para indexación adaptativa | 256 |
| Contenido de notas para indexación adaptativa | 2 MiB |
| Notas evaluadas por ciclo | 96 |
| Mínimo por partición | 8 |
| Candidatos de recuperación | 2 |
| Presupuesto temporal del evaluador | 2 segundos, cancelación cooperativa |
| Espera entre evaluaciones por instancia | 1 minuto |
| Contenido recuperado frente al control | Como máximo 1,25 veces |

Si se supera el tamaño del corpus, se utiliza la búsqueda original. La espera se
aplica por instancia del recuperador; una ejecución nueva puede revalidar el estado
persistido. La evaluación se dispara al recuperar memoria, no mediante un proceso
que entrene continuamente mientras Nexus está inactivo.

El coste medido es el tamaño en bytes de la nota recuperada, no los tokens facturados
ni el tiempo de inferencia. El motor usa una prueba pareada unilateral con umbral
0,05. Su interpretación estadística exige independencia entre grupos; separar por
nota no demuestra independencia semántica. Tampoco controla por sí solo la tasa de
falsos positivos de una secuencia ilimitada de experimentos.

La integración serializa los escritores de una tarea dentro del proceso. No se
debe ejecutar otro proceso con el mismo directorio de estado a la vez; para
comparaciones desde la terminal, usar un directorio de estado separado.

## Ejecutar una comprobación reproducible

Desde la raíz del repositorio:

```sh
go run ./cmd/nexus-evolve -demo
go run ./cmd/nexus-evolve -demo -state-dir /tmp/nexus-evolve-demo -json
go run ./cmd/nexus-evolve -memory-dir /ruta/a/memory -state-dir /tmp/nexus-evolve-comparacion
```

El demo usa 96 notas sintéticas en español. Incluye colisiones como `Red 001` y
`Redactor 001`: el buscador original confunde subcadenas; la variante comprueba
palabras y títulos completos. No requiere un modelo, PostgreSQL ni conexión de red.

Resultado comprobado en esta entrega: **11 de 29 aciertos reservados con el método
original y 29 de 29 con `phrase-v1`**. Esto demuestra la corrección del caso de
búsqueda por títulos de ese corpus sintético; no demuestra superioridad general
del agente, del modelo ni del sistema frente a ARTEX.

## Pruebas y siguiente extensión

```sh
go test -race ./evolve/... ./core/memory
go test ./core/agent
go test ./agents/red -run '^(TestAgentMemoryAutomaticallyAdaptsWithinTask|TestTaskMemorySharesWithinScopeAndSurvivesReopen)$' -count=1
go build ./...
```

Las pruebas verifican promoción, rechazo por sobreajuste, casos críticos, presupuesto,
reversión, cambios de evaluador y corpus, revalidación tras reinicio, cancelación,
fallos de persistencia, concurrencia, separación de tareas y el interruptor de
desactivación. La recuperación de consultas no evaluadas conserva el método original.

La siguiente extensión necesita evaluadores con resultados independientes para
tareas completas: exactitud del informe, trazabilidad de evidencias, errores y
coste real. Después podrán compararse estrategias de planificación o instrucciones
con el mismo contrato `Benchmark`. No se debe interpretar el éxito en búsqueda de
memoria como autorización para promover cambios en esos otros componentes.
