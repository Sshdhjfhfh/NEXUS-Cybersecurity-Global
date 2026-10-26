# Plan de mejora BLUE

2026-10-26. Propuesta incremental basada en `agents/blue`, `agents/prediction`,
`abm`, `zero_trust`, `governance`, `evidence` y `orchestrator/level12`.
Los módulos declarados inmutables permanecen sin cambios.

## Qué existe

SENTINEL evalúa observaciones HTTP aportadas, enlaza hallazgos con evidencia y
vuelve a autorizar cada observación. No recopila telemetría de endpoints por sí
solo. ABM/zero_trust contienen implementaciones y tests, pero no basta su existencia
para afirmar que toda acción del servidor atraviese esos controles.
Sigma es un parser/matcher parcial usado en un benchmark. No se encontró
ingesta SIEM conectada a ese matcher. Predicción calcula estimaciones por clase;
no es un detector entrenado de amenazas. Evidencia conserva cuerpos por hash;
gobernanza tiene informes/notificaciones y auditoría, con procedencia heredada.

## Arquitectura propuesta

Ingesta autenticada → normalizador versionado → almacenamiento con tenant/tarea →
enrutado por fuente → reglas validadas → correlación determinista → expediente →
resumen LLM con citas → recomendación → política separada de respuesta.

Evento mínimo: organización, fuente, sensor, activo, ID estable, `observed_at`,
`received_at`, esquema/versión y digest de contenido; campos sin normalizar se
conservan como referencia con controles de acceso. La normalización debe distinguir
campo ausente, nulo, string, entero y lista. No convertir todo a strings sin evaluar
el impacto en reglas. El enrutador resuelve `logsource` antes de aplicar Sigma.

| Prioridad | Entrega | Aceptación |
|---|---|---|
| P1, realizada | Visibilizar errores de recorrido al cargar reglas | Ruta ausente y permisos fallidos generan errores; reglas válidas se conservan. Cuatro pruebas, [evidencia](VALIDATION_MATRIX.md). |
| P1 | Contrato de subconjunto Sigma: rechazar modificadores y condiciones no soportados | Comparación contra pySigma/fixtures de especificación; un operador desconocido nunca se interpreta silenciosamente. |
| P1 | Ingesta reproducible y partición por organización | Pruebas de replay, duplicados, tamaño, timestamps, credenciales y lectura cruzada; fallo del almacén no pierde eventos silenciosamente. |
| P1 | Índice de reglas con licencia, autor, origen, versión y hash | Los mensajes de coincidencia preservan autor cuando lo exige DRL; las reglas inválidas quedan en cuarentena con motivo. |
| P2 | Ventanas de correlación y deduplicación | Positivos/negativos con desorden, retrasos y relojes sesgados; no unir incidentes de organizaciones distintas. |
| P2 | Investigación repetible | Playbooks de solo lectura: timeline, activos, regla, fuentes y evidencias; cada afirmación del resumen remite a un evento existente. |
| P2 | Conector SIEM por adaptador | Primero replay JSON; después un SIEM de laboratorio con fixture de API; retry acotado e idempotencia. No desplegado en esta entrega. |
| P2 | Inteligencia de amenazas | TTL, fuente, hash y confianza; ningún indicador externo transforma una hipótesis en hecho ni concede permisos. |
| P3 | Comportamiento/grafos | Comparar baseline simple contra modelos experimentales en separación temporal antes de introducir GPU o grafos globales. |

## Evaluación

Reportar TP, FP, TN y FN por escenario y regla; precisión `TP/(TP+FP)`, recall
`TP/(TP+FN)` y FPR `FP/(FP+TN)`, dejando indefinidos denominadores cero. Añadir
alertas por unidad de tiempo, tasa de eventos descartados, retraso de ingesta y
latencia de detección. Conservar falsos negativos y casos sin telemetría; no
convertir pérdida de eventos en éxito defensivo. Ajustar umbrales en desarrollo;
reservar organizaciones/escenarios/tiempos distintos para evaluación.

El LLM puede redactar consultas y recomendaciones. Bloqueo de usuarios, aislamiento
de máquinas y cambios de firewall exigen permisos externos, vigencia y auditoría
independientes. La contención automática necesita una política previamente
autorizada y un mecanismo de reversión comprobado por acción, fuera del modelo.

Fuentes: [SigmaHQ](https://github.com/SigmaHQ/sigma),
[pySigma](https://github.com/SigmaHQ/pySigma),
[Elastic detection-rules](https://github.com/elastic/detection-rules) — estudiar
procesos; ELv2 requiere evaluación antes de reutilizar— y
[NIST 800-61r3](https://csrc.nist.gov/pubs/sp/800/61/r3/final).
Conclusión: priorizar telemetría y semántica correcta; la clasificación generativa
y las correlaciones complejas deben ganarse su coste con mejores resultados.
