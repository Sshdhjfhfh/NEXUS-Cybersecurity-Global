# Incorporación de ARTEX a Nexus — 2026-10-15

La referencia revisada es el clon local `~/artex-reference/ARTEX`, commit
`a9d11cb852eadad00669e4f6a05ee59cbbd14860`, con norma v0.4.3. Nexus utiliza
los paquetes absorbidos bajo su propio módulo Go; el clon es una referencia,
no una dependencia de ejecución. Se conservan los avisos de procedencia y licencia.

## Incorporación actual

| Pieza | Base de ARTEX/norma | Adaptación de Nexus |
|---|---|---|
| Contexto de conversaciones largas | NOA resume rangos y archiva los originales; permite recuperar bloques de varios niveles. | `core/noa` y `core/noaadapter`, conectados a chat, agente principal, planificador y trabajador. Recibe el límite de contexto configurado del agente. |
| Activación de NOA | La integración anuncia una vuelta a la compresión estándar si falla la activación, pero la creación de sesión no comprobaba que se pudiera escribir el archivo. | Antes de activar, crea el directorio privado y prueba escritura y sincronización. Si falla, conserva las opciones de compresión estándar y emite un diagnóstico. |
| Memoria duradera | El SDK contiene memoria Markdown, pero el clon no la conecta mediante `MemoryOptions` a los agentes de ARTEX. | Los agentes de una tarea comparten `tasks/<id>/memory`; cada chat usa `sessions/<id>/memory`. Recuperación de términos españoles con o sin tildes, escrituras atómicas y coordinación de escritores dentro del proceso. |
| Español | Plantillas de agentes principalmente en chino, instrucciones NOA en inglés y encabezados de archivos en chino. | Pauta común de respuesta en español, memoria y herramientas de memoria en español, instrucciones permanentes y descripciones de `Compress` en español, panel y encabezados de los archivos traducidos. Se preservan claves, identificadores, citas y evidencias. |
| Skills empaquetadas | El clon incluye skills y herramientas de navegador. | El árbol de trabajo contiene paquetes en `resources/skills`, un instalador que conserva las copias del usuario y cambios de runtime Docker; sus pruebas locales pasan. La imagen no se reconstruyó en esta continuación. |

La pauta de idioma se añade al renderizar los prompts, también si hay una plantilla
guardada antes de esta migración. No reescribe las plantillas del usuario y admite
que este solicite otro idioma. Es una instrucción al modelo, no una garantía de que
todas sus respuestas respeten el idioma.

La compresión avanzada sigue siendo una opción experimental del servidor. El
interruptor se consulta para cada ejecución; no se activa automáticamente al
compilar. `NEXUS_MEMORY_ENABLED=false` o `0` permite desactivar la memoria.

## Comprobaciones

Las pruebas cubren escritura y reapertura de memoria, separación entre tareas,
concurrencia de escritores, activación y recuperación de NOA, almacenamiento
inutilizable, conservación de opciones al fallar, recuperación del original a
través de varios niveles y compatibilidad con registros de compresión antiguos.
La lectura del historial comprimido funciona con archivos de solo lectura y no
crea directorios cuando no hay estado; la prueba de escritura se limita a la
activación de nuevas compresiones.
Las pruebas de prompts verifican las plantillas predeterminadas, las personalizadas
y la recuperación ante errores de plantilla.

```sh
go build ./...
go test ./core/memory ./core/noa ./core/noaadapter
go test ./agents/red -run '^(TestAdvancedCompactionToggleAndFallback|TestTaskMemorySharesWithinScopeAndSurvivesReopen|TestRenderSystemOverrideAndFallback|Test.*Prompt.*|Test.*System.*)$' -count=1
go test -race ./core/memory ./core/noaadapter -run 'TestConcurrent|TestArchiverConcurrentWrites|TestEnableRejectsUnavailableStorageWithoutChangingOptions|TestCompressEndToEnd|TestResumeFromDiskReproducesTheSameView|TestSpanishRecallAndReopen|TestMaterializeCanReadWithoutWritePermission'
python3 -m unittest discover -s scripts -p 'test_seed_skills.py' -v
```

Estas comprobaciones no ejecutan la suite completa que depende de PostgreSQL ni
demuestran el comportamiento de un modelo real. Tampoco equivalen a un despliegue.

## Siguiente trabajo y límites

1. Traducir y revisar las plantillas editables heredadas y los avisos operativos
   restantes. Las reglas detalladas de compresión y varios errores internos aún
   conservan el inglés original; no se declara traducción completa de NOA.
2. Reconstruir la imagen y comprobar permisos de los volúmenes existentes, arranque,
   skills y navegador. Los cambios previos de Docker tienen su propia validación
   pendiente.
3. Medir con el mismo modelo y la misma tarea de laboratorio el consumo de contexto,
   la recuperación de hechos, la evidencia aportada y el idioma del informe. Hay
   un escenario sintético en `lab/benchmark`; esta continuación no lo ha ejecutado.
4. Elegir las siguientes incorporaciones a partir de fallos medidos. El servidor,
   herramientas, captura y grafos ya tienen código heredado: antes de copiar más,
   comprobar qué está conectado y qué mejora realmente el resultado.

La memoria actual es Markdown con búsqueda por términos; no es ChromaDB, una base
vectorial ni aprendizaje automático. La coordinación de escritores es local al
proceso. La comprobación inicial de archivos no garantiza que el disco siga
disponible durante toda la sesión; los errores posteriores se gestionan en la
operación de compresión. No se afirma una mejora cuantitativa frente a ARTEX sin
mediciones comparables.

Los módulos declarados inmutables siguen intactos y los bloqueos pendientes del
laboratorio continúan descritos en [Nivel 1–2](NIVEL12_STATUS.md).

## Revisión de memoria — 2026-10-16

Se reprodujeron y corrigieron dos lecturas que podían mezclar datos entre tareas:
un enlace simbólico en el directorio final de memoria y un enlace en `MEMORY.md`.
Las lecturas verifican ahora el directorio igual que las escrituras. Si el índice
no es un archivo regular, se reconstruye en memoria desde las notas locales sin
modificar archivos. Consultar un almacén inexistente no crea directorios.

Las pruebas de regresión comprueban lectura, búsqueda, contexto del agente y
conservación del almacén al que apunta el enlace. Este cambio rechaza enlaces
finales existentes; no establece aislamiento frente a reemplazos concurrentes
de rutas ni frente a enlaces en directorios antecesores.

La revisión local comprobó `go build ./...`, memoria, evolve, `core/noa` y
`core/noaadapter` con `-race`, las pruebas seleccionadas de integración de agentes,
prompts y activación de NOA, y el instalador de skills. La suite completa de
`core/noaadapter` con detección de carreras terminó en unos 348 segundos.
El demo de evolve reprodujo 11/29 aciertos de control y 29/29 del candidato para
búsquedas por título. Estos resultados no sustituyen la evaluación con un modelo
real ni el despliegue Docker, que siguen pendientes.
