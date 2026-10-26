# Coordinación y evolución multiagente

2026-10-26. Revisión de `core/{agent,subagent,coordinator,memory,llm}`, `swarm`,
`llm_gateway/pool`, `agents/red` y `orchestrator/server`.

## Comunicación actual

Las sesiones de `core/agent` usan `core/harness` para alternar mensajes LLM y
herramientas. `core/subagent` crea contextos hijos, filtra herramientas y limita
profundidad; el historial puede almacenarse en una sidechain. Un registro vacío
de herramientas específicas hereda las del padre: no implica mínimo privilegio.
`core/coordinator.RunParallel` reparte tareas con un semáforo y devuelve resultados
ordenados. No contiene un bus distribuido duradero.

El runtime RED organiza intents/facts/findings mediante el almacén PostgreSQL,
claims y transiciones CAS (`db`, `orchestrator/server/engine.go`); usa eventos al
planificador y conclusiones del worker. `swarm` es otro scheduler, con ocho roles
y DAG de laboratorio, que transmite payloads/resultados estructurados en memoria.
No es correcto sumar ambos y afirmar coordinación distribuida tolerante a fallos.

Memoria por tarea/chat y compresión NOA están conectadas a RED. No se encontró
memoria compartida vectorial o `knowledge_bus` implementado. El pool LLM admite
proveedores compatibles, prioridades, rotación, salud y failover. Evita volver a
enviar un stream después de entregar eventos, propiedad importante para no repetir
acciones. Ventanas y tokens son límites distintos del coste monetario total.

## Arquitectura objetivo sin sustitución del runtime

| Componente | Propuesta original | Evidencia de aceptación |
|---|---|---|
| Mensajes | Sobre versionado con tenant, task, run, intent, causation, correlation, schema y digest | Rechazo de mensajes cruzados, corruptos, enormes o repetidos. |
| Asignación | Extender claims existentes con lease, fencing y capacidades; cola/outbox transaccional | Caída tras claim, antes/después del commit y durante resultado: no se pierde trabajo ni repiten efectos. |
| Especialización | Capacidades de análisis, verificación y reporte declaradas por host | Un rol no obtiene herramientas/permisos de otro por texto del prompt. |
| Memoria privada | Separar contexto y notas por agente/tarea; retención y acceso explícitos | Canarios y pruebas de lectura cruzada; borrar tarea invalida acceso y cachés. |
| Memoria compartida | Solo hechos con procedencia, estado y ACL; promoción desde notas mediante verificación | Una respuesta de otro agente no se vuelve verdad por repetirse. |
| Recuperación | Heartbeats, timeout, retry acotado y cuarentena de trabajo no idempotente | Inyección de fallos de proceso/red/almacén, sin duplicación de acciones. |
| Presupuesto | Ledger por tarea y organización para tokens, llamadas, tiempo y coste | Reserva antes de ejecutar, conciliación tras respuesta y tope bajo concurrencia. |
| Verificador | Proceso/identidad distintos, sin modificar candidato ni evaluador durante ensayo | Candidato no puede escribir expectativas, políticas ni resultados del juez. |

El scheduler debe adquirir slots respetando cancelación y no iniciar nuevos
workers tras vencer el plazo. Go no puede terminar de forma segura una goroutine
arbitraria: los procesos de herramientas deben tener supervisión del host.
No prometer exactamente una ejecución; usar al menos una entrega con idempotencia
en operaciones apropiadas y revisión explícita donde haya efectos irreversibles.

## Proveedores locales y remotos

Mantener `llm.Provider` como contrato estable. Evaluar por tarea un endpoint local
compatible y dos proveedores remotos configurados por el operador; con datos
sintéticos y autorización de transferencia. Medir éxito verificable, latencia,
tokens, coste y fallos. Separar generador de evaluador también en identidad y
permisos; dos modelos coincidentes no son un oráculo de verdad.

Primero routing estático con fallback y presupuesto. Después, routing adaptativo
solo si mejora un conjunto reservado frente al mismo presupuesto. Los modelos
locales reducen salida de datos, pero requieren memoria para pesos, KV cache y
concurrencia; no se han medido modelos en esta sesión.

## Orden y decisión

Primero pruebas de contratos y separación de organizaciones; después recuperación
y ledger; luego memoria compartida y routing. Estudiar checkpoints de
[LangGraph](https://github.com/langchain-ai/langgraph), sin reemplazar el core Go.
[AutoGen](https://github.com/microsoft/autogen) sirve de referencia histórica: su
README anuncia archivo/no mantenimiento, aunque `archived` de la API observado
era falso; no elegirlo como nueva dependencia estratégica sin aclarar esa discrepancia.
[ReAct](https://arxiv.org/abs/2210.03629) motiva el ciclo acción/observación;
[AgentDojo](https://arxiv.org/abs/2406.13352) aporta evaluación de contenido no fiable.

Riesgos: carreras de presupuesto, contaminación de memoria, efectos repetidos,
fuga entre tenants y mensajes que contienen instrucciones hostiles. Reversión:
adaptadores y workers nuevos desactivables; mantener lectura compatible y no
retirar validaciones para restaurar disponibilidad. Todo lo anterior es propuesta,
no una capacidad distribuida declarada como terminada.
