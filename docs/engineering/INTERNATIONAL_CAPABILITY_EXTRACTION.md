# Extracción internacional de capacidades — 2026-10-26

Esta ampliación utiliza los [18 repositorios ya auditados](../research/GITHUB_REPOSITORY_RESEARCH.md).
No repite sus consultas ni cambia sus resultados históricos. Se consultaron API,
árbol, README, licencias y manifiestos de los nuevos candidatos. La disponibilidad
de código y pruebas no equivale a reproducir sus resultados ni a auditar toda su
cadena de dependencias. Ninguno se ejecutó ni se incorporó como dependencia.
Los cambios originales comprobados se detallan en [implementación](IMPLEMENTATION_20261026.md).

Se conservan SHA, fechas UTC, URLs consultadas, errores y hashes de archivos
inspeccionados en [metadatos](evidence/international-repositories.json).
Una fecha de push puede pertenecer a otra rama. No se usan estrellas para ordenar
decisiones. La procedencia institucional sirve para diversificar fuentes; no
constituye una medida de calidad o de capacidad de un país.

## Matriz de los nuevos candidatos

Cada fila reúne problema/método/resultados observables/limitaciones; situación
actual en NEXUS y transferencia propuesta; dependencias/recursos/riesgos/licencia;
y prueba de aceptación. Los costes de hardware externos no se midieron. Las
clasificaciones son decisiones de ingeniería de esta sesión, no afirmaciones de
superioridad publicadas.

| Referencia y decisión | Problema, enfoque, evidencia y limitación | NEXUS actual, extracción e implementación propia | Dependencias, recursos, seguridad, licencia y aceptación |
|---|---|---|---|
| [Tencent AI-Infra-Guard](https://github.com/Tencent/AI-Infra-Guard) — **ADAPTAR** conceptos | Inspección de infraestructura IA, MCP y skills; plugins y comprobaciones. Código y pruebas localizados; eficacia no reproducida aquí. Un inventario de checks no acredita protección universal. | Existen permisos de herramientas; falta evaluar contratos MCP y memoria con un corpus reservado. Desarrollar fixtures propios de herramientas fuera de alcance e instrucciones no confiables antes de conectar checks. | Python/Go y plugins; coste depende del escáner/modelo, sin medición. Riesgo de ejecución de plugins y datos hostiles. Apache-2.0: conservar avisos y revisar dependencias. Aceptación: positivos/negativos de autorización e inyección; ninguna decisión del modelo altera permisos. |
| [Alibaba LSCBench](https://github.com/alibaba/last-secure-code-benchmark) — **EXPERIMENTAR** | Evalúa generación de código con tests funcionales y de seguridad separados; benchmark y paper disponibles. Casos deliberadamente vulnerables requieren laboratorio. No ejecutado en NEXUS. | Ya hay tests de regresión y evaluador evolutivo. Tomar el principio de verificadores separados; diseñar tareas originales y reservadas. El ciclo actual no modifica código. | Python, contenedores y toolchains según tarea; recursos no medidos. Riesgo de código vulnerable/ejecutable y filtración de tests. Apache-2.0; revisar licencias de cada fixture. Aceptación: código funcional inseguro debe ser rechazado por un evaluador externo al candidato. |
| [ACESEvals](https://github.com/microsoft/ACESEvals) — **ADAPTAR** contratos | Tareas/grading declarativos sobre ACES e inspect_ai; incluye dominios de investigación de amenazas. Archivos inspeccionados; no reproduce por sí solo resultados de modelos. | NEXUS tiene `evolve.Benchmark`, nuevos digests y motor pasivo. Reimplementar un contrato de tarea/evidencias separado de estrategia; futuro adaptador de solo lectura, sin sustituir el runtime Go. | Python, ACES, inspect_ai y servicios/datos por tarea. CPU/tokens crecen con casos y herramientas. Riesgos de datos sensibles y costes. MIT; dependencias por separado. Aceptación: un mismo caso/presupuesto evalúa agente individual y coordinado con grader reservado. |
| [SecRL](https://github.com/microsoft/SecRL) — **EXPERIMENTAR** | Investigación de amenazas mediante consultas y aprendizaje; ofrece ExCyTIn. README remite al harness ACESEvals más reciente. Evaluación científica publicada; no reproducida aquí. | Falta investigación multisalton sobre telemetría real. Primero consultas de solo lectura y respuestas ligadas a evidencia; posponer RL hasta tener datos y baselines. | Python/SQL y modelos según experimento; entrenamiento puede requerir GPU, dimensionamiento pendiente. Riesgo de consultas sin límites y contaminación train/test. MIT. Aceptación: respuestas puntuadas contra evidencias reservadas, con presupuesto y sin acceso de escritura. |
| [DefenderBench](https://github.com/microsoft/DefenderBench) — **DESCARTAR** integración comercial actual | Evaluación variada de tareas defensivas; código/paper consultados. No se ejecutó. La disponibilidad pública no concede todos los derechos. | Estudiar la separación por tareas como referencia conceptual; desarrollar fixtures NEXUS originales. No incluir materiales, código ni datasets en el producto. | La licencia raíz es **Microsoft Departing Intern License Terms**, no MIT: uso de investigación no comercial y restricciones de redistribución/servicio. Recursos no medidos. Aceptación previa a cualquier reutilización: permiso compatible verificable; actualmente no satisfecho. |
| [CyberBattleSim](https://github.com/microsoft/CyberBattleSim) — **EXPERIMENTAR** | Simulación de decisiones de agentes en grafos y aprendizaje por refuerzo. Entorno Python disponible; no representa una intrusión real ni prueba transferencia a redes. | `lab`, `ctf_engine` y `evolve` permiten un adaptador posterior. Construir escenarios pequeños originales con alcance inmutable y evaluador separado; no importar un agente ofensivo a producción. | Python y librerías RL; coste proporcional a escenarios/episodios, sin medición. Riesgo de sobreajuste al simulador. MIT. Aceptación: presupuestos iguales, semillas registradas, escenarios reservados y acciones fuera de alcance rechazadas. |
| [KLara](https://github.com/KasperskyLab/klara) — **DESCARTAR** integración no verificable | La API y URL indicadas devolvieron 404. Resultados indexados históricos describen hunting YARA, pero no permiten confirmar fuente/licencia/mantenimiento actuales. | NEXUS no adquiere YARA por mencionar KLara. Definir interfaz de escaneo local con reglas de licencia revisada, si se valida otro componente. | Dependencias, costes y licencia actuales **no verificados**. Riesgo de atribuir capacidades o derechos inexistentes. Aceptación: fuente accesible, licencia clara y muestras positivas/negativas; pendiente. |
| [OSV-Scanner](https://github.com/google/osv-scanner) — **INTEGRAR**, pendiente de ejecución controlada | Análisis de dependencias/manifiestos e imágenes; código Go y pruebas observables. No se ejecutó el binario externo aquí. Un match de versión no demuestra explotabilidad. | SBOM y consultas OSV ya existen. Añadir más adelante un adaptador CLI fijado por versión y JSON, con alcanzabilidad por artefacto; el triage actual conserva los 29 componentes visibles. | Go y bases/API de avisos; CPU/red según artefacto, sin medida local del scanner. Riesgo de datos incompletos y falsos descartes. Apache-2.0. Aceptación: fixture vulnerable/corregido, fallos de red explícitos y separación manifest/artefacto/símbolo. |
| [SeClaw / SPECSYNTH-CLAWBENCH](https://github.com/seclaw-eval/seclaw-eval) — **ADAPTAR** método | El README actual usa el nombre SPECSYNTH-CLAWBENCH; tareas sintéticas, servicios MCP simulados y graders por tarea. No es telemetría real. | Adoptar separación entre servicios del escenario y grader. Crear casos originales de atribución, inyección y ejecución fuera de alcance; no afirmar que NEXUS ya pasó su benchmark. | Python/Docker y modelos; consumo no medido. Riesgo de evaluar solo mocks o exponer datos reservados. Apache-2.0, revisar datos por separado. Aceptación: corroborar controles con ANS/NHI y procesos reales además de fixtures. |
| [LLM-Agent-Security-Survey](https://github.com/sunyinggang/LLM-Agent-Security-Survey) — **EXPERIMENTAR** bibliografía | Catálogo de investigaciones sobre riesgos de agentes; no es un motor defensivo ni un benchmark ejecutado. | Usar referencias primarias para diseñar pruebas de memoria y herramientas; no añadir otro runtime. No se copió contenido al producto. | Sin licencia raíz identificada; no reutilizar código/texto como componente. Recursos de la bibliografía triviales; los experimentos dependen de cada paper. Aceptación: validar la referencia primaria y reproducir el caso con fixture propio. |
| [Positive Technologies](https://github.com/PositiveTechnologies/seq2seq-web-attack-detection) — **DESCARTAR** runtime; **EXPERIMENTAR** concepto | La URL solicitada es una organización. Se examinó este proyecto concreto: autoencoder seq2seq para anomalías HTTP, notebook y dependencias antiguas; repositorio archivado. No se midió su detección. | El nuevo motor NEXUS usa reglas, no anomalías aprendidas. Solo justificar un experimento original contra baseline de reglas si hay datos etiquetados y costes medidos. | README refiere Python 2.7; GitHub identifica MIT. Fecha del último commit no fijada en esta consulta: no inventar mantenimiento reciente. Riesgo de dependencias obsoletas. Aceptación: dataset reservado, falsos positivos y latencia; no adoptar el runtime antiguo. |

Para los nueve repositorios nuevos accesibles, los commits fijados y fechas son:

| Repositorio | Commit de rama predeterminada | Fecha UTC | Licencia raíz observada |
|---|---|---|---|
| Tencent/AI-Infra-Guard | `7457acf1212d6b12093a662760969134426fcfa0` | 2026-10-26 | Apache-2.0 |
| alibaba/last-secure-code-benchmark | `d94492d4ad8ae899d354e45aa90d99f9e3002757` | 2026-10-21 | Apache-2.0 |
| microsoft/ACESEvals | `2c76fbdea415d6fa6a87bbad846686fefd6ee9bb` | 2026-09-15 | MIT |
| microsoft/SecRL | `a1b234f7a846133f08b7b5e1614c5c63b96f3b5c` | 2026-08-27 | MIT |
| microsoft/DefenderBench | `e431098e217ee6778f572df202785f1a705df167` | 2025-07-09 | Restrictiva, no comercial |
| microsoft/CyberBattleSim | `aebf185ab0a7515f983ee696cb7b71d3fdcd107f` | 2026-08-25 | MIT |
| google/osv-scanner | `ce7ef686680452c1c6415a29fd94592c3b06b30a` | 2026-10-25 | Apache-2.0 |
| seclaw-eval/seclaw-eval | `a3722ca6a754585c26a41e7b9f076fce0a19340f` | 2026-07-13 | Apache-2.0 |
| sunyinggang/LLM-Agent-Security-Survey | `78d49370fbf227f6f19e4179edf7c732e8ded0e0` | 2026-08-05 | No identificada |

Las URLs de licencia/README fijadas al SHA están en la evidencia JSON. El hallazgo
de DefenderBench procede del [texto efectivo](https://github.com/microsoft/DefenderBench/blob/e431098e217ee6778f572df202785f1a705df167/LICENSE),
que prevalece sobre describir el proyecto informalmente como abierto.

## Continuidad de los 18 candidatos anteriores

La ficha histórica enlazada contiene arquitectura, lenguajes, licencias,
dependencias, límites de evidencia y costes estimados para cada uno. Esta tabla
añade una decisión de extracción, sin volver a ejecutar investigación ni asumir
que la clasificación equivale a una integración realizada.

| Repositorio previo | Decisión y capacidad acotada | Estado de implementación en esta sesión |
|---|---|---|
| vxcontrol/pentagi | ADAPTAR contratos de ejecución/trazas | No importar stack; conservar gate existente |
| usestrix/strix | ADAPTAR verificación independiente de hallazgos | Sin conector externo nuevo |
| pursuingf/CyberEvolver | REIMPLEMENTAR conceptos de generaciones; no copiar código sin licencia | Runner original de selección acotada, no evolución de scaffolds |
| apache/caldera | EXPERIMENTAR escenarios de laboratorio | Sin implantes o ejecución de abilities |
| investigato/pentestgpt | ADAPTAR planificación por tareas, verificando procedencia del fork | Sin nuevo runtime |
| aliasrobotics/cai | DESCARTAR incorporación comercial sin derechos sobre añadidos | No integrado |
| 0x4m4/hexstrike-ai | REIMPLEMENTAR adaptadores mínimos con alcance verificable | No integrar catálogo completo |
| SigmaHQ/sigma | INTEGRAR formato/subconjunto; reglas con licencia por archivo | Compilador original y fixtures propios; no corpus embebido |
| elastic/detection-rules | DESCARTAR copia de reglas hasta resolver ELv2; estudiar método | Sin reglas copiadas |
| Infini-AI-Lab/SED | EXPERIMENTAR diagnóstico/selección de estrategias | No ejecutar optimizador externo |
| modelscope/AgentEvolver | EXPERIMENTAR transferencia; posponer entrenamiento RL | No GPU ni entrenamiento incorporados |
| SigmaHQ/pySigma | ADAPTAR como referencia diferencial, revisar LGPL | Aún sin dependencia Python ni comparación exhaustiva |
| mitre-attack/attack-stix-data | INTEGRAR taxonomía con avisos requeridos | Extraer tags de reglas; no demostrar cobertura por taxonomía |
| redcanaryco/atomic-red-team | EXPERIMENTAR escenarios revisados en laboratorio | Sin tests ofensivos externos ejecutados |
| langchain-ai/langgraph | REIMPLEMENTAR durabilidad según necesidad medida | Cancelación mejorada sobre coordinador Go existente |
| microsoft/autogen | DESCARTAR nueva dependencia por estado de mantenimiento/procedencia | No migrar runtime |
| google/oss-fuzz | ADAPTAR prácticas de fuzzing | Fuzzing local de Parse/Compile, 146.805 ejecuciones |
| wazuh/wazuh | EXPERIMENTAR conector desacoplado y licencia por componente | Biblioteca de normalización propia, sin conector Wazuh |

## Literatura primaria y decisiones experimentales

Las referencias siguientes distinguen resultados publicados de propuestas NEXUS.
No se reprodujeron sus benchmarks. Año, versión e identificador se verificaron en
las páginas primarias; no se asignan DOIs de congresos no comprobados.

| Publicación verificable | Método, resultado publicado y límite | Aplicación propuesta y recursos |
|---|---|---|
| **ExCyTIn-Bench: Evaluating LLM agents on Cyber Threat Investigation**, Yiran Wu, Mauricio Velazco, Andrew Zhao, Manuel Raúl Meléndez Luján, Srisuma Movva, Yogesh K Roy, Quang Nguyen, Roberto Rodriguez, Qingyun Wu, Michael Albada, Julia Kiseleva y Anand Mudgerikar. 2025; v3 2026-05-01. [arXiv:2507.14201v3](https://arxiv.org/abs/2507.14201v3) | Preguntas derivadas de grafos de investigación y logs de un tenant Azure controlado; evalúa agentes y ancla respuestas en evidencias. Los resultados dependen de esos datos/modelos y no acreditan investigación universal. Los autores indican aceptación ICML 2026. | Grader con respuestas ligadas a eventos; SQL de solo lectura y presupuesto de consultas/tokens. Coste NEXUS sin medir; no necesita RL para un primer baseline. |
| **Last Secure Code Benchmark: Is the Code You Vibe Secure?**, Tianyi Wu y otros 14 autores, 2026; v1 2026-10-10. [arXiv:2610.14766v1](https://arxiv.org/abs/2610.14766v1) | Pruebas ocultas funcionales y de seguridad sobre tareas derivadas de vulnerabilidades. La evaluación publicada muestra que funcionar no garantiza seguridad. No se trasladan sus tasas a NEXUS; preprint reciente y corpus finito. | Verificadores externos separados antes de aceptar variantes de código. Contenedores desechables, toolchains y recursos por tarea; todavía no implementado. |
| **CyberEvolver: Structured Self-Evolution for Cybersecurity Agents On the Fly**, Yihe Fan, Changyi Li, Lichen Xu, Xudong Pan, Jiarun Dai, Hong Geng y Min Yang. 2026; v2 2026-06-16. [arXiv:2605.26195v2](https://arxiv.org/abs/2605.26195v2) | Cuatro capas de scaffold, diagnóstico de trazas y búsqueda por poblaciones; experimentos en tareas de seguridad con modelos abiertos. No demuestra transferencia general ni mejora ilimitada; código sin licencia raíz identificada. | Inspiración para historia de generaciones; implementación actual solo selecciona catálogo fijo. Búsqueda futura multiplicaría evaluaciones por candidatos, casos y generaciones; presupuesto antes de ampliar. |
| **DefenderBench: A Toolkit for Evaluating Language Agents in Cybersecurity Environments**, Chiyu Zhang, Marc-Alexandre Cote, Michael Albada, Anush Sankaran, Jack W. Stokes, Tong Wang, Amir Abdi, William Blum y Muhammad Abdul-Mageed. 2025; v4 2025-10-14. [arXiv:2506.00739v4](https://arxiv.org/abs/2506.00739v4) | Evaluación por tareas defensivas; desempeño variable según tarea y modelo. Resultados científicos no conceden derechos comerciales sobre materiales. | Estudiar metodología y crear tareas originales. No incorporar sus materiales bajo la licencia actual; recursos NEXUS no medidos. |
| **Improving Methodologies for Agentic Evaluations Across Domains: Leakage of Sensitive Information, Fraud and Cybersecurity Threats**, Ee Wei Seah y otros 69 autores. 2026-01-22. [arXiv:2601.15679v1](https://arxiv.org/abs/2601.15679v1) | Evaluación conjunta de institutos de Singapur, Japón, Corea del Sur, Reino Unido, Francia y otras jurisdicciones; estudia comparabilidad metodológica, no un ranking nacional. Modelos abiertos/cerrados y varios dominios. | Comparar mismo presupuesto y condiciones, registrar diferencias de entorno; coste según modelos y repeticiones. La colaboración documenta procedencia sin inferir afiliaciones a partir de nombres. |
| **Analysis of LLMs Against Prompt Injection and Jailbreak Attacks**, Piyush Jaiswal, Aaditya Pratap, Shreyansh Saraswati, Harsh Kasyap y Somanath Tripathy. 2026-02-24. [arXiv:2602.22242v1](https://arxiv.org/abs/2602.22242v1) | Modelos abiertos y filtros durante inferencia; los autores observan variación entre modelos y fallos ante prompts largos. Dataset manual y modelos concretos limitan generalización. Afiliaciones indias verificadas en [texto primario](https://arxiv.org/html/2602.22242v1), incluidas NIT Trichy, IIT (BHU) e IIT Patna. | Probar instrucciones en telemetría/memoria y comprobar que no cambian permisos. Una clasificación LLM no sustituye gates independientes. Coste de inferencia según modelos; no se entrenaron ni ejecutaron esos modelos aquí. |

La auditoría científica anterior mantiene las referencias europeas y adicionales
en [SCIENTIFIC_RESEARCH.md](../research/SCIENTIFIC_RESEARCH.md). No se comprobó un
experimento KLara actual ni un resultado ruso reciente reproducible; se registra
esa ausencia en lugar de completar artificialmente una cuota geográfica.

## Decisión de incorporación y siguiente experimento

La implementación ya realizada usa ideas de semántica validada, trazabilidad,
evaluación separada y presupuestos; todo el código nuevo es original. La licencia
raíz de NEXUS y sus atribuciones existentes permanecen aplicables. No hubo nuevos
paquetes, código externo vendorizado ni materiales de DefenderBench incorporados.

Siguiente integración de alto valor: diagnóstico de dependencias por artefacto y
posterior ingesta Blue autenticada por tenant, antes de exponer endpoints o añadir
contenedores de benchmarks. Para evolución, preparar un holdout de familias
realmente diferentes y generadores declarativos acotados, manteniendo evaluador y
permisos fuera de su control. Las cifras del microbenchmark y del laboratorio
sintético actual son baselines locales, no una comparación ganada contra los
proyectos internacionales.
