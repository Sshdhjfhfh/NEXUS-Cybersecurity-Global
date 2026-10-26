# Preparación de un asesor local para Nexus

Este módulo inicia la preparación y medición de un modelo local. **No entrena
pesos ni convierte el modelo en un especialista**. Exporta catorce lecciones
sintéticas para revisión y evalúa ocho ejercicios de análisis de código y
evidencias. No ejecuta salidas del modelo ni modifica los agentes de Nexus.

El objetivo posterior es un asesor que pueda planificar auditorías autorizadas,
interpretar resultados de herramientas, distinguir hipótesis de hallazgos y
proponer correcciones comprobables. Esta primera entrega mide solo clasificación,
selección de evidencia y categoría de corrección; no mide ejecución ofensiva,
planificación completa, capacidades máximas ni calidad semántica de explicaciones.

## Uso

Requiere Python 3, sin paquetes adicionales. Ollama debe estar escuchando en
`127.0.0.1:11434` y el modelo ya instalado. No descarga modelos ni usa proveedor
de respaldo. Ejecutar desde la raíz, con un directorio nuevo para cada ensayo:

```sh
python3 -m unittest discover -s lab/academy -p 'test_*.py' -v
python3 lab/academy/run.py export --out lab/academy/artifacts/lecciones-001
python3 lab/academy/run.py evaluate --model qwen3:4b --out lab/academy/artifacts/base-001
python3 lab/academy/run.py evaluate --model qwen3:4b --profile defined-labels-v1 --out lab/academy/artifacts/instrucciones-001
```

`train.jsonl` contiene conversaciones para una futura fase de ajuste supervisado;
`provenance.json` identifica origen, autor y estado de revisión. Exportar no
entrena nada. No usar catorce ejemplos como prueba de cobertura del dominio.

Las respuestas esperadas de evaluación no se envían al alumno ni al revisor de
lecciones. Las familias de escenarios de entrenamiento y evaluación son distintas,
pero comparten conceptos. Los casos de evaluación están publicados en el código:
son un conjunto pequeño de desarrollo, **no un examen secreto ni estadísticamente
suficiente para promover un modelo**. Tras ajustar instrucciones mirando estos
resultados, hace falta un conjunto independiente para medir generalización.

Cada ejecución registra huella del conjunto y del prompt, versión de Ollama,
configuración y digest del modelo, parámetros, respuestas, tokens informados por
Ollama, duración y errores. Los errores de transporte cuentan como fallos en el
total y también se identifican por separado. No confundirlos con errores de
conocimiento. Un timeout del cliente no demuestra cancelación de la generación en
el servidor. El contexto solicitado es 4096; la metadata no prueba por sí sola
el contexto efectivo durante inferencia.

La calificación exige JSON válido, etiquetas correctas y el conjunto exacto de
evidencias seleccionado por el autor del caso. Citar evidencia adicional puede
fallar esta regla aunque una explicación humana la considere útil. Las
explicaciones se conservan para revisión, pero no se califican semánticamente.
Una respuesta contradictoria puede pasar los controles estructurados: no usar
estas puntuaciones como única puerta para autorizar acciones.

`defined-labels-v1` añade definiciones a las instrucciones. Es una variante de
prompt diseñada después de observar errores, no destilación ni pesos nuevos.

## Papel de Codex y Claude

Codex crea los casos y comprueba su coherencia. Claude revisa únicamente las
lecciones y propone material adicional. Sus opiniones se contrastan con evidencia;
el acuerdo de dos modelos no certifica corrección. No incorporar automáticamente
sus respuestas al corpus ni atribuirle material si la llamada falla.

El intento y la respuesta de esta sesión se guardan localmente en `artifacts/`.
Solo se envía material sintético, sin credenciales ni conversaciones de Nexus.
El alumno se consulta exclusivamente en loopback, sin proxies ni redirecciones.

## Mejora continua y permisos

El motor existente `evolve/` compara estrategias registradas de recuperación de
memoria, con casos reservados, promoción y reversión. No es entrenamiento de pesos
ni un sistema que se reescribe mientras está inactivo. Véase
[EVOLVE.md](../../docs/EVOLVE.md).

La inspección del código de esta sesión encontró `ModeBypass` en el agente
principal, planificador y trabajador. Evita las confirmaciones ordinarias de esa
capa; siguen existiendo denegaciones explícitas, interceptores y controles del
entorno. Esto describe el código, no certifica la configuración de un contenedor
en ejecución. No se ha reproducido un bloqueo de permisos dentro de Nexus.

Las confirmaciones de Codex/Claude y los permisos del sistema operativo son capas
independientes. No son una capacidad que el modelo pueda aprender a eliminar.
Un timeout del proveedor tampoco se soluciona quitando permisos.

Para ampliar la mejora continua a tareas completas:

1. Medir tareas sintéticas de principio a fin con evidencia del objetivo,
   herramientas realmente ejecutadas, falsos positivos, duración y coste.
2. Clasificar fallos: modelo, instrucciones ambiguas, dependencia ausente,
   error de infraestructura o denegación real con su regla y acción concretas.
3. Proponer un cambio acotado en instrucciones o código, sin modificar la rúbrica.
4. Compararlo en escenarios nuevos, conservar trazas y rechazar regresiones.
5. Promover una versión verificable con posibilidad de reversión. La evaluación
   debe permanecer fuera del control del agente evaluado.

Las operaciones repetidas de lectura, pruebas y escritura de artefactos dentro
del laboratorio pueden autorizarse de antemano con límites de recursos y alcance.
Esto no requiere eliminar todos los controles ni dar al agente autoridad para
ampliar sus propios permisos. No se ha añadido un daemon ni cambiado una política
de permisos en esta entrega.

## Pendiente antes de entrenar y sustituir el cerebro

Hace falta ampliar y revisar el corpus, preparar un entorno de entrenamiento,
verificar el checkpoint base y su licencia, medir consumo de memoria con una
prueba pequeña y reservar casos nuevos. Después se puede ajustar un candidato,
exportarlo, comparar con la base y conectarlo a una instancia de laboratorio de
Nexus. No activar un candidato en producción por memorizar este conjunto.

En el entorno inspeccionado hay una RTX 3050 de 4 GiB y aproximadamente 10,7 GiB
de RAM total; no están instalados `torch`, `transformers`, `peft`, `trl`,
`datasets` ni `bitsandbytes` en el Python consultado. No se ha validado que un
entrenamiento concreto quepa en ese hardware. Ollama tiene `qwen3:4b` y
`qwen3-16k:4b`, además de modelos menores; la medición usa el primero.

Referencias de implementación:

- [API de chat de Ollama](https://docs.ollama.com/api/chat): mensajes, formato JSON,
  pensamiento y métricas.
- [Importación en Ollama](https://docs.ollama.com/import): importar pesos es una
  fase distinta de preparar datos o entrenar.
- [Autorización de OWASP](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html):
  comprobación de permisos en cada solicitud y denegación por defecto.
