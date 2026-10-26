# Matriz de validación y entrega

2026-10-26. Base: `21ac386100218b2ab0b8f907e6aa52e9e8906698`.
Rama: `research/nexus-evidence-20261026`. Cambios locales revisables, sin despliegue.

## Resultados reales

Los recuentos Go incluyen subtests y semillas de fuzzing ejecutadas por `go test`;
no equivalen a tareas operativas independientes. Los 14 paquetes sin tests se
reportan aparte de los casos omitidos. Evidencia con tiempos, nombres y estados:
[test-results.json](evidence/test-results.json). Los logs crudos completos están
en `/tmp/nexus-research-*.jsonl` durante esta sesión; sus SHA256 se preservan en
el resumen. No se archivaron cuerpos completos de tráfico o logs de credenciales.

| Comprobación | Resultado | Alcance y límite |
|---|---|---|
| Línea base focalizada, antes de editar el repositorio | 15 paquetes aprobados; 90 casos aprobados, 2 omitidos | BLUE/Sigma/detectbench, predicción, scanner, swarm, lab/network, evolve/recall, memoria, plan/coordinador/subagente y pool. |
| Compilación inicial `go build ./...` | Código de salida 0; advertencia de caché en sandbox; repetición autorizada 0 sin advertencia | No arranca servicios ni compila UI embebida. |
| `go vet ./...` inicial y posterior | Código 0 | Análisis estático; no sustituye revisión de seguridad. |
| Pruebas nuevas sobre cargador sin arreglar | Dos fallos reproducidos: ruta ausente y subdirectorio sin permisos | [Log de regresión](evidence/loader-regression-before.log); los otros dos casos de compatibilidad pasaron. |
| Suite completa inicial en sandbox | 41 paquetes aprobados, 15 fallidos y 14 sin tests | Fallos de sockets/DB del entorno más regresiones nuevas; no usar como veredicto del producto. |
| Suite completa autorizada, antes del arreglo | 53 paquetes aprobados, 3 fallidos y 14 sin tests; 1.917 casos aprobados, 4 fallidos, 7 omitidos | Incluye dos fallos nuevos del cargador y dos preexistentes observados: WebFetch y composición segura. PostgreSQL aislado. |
| `go test -race -count=1 ./agents/blue/... ./evolve/...` posterior | 5 paquetes aprobados; 39 casos aprobados y 1 omitido | Incluye las cuatro pruebas nuevas. Sin carreras detectadas en estos paquetes/recorridos. |
| Suite completa posterior `go test -json -p 1 -count=1 -timeout 120s ./...` | **Código 1: 55 paquetes aprobados, 1 fallido, 14 sin tests; 1.920 casos aprobados, 1 fallido y 7 omitidos** | `TestWebFetchProxyFallback` sigue agotando el plazo. No se declara suite verde. |
| Compilación posterior `go build -buildvcs=false ./...` | Código 0 | Sin estampado VCS; backend/default tags, no build Docker ni frontend. |
| Python `zero_day` | 11/11 aprobadas | Unitarias de campaña/guía sobre fixtures; no se ejecutó AFL++ contra software real. |
| Python `lab/academy` | 8/8 aprobadas | Material/evaluador sintéticos; no inferencia ni entrenamiento. |
| Python `lab/benchmark` | Primer intento: 5 errores de sockets; repetición autorizada: 13/13 aprobadas | Servidores HTTP de loopback propios, sin objetivos externos. |
| Inventario Go/npm + consulta OSV | 1.021 componentes, respuesta por cada consulta; 29 componentes con avisos | Coincidencias de versión, sin análisis de alcanzabilidad ni imagen de producción. |
| SBOM CycloneDX 1.6 | Cero errores de esquema; referencias únicas y correspondencia inventario/OSV | Inventario agregado de manifiestos, no SBOM completo de la imagen. |
| Revisión de GitHub | 18 repositorios existentes; 74 fuentes de archivos con URL/SHA256 | No se ejecutaron sus suites ni se reprodujeron sus resultados. |

## Fallos y omisiones que permanecen

**WebFetch:** el test configura un proxy en `127.0.0.1:1`, espera un error que
permita fallback directo y termina con `context deadline exceeded` a unos 30 s.
Ocurrió antes y después del arreglo de Sigma. Se inspeccionó el test y el código;
no se determinó si la causa raíz es el entorno o un defecto del manejo de proxy.
No se cambió esa lógica, no se relajó el proxy y no se convirtió el fallo en skip.
Además, el fallback directo merece revisión cuando el proxy sea un control de
captura/autorización obligatorio; no se asume que lo sea en todas las instalaciones.

**Composición segura:** `TestLabSecureCompositionWithRealIdentities` devolvió
403 en la primera suite completa autorizada, con `detail=<nil>` para una decisión
de gate reconsultada por su fixture. Pasó en la segunda suite sin modificar ese
componente. Posible dependencia de estado/orden o condición transitoria, sin causa
confirmada. El mismo PostgreSQL temporal se reutilizó entre esas dos ejecuciones;
una reproducción con base recién creada por ejecución es pendiente explícito.
Un pase posterior no demuestra que el comportamiento intermitente esté resuelto.

Siete casos omitidos en la suite completa posterior:

- `TestLoadRealSigmaRules`: corpus externo no configurado.
- `TestHTTPXAuthorizesThenProbes`: binario httpx no instalado.
- `TestHowToCompressRulesMatchesUpstream` y `TestTierRulesAreVerbatim`: fuentes upstream no disponibles en el fixture.
- `FuzzTruncateBody/seed#2` y `/seed#3`: omisiones definidas por los fixtures de semillas; no se realizó una campaña de fuzzing.
- `TestLiveContextReview`: integración con modelo real opt-in sin configurar.

No se verificaron UI/build estático, SSO/RBAC productivo, recuperación operacional,
aislamiento completo entre empresas, rendimiento bajo carga, contención productiva,
descubrimiento de zero-days, ni mejora recursiva. No se ejecutaron pruebas ofensivas
contra terceros. La auditoría no certifica todas las superficies del código.

## Cambio implementado y justificación

| Archivo | Modificación |
|---|---|
| `agents/blue/sigma/sigma.go` | `LoadDir` registra errores de `filepath.WalkDir` con ruta y causa envuelta, antes de consultar `DirEntry`. Mantiene la firma y la carga parcial de reglas válidas. |
| `agents/blue/sigma/load_dir_test.go` | Cuatro pruebas: raíz ausente, árbol vacío, mezcla de regla válida/error y subdirectorio inaccesible. El caso de permisos se omite cuando el usuario puede leer pese a `chmod`; se ejecutó y aprobó en este entorno. |
| `docs/research/*.md` | Los doce documentos de auditoría, investigación, planes, licencias y validación solicitados. |
| `docs/research/evidence/*` | Inventarios, metadatos/hashes públicos, resultados de tests, SBOM, OSV y snapshots textuales de helpers propios. |

El fallo podía presentar «cero reglas y cero errores» cuando la fuente ni siquiera
era accesible. Se mejora la trazabilidad sin introducir dependencias, reglas
externas, permisos, acciones ofensivas ni cambios de configuración. Los módulos
inmutables, secretos, bases productivas y APIs del servidor no se modificaron.

Riesgo/regresión posible: un consumidor que hasta ahora ignoraba un problema de
recorrido puede recibir más errores y decidir abortar. Es el contrato esperado de
la lista de errores; este cambio no impone abortar ni garantiza por sí solo que
los consumidores inspeccionen esa lista. No corrige semántica avanzada Sigma.

## Reproducción segura

Desde la raíz, con Go 1.26.3 y dependencias disponibles:

```sh
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off go build -buildvcs=false ./...
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off go vet ./...
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off go test -race -count=1 ./agents/blue/... ./evolve/...
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s zero_day -p 'test_*.py' -v
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s lab/academy -p 'test_*.py' -v
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s lab/benchmark -p 'test_*.py' -v
```

La suite completa requiere un PostgreSQL **descartable**. No apuntar tests al DSN
habitual: fixtures migran, escriben y limpian registros. En esta sesión se usó una
imagen local `postgres:16-alpine` con `--pull never`, `--network none`, datos tmpfs
y un socket Unix dedicado `/tmp/nexus-research-pg-socket`. La ID de imagen está en
el resumen; no se inspeccionó la base de producción. `NEXUS_CONFIG` se apuntó a una
ruta temporal inexistente para impedir fallback a configuración habitual.
Al finalizar se retiró exclusivamente el contenedor temporal
`nexus-research-pg-20261026`; no se detuvieron ni modificaron servicios productivos.

Después de crear una base de test nueva y definir su variable explícita:

```sh
NEXUS_CONFIG=/tmp/nexus-research-no-config.json \
NEXUS_PG_DSN="${NEXUS_TEST_PG_DSN:?define a disposable PostgreSQL DSN}" \
GOCACHE=/tmp/nexus-research-go-cache GOPROXY=off GOSUMDB=off \
go test -json -p 1 -count=1 -timeout 120s ./...
```

Hace falta permiso del entorno para sockets locales; un error del sandbox no es
un fallo funcional confirmado. `-p 1` reduce conflictos entre suites que comparten
fixtures, sin demostrar aislamiento perfecto. Los helpers de investigación propios
se archivan como `.py.txt`, con rutas de esta sesión; para reproducir consultas,
revisar y ajustar esas rutas. No contienen scripts externos ni se ejecutan al
arrancar NEXUS. Metadatos GitHub y avisos OSV cambiarán: comparar fecha/SHA.

## Reversión y pendientes

Para revertir exclusivamente la mejora de código, tras conservar su diff y
comprobar que no existen cambios posteriores del usuario en esos archivos:

```sh
git restore --source=21ac386100218b2ab0b8f907e6aa52e9e8906698 -- agents/blue/sigma/sigma.go
mv agents/blue/sigma/load_dir_test.go /tmp/nexus-load-dir-regression-test.go
```

No ejecutar `git reset --hard` ni borrar módulos. Conservar documentación y
evidencias como registro del fallo. No es necesario reiniciar producción: el cambio
no fue desplegado. Si se publica mediante un commit futuro, revertir ese commit
de código y sus pruebas juntos, sin retirar controles de seguridad.

Pendientes prioritarios: causa raíz del timeout y del 403 intermitente; conformidad
Sigma y observación por fuente; pruebas de egreso por conexión en scanners;
aislamiento de tenant; fuentes upstream de memoria; corpus/targets reales propios;
SBOM de imagen/Python/modelos y procedencia ARTEX/norma. El resto de fases conserva
sus criterios de salida en [TECHNICAL_ROADMAP_12_MONTHS.md](TECHNICAL_ROADMAP_12_MONTHS.md).
