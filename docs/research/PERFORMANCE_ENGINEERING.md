# Ingeniería de rendimiento

2026-10-26. No se midieron EPS, p95, RAM bajo carga ni tokens/s de NEXUS en esta
sesión. La duración de una prueba unitaria no equivale a un benchmark productivo.

## Línea base y recursos

El entorno informa ocho CPUs lógicas y aproximadamente 10,7 GiB de RAM visible
en `/proc/meminfo`; no se garantiza capacidad dedicada. `nvidia-smi` no pudo
inicializar NVML porque el SO bloquea acceso a GPU; no se infiere que no exista.
Go 1.26.3. Pruebas focalizadas y compilación están en la matriz. No se ejecutó
inferencia ni se descargaron modelos, imágenes o toolchains de terceros.

Para presupuestar inferencia, memoria de pesos aproximada `P*b/8` bytes, donde P
son parámetros y b bits por peso, antes de escalas, buffers, KV cache y runtime.
Un modelo de 8.000 millones a cuatro bits requiere unos 4 GB solo para pesos;
eso no garantiza que quepa con contexto largo o sesiones concurrentes. La KV cache
crece con capas, cabezas KV, dimensión, tokens, precisión y sesiones; medir con el
runtime/modelo exactos. Evitar dimensionar solo por nombre comercial o cuantización.

| Trabajo | Presupuesto inicial de laboratorio, no requisito medido |
|---|---|
| Reglas y tests Go, sin modelo | 4–8 CPU, 8–16 GiB RAM y SSD; PostgreSQL efímero separado. |
| Replay de telemetría/colas | 8 CPU, 16–32 GiB; variar tasa y tamaño; retención se dimensiona con volumen observado. |
| Inferencia local pequeña cuantizada | 16–32 GiB RAM; GPU de 12–24 GiB como candidato de ensayo, sujeto a modelo/contexto. |
| Fine-tuning o RL | No comprar equipo todavía: estimar según modelo, LoRA/entrenamiento completo, batch y rollout; puede requerir varias GPU. |
| Multiempresa | Procesos de herramientas separados por tenant, cuotas CPU/RAM y capacidad calculada con concurrencia real. |

Estas cifras son estimaciones de ingeniería revisables, no resultados publicados
ni compromisos de capacidad o presupuesto monetario.

## Plan reproducible

Crear datasets sintéticos licenciados con distribuciones/tamaños fijos y hashes.
Fijar commit, Go, SO, límites del contenedor y configuración. Hacer warmup, varias
repeticiones y publicar muestras. Comparar una variable por vez:

| Área | Medida | Experimento/criterio |
|---|---|---|
| Detección | EPS, p50/p95/p99, allocations y FP/FN | 100/1.000/10.000 reglas y eventos pequeños/grandes; throughput no compensa semántica incorrecta. |
| Orquestación | Latencia de cola, slots, goroutines y cancelación | 1/8/32 tareas con fallos inyectados; presupuesto total respeta límites. |
| Persistencia | Latencia de commit, bytes/evento y recuperación | Caída antes/después del commit, outbox y replay; ningún resultado ambiguo se marca exitoso. |
| Memoria | Recall@k/MRR, tokens recuperados, latencia y fuga entre tareas | Separar título exacto de consultas generales; corpus reservado por escenario. |
| LLM | TTFT, tokens/s, uso informado, coste/tarea y éxito comprobado | Mismos casos y presupuesto; clasificar fallos de proveedor y del agente por separado. |
| API | Latencia, errores, autorización por tenant y saturación | Carga solo sobre instancia temporal; límites y shedding sin perder auditoría. |

`pprof`, trazas Go y perfiles SQL deben usarse sobre laboratorio para localizar
cuellos antes de cambiar algoritmos o lenguajes. Definir SLO numérico después de
obtener una línea base; no inventar un p95 objetivo como capacidad ya alcanzada.

## Decisiones técnicas

Conservar Go en API, coordinación, conectores y control de procesos. Python para
experimentos, notebooks, evaluación y adaptadores de ecosistemas ML. Considerar
Rust solo ante un límite demostrado en parsers/agentes nativos o necesidad concreta
de seguridad de memoria; no migrar por una promesa genérica de velocidad.

Usar concurrencia acotada, backpressure y presupuestos por organización. Persistir
estado crítico antes de responder; evitar colas solo en memoria para trabajos que
deban sobrevivir. Reutilizar PostgreSQL/outbox antes de operar un broker nuevo,
salvo que las mediciones justifiquen ese coste. Separar plano de ejecución del
servidor web para no compartir su radio de fallo.

Fuentes: código `swarm`, `llm_gateway/pool`, `evolve`, `evidence/store.go`;
[diagnósticos Go](https://go.dev/doc/diagnostics),
[diseño de LangGraph](https://github.com/langchain-ai/langgraph),
[AgentEvolver](https://github.com/modelscope/AgentEvolver).
Conclusión: primero control y métricas; GPU, brokers y reescrituras quedan sujetos
a evidencia. Reversión de optimizaciones: referencia y configuración previas, con
equivalencia funcional y de autorización comprobada.
