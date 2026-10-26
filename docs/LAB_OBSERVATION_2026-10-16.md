# Prueba observada de Nexus — 16 de octubre de 2026

**Resultado: la instancia recibió la tarea, pero no completó la primera fase de
planificación ni inició la auditoría HTTP durante la ventana de observación.**
No se ha determinado la capacidad máxima de Nexus. Se midió esta combinación
concreta de código, entorno y modelo local.

## Configuración y método

- Instancia independiente: proyecto Docker `nexus-observe-20261016`, tarea `1`.
  No se utilizó la base de datos de producción ni la ejecución antigua.
- Binario construido desde el árbol de trabajo actual, incluyendo los cambios
  de memoria y evolve. SHA-256:
  `4de32a2da244ab7748684e7aa5e2d4f2cf7ec97f7d6df1cc470bf069aea6a14c`.
- Imagen base local `nexus-ia/orchestrator:v6.0.0` (`df518ec624b4`), con el binario
  actual y las skills montados en solo lectura. No es una reconstrucción completa
  del Dockerfile nuevo. La imagen comprobada tiene Python; no se encontraron
  `curl`, `bash` ni `rg` en su PATH.
- Modelo local `qwen3:4b`, cuantización `Q4_K_M`. Perfil: 4.096 tokens máximos de
  salida, contexto configurado de 32 Ki tokens y pensamiento configurado como
  desactivado. Ollama informó un contexto efectivo de **4.096 tokens**, con unos
  2,34 GB en VRAM. No se verificó que el perfil cambie el contexto real de Ollama.
- El contenedor Nexus estuvo conectado únicamente a una red Docker interna,
  sin gateway externo. El puente del modelo únicamente admite las rutas de
  modelos y completaciones del Ollama local; no reenvía destinos arbitrarios.
- Objetivo sintético propio: `http://target:8080`. Tarea: descubrir rutas,
  comprobar exposición de configuración, acceso a registros privados y
  cabeceras, y entregar un informe con evidencia y correcciones.
- Alcance: solo GET, máximo 30 peticiones, una por segundo y cinco minutos.
  El evaluador no entregó las respuestas esperadas al agente.
- El observador creó la tarea mediante la API normal de Nexus y después realizó
  lecturas de estado, actividad y registros. No envió mensajes, pistas,
  aprobaciones de herramientas ni correcciones durante la ejecución.

## Evidencia observada

| Medida | Resultado |
|---|---|
| Creación de tarea | 09:59:26, hora de Lima |
| Actividad inicial | Primera ronda de descomposición de objetivos |
| Peticiones recibidas por el objetivo | **0**, registro vacío |
| Herramientas ejecutadas en el historial recogido | **0** |
| Hallazgos registrados | **0** |
| Informe de auditoría | No producido; no existía `/app/data/tasks/1` |
| Estado antes del cierre | `created`, con la intención inicial todavía `open` |
| Cierre del observador | Pausa a las 10:05:03, tras aproximadamente 337 segundos |
| Estado final | `paused`, no `completed` ni `timeout` |

La tarea solicitaba 300 segundos. El observador permitió hasta 330 segundos de
observación para recoger el cierre; las llamadas a la API añadieron latencia.
Nexus no llegó por sí mismo a un estado terminal en esa ventana.

Al pausar, el historial publicó un fragmento de razonamiento del planificador y
un resultado de interrupción durante la salida del modelo. El backend registró
que la descomposición no produjo resultado y creó un objetivo de respaldo. Esa
creación ocurrió al cerrar la prueba, no constituye una auditoría completada.

Los contadores de tokens quedaron en cero, pero **no prueban consumo nulo**:
hubo razonamiento recibido del modelo. No se obtuvo una medición fiable del
consumo total de inferencia interrumpida. El puente registró `BrokenPipeError`
al intentar escribir en una conexión cerrada; ese error por sí solo no identifica
la causa de la espera anterior.

## Límites y próximos experimentos

El arranque y la comunicación API funcionan. En esta prueba no se demostró la
capacidad de pasar de planificación a herramientas, encontrar problemas o
redactar un informe. Tampoco se midieron memoria a largo plazo, mejora continua,
swarm, precisión, recuperación ni una auditoría compleja. La ausencia de acciones
no permite certificar el cumplimiento activo del alcance.

Se observaron dos condiciones que deben investigarse por separado:

1. El backend no extrajo automáticamente `target` como host/IP para crear el
   alcance. Lo registró expresamente; no se añadió alcance durante la prueba.
2. El contexto efectivo informado por Ollama no coincide con el perfil de Nexus.
   Además, se recibió razonamiento pese a la configuración de pensamiento
   desactivado. No se ha establecido causalidad entre estas condiciones y la espera.

La siguiente medición debería preparar explícitamente el alcance antes de
iniciar la tarea y verificar los parámetros efectivos de inferencia. Es una
configuración experimental nueva: sus resultados deberán conservarse separados
de esta observación. No se modificó el código de los agentes para rescatar esta
ejecución ni se resolvió la auditoría en su lugar.

## Archivos de evidencia

Directorio local: `lab/benchmark/artifacts/observacion-20261016/`.
Contiene asignación, estados antes y después del cierre, actividad, intención,
objetivos, hallazgos, registro del objetivo, registros de Nexus y del puente,
metadatos y huella del binario. Los artefactos están excluidos de Git y del
contexto Docker porque también incluyen una sesión de autenticación local.

La ejecución antigua de `lab/benchmark/artifacts/` no se sobrescribió ni se contó
como resultado de esta prueba.
