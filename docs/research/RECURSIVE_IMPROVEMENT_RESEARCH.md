# Investigación de mejora recursiva verificable

2026-10-26. La distinción entre optimización acotada y nueva capacidad es obligatoria.

## Estado medible actual

`evolve.Engine` impone catálogo, política, límites, comparación pareada,
holdout, presupuesto y persistencia. `evolve/recall` adapta recuperación por
títulos y revierte ante cambios/regresión. No genera nuevos algoritmos libremente.
`detectbench` compara reglas originales estáticas: train/holdout contienen eventos
repetidos con IDs distintos y el digest incluye IDs/etiquetas, no todo el contenido.
Sirve para probar mecánica del motor, no para afirmar transferencia de detección.

`rsi_detector` registra series y alerta sobre tendencias. Una aceleración en sus
indicadores no demuestra que un agente haya mejorado el mecanismo que lo mejora.
`selfimprove/update` instala binarios y verifica checksums; no es evolución.
`lab/academy` exporta lecciones y puntúa ejercicios sintéticos sin entrenar pesos.

| Tipo | Evidencia necesaria | Estado NEXUS |
|---|---|---|
| Adaptación de prompts | Hash de prompt, tareas reservadas, mismo presupuesto | Posible experimento; no demostrada aquí. |
| Optimización de herramientas | Resultado correcto con menor coste/latencia | Adaptadores y tests existen; mejora cuantificada pendiente. |
| Estrategias | Transferencia a escenarios no usados al generar variantes | Catálogo de estrategias acotado; generalización no demostrada. |
| Recuperación de memoria | Consultas y corpus reservados sin duplicación | Recuperación de títulos implementada; no equivale a comprensión del contenido. |
| Habilidades | Contrato, procedencia, verificación y transferencia | Skills existen; generación autónoma validada pendiente. |
| Cambio de código | Checkout aislado, compilación/tests externos y adversariales | No permitido a candidatos sobre el evaluador o el sistema activo. |
| Ajuste de pesos | Dataset licenciado, configuración, checkpoints, GPU y examen independiente | No implementado por exportar JSONL o usar memoria. |
| Mejora recursiva | Un generador nuevo produce mejores variantes que el anterior en familias nuevas, a igual presupuesto | Hipótesis científica; no demostrada. |

## Protocolo de generaciones

Congelar versión de tarea, evaluador, política, toolchain y alcance. Dividir por
escenario, organización y tiempo, no por ID de evento. Mantener desarrollo,
selección y test sellado separados. Agrupar muestras correlacionadas y conservar
la unidad independiente en el análisis estadístico.

Cada generación produce un manifiesto: parent, hipótesis, diff, hashes de datos,
modelo/versión/parámetros, semillas, recursos y pruebas. El generador solo ve
inputs de desarrollo y trazas redactadas; no ve etiquetas reservadas. El evaluador
ejecuta casos fuera del workspace candidato y conserva su identidad/configuración.
Errores, timeout y fallos de seguridad cuentan explícitamente, sin descartarlos.

Propuesta inicial revisable: hasta cuatro variantes por generación y diez
generaciones, con tope de tiempo y tokens acordado por experimento. Seleccionar
con desarrollo; evaluar un ganador en reservado. No reintentar diferentes variantes
contra el mismo reservado hasta lograr un resultado favorable. Renovar el test
cuando se use su feedback para diseñar candidatos. Usar comparación pareada,
intervalos y corrección por múltiples comparaciones según el diseño.

Promoción exige: mejora práctica predefinida, seguridad no inferior, negativos sin
regresiones críticas, coste dentro de política y evidencia reproducible. Las cotas
del motor actual son una política inicial, no una justificación estadística universal.
Separar promoción de estrategia de despliegue de código. Una mejora experimental
ofensiva no se despliega automáticamente en producción.

## Experimento del generador

Comparar G0 y G1 con la misma familia de variantes permitidas, presupuesto y
modelos; evaluar sus mejores descendientes en escenarios nuevos. Medir éxito,
seguridad, diversidad, coste por variante aceptada y estabilidad entre semillas.
Incluir búsqueda aleatoria acotada y selección manual como controles. Si G1 solo
aprovecha el test observado, reportar sobreajuste; si produce más intentos por mayor
presupuesto, no atribuirle mejora recursiva.

## Fuentes, transferencias y límites

[Reflexion](https://arxiv.org/abs/2303.11366): memoria verbal sin actualizar pesos.
[Voyager](https://arxiv.org/abs/2305.16291): biblioteca de habilidades en Minecraft,
sin prueba directa de ciberseguridad.
[ADAS](https://arxiv.org/abs/2408.08435): búsqueda de sistemas en código,
con necesidad de sandbox y test independiente.
[CyberEvolver](https://arxiv.org/abs/2605.26195): mutación estructurada por objetivo;
no demuestra transferencia universal y su repo carece de licencia raíz identificada.
[SED](https://arxiv.org/abs/2609.36603): políticas recuperadas tras juzgar episodios,
sin entrenamiento; el juez puede equivocarse y la memoria necesita defensa ante
envenenamiento. Adaptar ideas de memoria sin dar a los candidatos autoridad sobre
las restricciones del host.
[AgentEvolver](https://arxiv.org/abs/2511.10395): pipeline RL distinto, con coste
de entrenamiento y entornos que no se justifican aún para NEXUS.

Decisión: completar independencia y hashes del benchmark antes de añadir
generación de código o GPUs. Reversión de cada experimento: estrategia base,
memoria/catálogo previo y registro de motivo; conservar pruebas de la regresión.
