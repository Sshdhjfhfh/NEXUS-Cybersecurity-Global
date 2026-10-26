# Implementación y validación — 2026-10-26

Rama: `implementation/nexus-reliability-20261026`, sobre
`research/nexus-evidence-20261026`. Base Git:
`21ac386100218b2ab0b8f907e6aa52e9e8906698`. Espacio autorizado:
`/home/saul/nexus-real`. Los cambios siguen sin commit y no se desplegaron.
Se conservaron los informes y la corrección anterior de `Sigma.LoadDir`.

## Resultado comprobado

Se implementaron detección pasiva con autorización por evento, compilación de un
subconjunto explícito de Sigma, experimentos por generaciones sobre estrategias
registradas y cancelación previa al arranque de subagentes. No se modificaron los
módulos declarados inmutables en el README, las políticas de producción ni los
manifiestos de dependencias. No se ejecutaron pruebas ofensivas contra terceros.

La suite final terminó con **1.957 resultados de prueba aprobados, siete omitidos,
57 paquetes aprobados y 14 sin pruebas**; ningún fallo. Los recuentos incluyen los
subtests que informa `go test -json`; no equivalen a escenarios independientes.
Compilación y `go vet ./...`: salida 0. Evidencia estructurada:
[validación](evidence/validation.json), [archivos y hashes](evidence/source-manifest.json),
[parche reversible](evidence/implementation.patch).

## Ciclo 1 — timeout del proxy y diagnóstico del 403

**Código:** `core/tool/builtins_test.go`,
`orchestrator/server/level12_security_test.go`.

El test suponía que `127.0.0.1:1` rechazaría la conexión inmediatamente. En este
entorno la conexión quedó esperando: se reprodujo el fallo a los 30,073 segundos.
La causa de esta reproducción fue la dependencia de un puerto externo al fixture,
no una carrera demostrada del cliente. El nuevo proxy HTTP local acepta la
conexión y la cierra con EOF. Se comprueba una petición al proxy y una al destino.
No se aumentó ningún timeout ni se modificó el fallback productivo.

**Antes/después:** fallo a los 30,073 s → 20 repeticiones con `-race` aprobadas,
aproximadamente 0,03 s por caso. Ver `proxy-before.log` y `proxy-after.log`.

El 403 histórico **no se reprodujo** en 20 repeticiones contra PostgreSQL aislado
ni en las suites completas posteriores. El test conserva ahora la primera
denegación observada por el callback real de auditoría firmada. Antes, la consulta
diagnóstica posterior podía mostrar el estado de otro trabajador y ocultar la
causa. Esto mejora la investigación; no demuestra haber corregido el 403.

**Aceptación:** fallback probado sin depender de un puerto supuestamente cerrado;
controles de autorización intactos. **Pendiente:** capturar la primera denegación
si el 403 reaparece y reducirla a un caso reproducible.

## Ciclo 2 — semántica Sigma y compilación reutilizable

**Código:** `agents/blue/sigma/{sigma.go,match.go,compile.go,condition.go,semantic_test.go}`.
`Rule.Match` conserva su firma; `Rule.Compile` crea predicados inmutables para la
ruta frecuente. `Parse` valida la semántica, el tamaño y un único documento YAML.

Se implementaron conteos reales `N of`, precedencia booleana, patrones de nombres
de selecciones, comodines y escapes en valores, `null`, `exists`, `cased` y flags
de regex. Los patrones regex conservan mayúsculas y escapes. Modificadores,
condiciones y estructuras no soportadas producen un error. El compilador limita
tamaño, selecciones, tokens y profundidad. No implementa toda la especificación:
correlación Sigma, `timeframe`, `near` y expresiones fuera del subconjunto se
rechazan; el motor Go de regex tampoco equivale a PCRE completo.

**Evidencia antes/después:** el evaluador anterior producía tres falsos positivos
en la tabla completa de `2 of 3`; ahora resuelve los ocho casos correctamente.
Ocho reglas inválidas antes aceptadas se rechazan. La revisión final también
rechaza `all` escalar o con lista de un solo valor, como exige la especificación,
y prueba un caso positivo donde deben coincidir todos los valores.
Pruebas de regex, valores
ausentes, concurrencia y precedencia aprobadas. Fuzzing: 146.805 ejecuciones en
seis segundos, sin fallo; otra ejecución al cierre añadió 78.785 ejecuciones
sin fallo tras el ajuste de `all`. Eso no prueba ausencia de defectos.

| Microbenchmark local, mediana de tres ejecuciones | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `Rule.Match` anterior | 1.789 | 580 | 15 |
| `CompiledRule.Match` nuevo | 80,76 | 0 | 0 |
| `Rule.Match` nuevo, validación por llamada | 2.946 | 7.795 | 29 |

La mejora de aproximadamente 22 veces corresponde a **evaluar un predicado ya
compilado**, excluye compilación y solo mide el pequeño fixture indicado. La API
de compatibilidad se vuelve más costosa porque valida en cada llamada. El nuevo
motor de telemetría y `detectbench` usan la ruta compilada. No se extrapola esta
medición al rendimiento de un SIEM o del sistema completo.

**Fuente:** [especificación Sigma](https://github.com/SigmaHQ/sigma-specification/blob/main/specification/sigma-rules-specification.md)
y [modificadores](https://github.com/SigmaHQ/sigma-specification/blob/main/specification/sigma-appendix-modifiers.md).
Implementación original; no se copiaron reglas comunitarias. Se conservan autor,
licencia y referencias aportadas por las reglas cargadas.

**Riesgo:** reglas antes aceptadas por aproximación pueden rechazarse ahora.
Revisar los errores de carga y ampliar el subconjunto con fixtures positivos y
negativos antes de activar otros corpus. El corpus externo opcional no se probó.

## Ciclo 3 — motor Blue Team pasivo con evidencias

**Código:** `agents/blue/telemetry/{engine.go,json.go,engine_test.go}` y ampliación
de `lab/security/integration_test.go`.

API funcional: `Decode`, `New`, `Engine.Analyze`. Normaliza eventos JSON planos,
conserva números grandes, rechaza claves duplicadas, atribución inválida, objetos
anidados, valores no finitos y documentos concatenados. Límites: 256 KiB por
evento, 10.000 eventos y 16 MiB por lote, 1.000 reglas por motor. Esta primera API
se limita a IP privadas literales de laboratorio.

El llamador debe autenticar organización y tarea **independientemente** de los
eventos. Cada evento debe coincidir con esos parámetros y superar el autorizador
existente para `blue.analyze`. No hay autorizador permisivo predeterminado. Ante
una denegación no se entrega un informe parcial. No abre conexiones a objetivos
ni realiza contención.

Compila reglas una vez; enruta por `logsource`; deduplica ID/contenido y rechaza
un mismo ID con contenidos diferentes. Agrupa coincidencias por regla, activo y
ventana fija dentro de una petición. Produce alertas con organización, tarea,
evidencias, fechas, procedencia y digest SHA-256 del lote. Las etiquetas ATT&CK
clasifican reglas: no representan técnicas emuladas o cobertura demostrada.

**Pruebas:** tres eventos positivos, dos negativos y un duplicado → tres
coincidencias, cero falsos positivos en ese fixture y dos grupos temporales.
Se prueban atribución cruzada, duplicados conflictivos, revocación, cancelación y
16 organizaciones concurrentes. No es una estimación de precisión poblacional.
La integración usa ANS/NHI persistentes reales, auditoría firmada, ABM y PostgreSQL
aislado: admite el evento antes de revocar y lo rechaza tras revocación persistida.
OPSEC y ética usan callbacks estrictos del fixture, no una validación productiva
completa. Tres repeticiones con `-race` aprobadas.

**Fuente conceptual:** contratos de evidencias/evaluadores de
[ExCyTIn-Bench](https://arxiv.org/abs/2507.14201v3) y
[ACESEvals](https://github.com/microsoft/ACESEvals).
No se importó código ni datasets de esos proyectos.

**Límite:** biblioteca utilizable, todavía sin endpoint HTTP de ingesta ni conector
SIEM desplegado. Correlación dentro del lote, sin estado persistente entre lotes.
La separación por petición no demuestra aislamiento SaaS global. No incorpora
YARA, análisis de comportamiento aprendido ni remediación automática.

## Ciclo 4 — experimentación por generaciones y evaluación independiente

**Código:** `evolve/{engine.go,experiment.go,experiment_test.go}`,
`agents/blue/detectbench/{detectbench.go,detectbench_test.go}`,
`cmd/nexus-evolve/main.go`.

`RunExperiment` reutiliza el motor existente y fija catálogo, política, evaluador,
presupuesto y duración antes del primer ciclo. Ejecuta hasta 32 generaciones,
8.192 evaluaciones de casos y una hora; el CLI usa un minuto. Rechaza directorios
de estado productivo. La selección activa solo vive en memoria. La fábrica del
evaluador proporciona datos nuevos por generación; hashes de entradas y grupos
impiden reutilizar exactamente casos aunque se cambien sus nombres. Los hashes
no demuestran independencia estadística ni evitan toda contaminación semántica.

Se registran agente, generación, estrategia, dataset, evaluador, contrato, casos,
aciertos, regresiones, holdout, coste en unidades declaradas y tiempo. Los tokens
no medidos quedan `null` y las habilidades adquiridas quedan vacías. La huella
de `detectbench/2` incluye reglas, eventos, etiquetas y casos, no solo sus nombres.

**Ejecución real:** [generaciones](evidence/generations.json). En cada generación
hay 20 casos sintéticos (10 entrenamiento, 10 reservados). Baseline: 8/20;
estrategia seleccionada: 20/20. Mejora en generación 1, sin nueva mejora en 2 y 3;
parada `plateau` en 3; 150 evaluaciones totales, ninguna regresión en esos casos.
Los negativos impiden promover una estrategia que alerta indiscriminadamente.
El experimento con presupuesto 20 termina con error de presupuesto y sin promover
una generación incompleta. También se prueba replay con IDs cambiados y la
prohibición de escribir estado productivo.

Esto es selección autónoma **entre estrategias previamente registradas** sobre
familias sintéticas relacionadas. No genera habilidades, prompts ni código nuevos,
no entrena LLM, no demuestra transferencia general y no implementa mejora recursiva
del generador. El aislamiento es respecto al estado productivo: el evaluador es
código anfitrión de confianza, no una sandbox para plugins arbitrarios. Un callback
que ignore el contexto necesita aislamiento por proceso antes de admitir código
no confiable. El coste registrado no equivale a dinero ni energía consumida.

**Fuente conceptual:** [CyberEvolver](https://arxiv.org/abs/2605.26195v2), evaluación
separada de resultados funcionales y seguridad de
[LSCBench](https://arxiv.org/abs/2610.14766v1). Código original; no se ejecutaron
los benchmarks externos ni se atribuyen sus resultados a NEXUS.

## Ciclo 5 — cancelación de coordinación y subagentes

**Código:** `core/coordinator/{coordinator.go,coordinator_test.go}` y
`core/subagent/subagent.go`.

La adquisición del semáforo respeta cancelación. Se comprueba el contexto antes
de encolar, después de adquirir plaza y antes del ciclo de vida del subagente.
Las tareas no iniciadas conservan identidad/posición con su error. No se reescribe
la arquitectura ni se añade un planificador alternativo.

**Antes/después:** un contexto ya cancelado iniciaba tres subagentes; ahora inicia
cero. Se comprueba también invocación directa por herramienta sin hooks tras
cancelación. Veinte repeticiones con `-race` aprobadas. Esto evita trabajo inútil
previo al arranque; no prueba recuperación de procesos ni interrupción forzada de
proveedores que ignoren el contexto.

## Dependencias, derechos y decisiones

El [triage de 29 componentes](evidence/dependency-triage.json) relaciona versiones,
avisos, versiones corregidas publicadas, exposición estimada y prioridad. Hay
110 identificadores de aviso distintos; algunos son alias. Se recuperaron 109
detalles OSV; una consulta falló por red. El aviso restante de Browserslist se
contrastó en [GitHub Advisory](https://github.com/advisories/GHSA-73wf-gq98-2v4g).
Las versiones corregidas listadas pertenecen a varias ramas y no son una receta
de actualización ni garantizan corregir todos los avisos a la vez.

Los cuatro componentes Go afectados no aparecen en el grafo de imports de los
dos comandos principales revisados; eso no verifica símbolos alcanzables, otras
etiquetas de compilación, herramientas o imágenes instaladas. Next/sharp forman
parte de la construcción estática observada; despliegues alternativos y entradas
maliciosas al build siguen siendo relevantes. Ningún aviso fue suprimido y no se
actualizaron dependencias en este ciclo. El SBOM histórico sigue siendo válido
para los manifiestos sin cambios, no para todo artefacto desplegado.

Se preservaron licencia AGPL y atribuciones ARTEX/norma. Las aportaciones
originales no eliminan las obligaciones del núcleo derivado. AGPL admite comercio
y exige cumplir las condiciones de fuente correspondiente, incluida la sección
13 para versiones modificadas accesibles por red; no permite asumir un backend
derivado cerrado. [Texto de licencia](https://spdx.org/licenses/AGPL-3.0-or-later.html).
Resolver titularidad y permisos antes de elegir distribución o SaaS comercial.

**DefenderBench no se incorpora:** su
[licencia fijada](https://github.com/microsoft/DefenderBench/blob/e431098e217ee6778f572df202785f1a705df167/LICENSE)
limita el uso a investigación no comercial y restringe redistribución/servicio.
La matriz [internacional](INTERNATIONAL_CAPABILITY_EXTRACTION.md) distingue los
conceptos estudiados del código legalmente reutilizable.

## Reproducción y regresión

Condiciones: Linux amd64, Go 1.26.3, Intel i5-12500H, dependencias locales,
PostgreSQL 16 en contenedor temporal sin red y almacenamiento tmpfs. Las suites
compartieron esa base de pruebas; no se conectaron a la base de producción.
Los tiempos no son un benchmark de capacidad empresarial. La compilación Go y
las pruebas no vuelven a validar los 32 tests Python de la auditoría anterior.

Para la suite completa, establecer `NEXUS_PG_DSN` de una base **desechable y dedicada**
y `NEXUS_CONFIG` apuntando a un archivo inexistente dentro de un directorio temporal
para evitar cargar configuración productiva. Los tests usan sockets locales;
el sandbox del entorno necesitó autorización para ello. Con esas condiciones:

```sh
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off go build ./...
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off go vet ./...
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off go test -json -p 1 -count 1 -timeout 120s ./...
go test -race -count 1 ./agents/blue/... ./evolve/...
go test -race -count 20 ./core/coordinator ./core/subagent
go test -race -count 20 -run '^TestWebFetchProxyFallback$' ./core/tool
go test -race -count 3 ./lab/security
go test -run '^$' -bench 'BenchmarkSigma.*Match$' -benchmem -benchtime=100ms -count=3 ./agents/blue/sigma
go test -run '^$' -fuzz '^FuzzParseAndCompile$' -fuzztime=6s -parallel=2 ./agents/blue/sigma
go run ./cmd/nexus-evolve -detection-lab -generations 5 -evaluation-budget 512 -json
```

El mismo comando de laboratorio con presupuesto 20 **debe fallar** por límite;
su salida es evidencia de contención, no un test omitido. Las siete omisiones
históricas corresponden al corpus Sigma externo, httpx externo, dos comparaciones
con upstream NOA, dos seeds explícitamente omitidas y un LLM live. La integración
PostgreSQL sí se ejecutó. Ver nombres exactos en `validation.json`.

## Reversión y pendientes priorizados

El parche incluye solo los 20 archivos de esta implementación; excluye informes
históricos y `load_dir_test.go`, y conserva el arreglo previo de `LoadDir`.
Antes de revertir, guardar cualquier trabajo posterior y comprobar los hashes.
Validar `git apply --check -R docs/engineering/evidence/implementation.patch`;
si coincide, aplicar el mismo comando sin `--check` y volver a ejecutar pruebas.
Esto elimina únicamente los archivos nuevos contenidos en el parche y revierte
las modificaciones de ese ciclo. No usar `git reset --hard`. No hay migraciones
ni estado productivo que restaurar. Los documentos/evidencias pueden conservarse.

| Prioridad | Siguiente cambio | Criterio de aceptación y dependencia |
|---|---|---|
| P0 | Completar alcanzabilidad de dependencias y actualizar por familias | Escanear artefacto real y símbolos, registrar rango exacto/versión candidata, regenerar lockfile/SBOM y pasar build web/Go; sin actualizar todo a ciegas |
| P0 | Resolver exposición comercial ARTEX/norma | Mapa de procedencia por componente y decisión de distribución compatible; no borrar atribuciones |
| P0 | Capturar/reproducir el 403 si reaparece | Primera denegación firmada y test que falle antes del arreglo; actualmente no reproducido |
| P1 | Frontera de ingesta Blue autenticada por organización | Contexto tenant derivado de autenticación, límites por tenant, rechazo cruzado y auditoría real; antes de exponer la biblioteca como API |
| P1 | Egreso obligatorio fuera del LLM | Rechazar destinos fuera del alcance, redirecciones y resoluciones no autorizadas en pruebas con red local; revisar herramientas y proxy antes de ampliar Red Team |
| P1 | Validación diferencial del subconjunto Sigma | Casos aceptados/rechazados frente a referencia y corpus de licencia revisada; no confundir etiquetas con cobertura |
| P2 | Holdout externo y variantes declarativas limitadas | Evaluador reservado fuera del candidato; presupuestos por proceso antes de ejecutar código no confiable; medir transferencia sin reutilizar familias |
| P2 | Purple Team de laboratorio | Escenario autorizado → telemetría → detección → positivos/negativos y tiempo; sin presentarlo como incidente real |

La corrección de fiabilidad y las nuevas bibliotecas están validadas localmente.
El triage de dependencias, la integración SaaS y la evolución completa siguen
incompletos; no se considera cerrado el programa empresarial de ocho sprints.
