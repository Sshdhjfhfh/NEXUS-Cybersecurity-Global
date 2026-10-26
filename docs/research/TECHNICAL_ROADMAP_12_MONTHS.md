# Hoja de ruta técnica y estratégica

2026-10-26. Estimaciones revisables desde el inicio del programa, no promesas de
entrega ni demostración de que una fase está terminada. La investigación inicial
y el arreglo de carga Sigma de esta rama no completan las fases 0–3.

## Prioridades y dependencias

| Fase | Ventana orientativa | Entrega y dependencias | Criterio de salida |
|---|---|---|---|
| 0 Diagnóstico | Días 1–7 | Inventario, pruebas reproducibles y límites de documentación | Fallos/omisiones registrados, configuración de test independiente, capacidades clasificadas sin porcentajes ficticios. Auditoría exhaustiva adicional pendiente. |
| 1 Investigación | Días 1–14 | Repositorios y literatura vinculados a brechas reales | SHA/licencia/fuentes y matriz; no adoptar proyecto sin coste/riesgo. Se entregó revisión priorizada de 18 repositorios. |
| 2 Diseño | Días 5–21 | Contratos de acción/evidencia/tenant, amenazas y ADR; depende de 0/1 | Revisión de límites entre runtime heredado, laboratorio y runners, con aceptación por casos negativos. |
| 3 Fundamentos | Días 7–30, continuable | Observabilidad, aislamiento, políticas, SBOM y tests; depende de 2 | Pruebas de salida de alcance, auditoría fallida, revocación y separación entre organizaciones. Fallos de suite resueltos o explicados reproduciblemente. |
| 4 RED | Días 31–60 | Runner y adaptadores con hallazgos verificables; depende de 3 | Escaneo de fixtures propios sin egreso no autorizado; falsos positivos y reproducción medidos. |
| 5 BLUE | Días 31–75 | Normalización, Sigma estricto, replay e investigación; depende de 3 | Positivos/negativos, fuente/tenant correctos, errores visibles y atribuciones de reglas. |
| 6 PURPLE | Días 60–90 | Escenario→evento→detección→informe; depende de 4/5 | Estados de cobertura reales, tiempos y telemetría faltante separados. |
| 7 Multiagente | Meses 3–6 | Recuperación, idempotencia, coste y memoria; depende de 3 y tareas evaluables | Pruebas de caída/restart/replay sin efectos duplicados y sin fuga entre tenants. |
| 8 Evolución | Meses 4–9 | Generaciones con evaluación sellada; depende de 6/7 | Transferencia a casos independientes, seguridad/coste y regresiones publicados. No requiere modificar restricciones. |
| 9 Validación empresarial | Meses 4–12 | Carga, resiliencia, licencias, operación y pilotos; depende de entregas anteriores aplicables | SLO medidos, backup/restore, revisión de seguridad, soporte y capacidades comercializables demostradas. |

Las fases 4 y 5 pueden trabajar sobre fundamentos comunes después de su aceptación;
no se deben ampliar ofensiva, autonomía y aprendizaje a la vez sin poder atribuir
regresiones. Cada entrega requiere un cambio pequeño, tests, evidencia y reversión.

## Calendario de inversión

**Primeros siete días:** completar inventario, fijar pruebas aisladas, reproducir
WebFetch/403 observados, definir oráculo Sigma y mapa de permisos reales. Resolver
prioridad jurídica del núcleo y trazar componentes absorbidos. El arreglo inicial
de errores de recorrido ya está implementado, sin desplegar.

**Primeros treinta días:** corregir fallos confirmados, exigir semántica explícita
de reglas, medir autorización por acción y egreso, separar organización/tarea en
evidencias y establecer métricas. Triage OSV con alcanzabilidad antes de cambios
masivos. No sustituir OPSEC/ética ausentes por callbacks permisivos.

**Días 31–90:** runner de laboratorio, primeros conectores defensivos, dos escenarios
PURPLE con controles negativos y primeras pruebas de recuperación de workers.
Comparar tareas completas con baseline sin LLM y presupuesto equivalente.

**Meses 4–6:** carga, colas duraderas si se justifican, operación de identidad y
credenciales, retención y restauración, integración empresarial y pilotos
delimitados. Optimizar después de perfiles; no migrar lenguajes preventivamente.

**Meses 7–12:** ampliar escenarios y fuentes, evaluar memoria/políticas/estrategias
en test nuevo y fortalecer servicio/soporte. Investigar variantes de generadores
solo después de evaluación independiente. Preparar documentación comercial basada
en ensayos reproducibles, sin afirmar SOC autónomo o zero-days no medidos.

## Horizonte de tres años

| Año | Investigación y producto | Puerta de decisión |
|---|---|---|
| 1 | Plataforma fiable de evidencia, auditoría autorizada, triage asistido y validación de detecciones | Pilotos con contratos, licencias resueltas, incidentes operativos y métricas reales. |
| 2 | Cobertura multifuente/multiempresa, grafos de incidentes y estrategias transferibles; modelos locales/remotos comparados | Beneficio frente a reglas/heurísticas simples, generalización entre entornos y coste de operación sostenible. |
| 3 | Metaoptimización de agentes/generadores y posibles modelos especializados; investigación publicable | Replicación independiente, ganancias al mismo presupuesto, límites de seguridad inmutables y estabilidad a largo plazo. Sin garantía de mejora recursiva abierta. |

## Respuestas a las quince preguntas

1. **Qué funciona:** sesiones/agentes y coordinación por intents, herramientas,
   memoria/compresión, pool LLM, análisis HTTP pasivo, parser Sigma parcial,
   gates/identidades y evolución acotada; pruebas detalladas en la matriz. No todas
   esas funciones están conectadas de extremo a extremo en producción.
2. **Qué está incompleto:** OPSEC/ética para composición del lab, semántica Sigma,
   circuito PURPLE, recuperación distribuida, telemetría multifuente y aislamiento
   empresarial demostrado. `adversarial_rt` simula; CTF no tiene solver conectado.
3. **Diez repositorios útiles:** Sigma, pySigma, ATT&CK STIX, Atomic Red Team,
   PentAGI, Strix, OSS-Fuzz, LangGraph, SED y Caldera, con las condiciones de la
   [matriz](GITHUB_REPOSITORY_RESEARCH.md).
4. **Ciencia aplicable:** ReAct/PentestGPT para planificación con evidencia;
   AgentDojo para robustez; Reflexion/RAG para memoria; UNICORN/KAIROS para
   correlación. CyberEvolver/SED/ADAS son experimentos posteriores, no capacidades
   actuales de NEXUS. Ver [investigación científica](SCIENTIFIC_RESEARCH.md).
5. **Desarrollo original:** contratos y verificador de hallazgos, evidencia por
   tenant, evaluador PURPLE, scheduler/resiliencia y mecanismo de promoción; son
   diferenciadores integrados con el sistema existente, sujetos a procedencia.
6. **Open source a integrar:** datos ATT&CK, reglas permitidas, validación Sigma,
   escenarios selectivos y herramientas aisladas por adaptador; no plataformas
   enteras o dependencias redundantes.
7. **Licencias comerciales:** AGPL actual/derivados, restricciones CAI y ELv2,
   ausencia de licencia CyberEvolver, DRL/LGPL y componentes mixtos Wazuh/AutoGen.
   AGPL no prohíbe vender, pero condiciona modalidad y fuente correspondiente.
8. **Límites arquitectónicos:** dos modelos de coordinación, permisos diferentes
   entre runtime y laboratorio, procesos externos cuyo egreso no prueba el gate,
   memoria local y ausencia de durabilidad distribuida demostrada.
9. **Mejorar RED/BLUE con seguridad:** runner con egreso/política de host, eventos
   normalizados, reglas de semántica estricta y hallazgos reproducidos por un
   evaluador separado. Mantener producción fuera de los experimentos.
10. **Verificar evolución:** datos agrupados independientes, test sellado externo,
    mismos presupuestos, hashes completos, negativos, métricas pareadas, revisión
    de seguridad y reversión. Acuerdo entre LLMs no basta.
11. **Cómputo:** CPU/SSD para reglas y tests; inferencia local según pesos/contexto;
    GPU para entrenamiento solo tras justificarlo. [Estimaciones y fórmulas](PERFORMANCE_ENGINEERING.md),
    sin mediciones de GPU o throughput en esta sesión.
12. **Inversiones:** primero calidad/aislamiento/licencias y evidencia; después
    conectores/observabilidad, recuperación y pilotos; GPU/RL/grafos aprendidos
    después de baselines. Equipo inicial propuesto: responsables de backend,
    detección, validación/SRE y revisión de licencias, con roles combinables.
13. **Productos posibles:** auditoría web autorizada con retest, revisión asistida
    de incidentes y validación continua de detecciones. Son hipótesis comerciales;
    requieren pilotos, obligaciones de datos y soporte antes de vender capacidades.
14. **Riesgos:** ejecución fuera de alcance, evidencia/tenant mezclados, feedback
    contaminado, semántica incorrecta, dependencia de proveedores, resultados
    simulados vendidos como reales, licencias y jurisdicciones/datos de clientes.
    Los requisitos legales por país se revisarán sobre servicios concretos.
15. **Siguiente mejora:** tras el arreglo entregado, definir/rechazar explícitamente
    condiciones/modificadores Sigma no soportados y contrastarlos con un oráculo
    independiente. En paralelo de planificación, diseñar la prueba de egreso del
    scanner; su seguridad es condición previa a ampliar evaluación ofensiva.

## Revisión y seguimiento

Revisar prioridades cada dos semanas con evidencia de calidad, coste y riesgo.
No asignar porcentaje de terminación por número de archivos. Detener una línea
experimental que no supere baseline o que incremente permisos/coste sin beneficio.
Fuentes del plan: [auditoría](NEXUS_TECHNICAL_AUDIT.md),
[comparativa](GITHUB_REPOSITORY_RESEARCH.md), [NIST SSDF](https://csrc.nist.gov/pubs/sp/800/218/final)
y [ATT&CK](https://attack.mitre.org/resources/attack-data-and-tools/).
