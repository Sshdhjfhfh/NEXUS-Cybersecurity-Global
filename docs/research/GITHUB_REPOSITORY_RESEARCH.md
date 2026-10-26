# Investigación de repositorios GitHub

Fecha de consulta: 2026-10-26. Se verificaron los once repositorios iniciales y siete adicionales.

## Método y límites

Se consultaron API pública (metadatos, último commit de rama predeterminada y árbol recursivo), README, licencias y manifiestos en el SHA observado. No se clonaron ni ejecutaron proyectos externos. La evidencia está en [metadatos](evidence/github-metadata.json) y [manifiesto con hashes](evidence/github-source-manifest.json). Los árboles permiten localizar pruebas; los recuentos por nombres no miden cobertura ni calidad. Las afirmaciones de funcionalidad describen código/documentación disponible, no una reproducción experimental.

Las búsquedas adicionales por API fueron `agentic cybersecurity in:name,description` y `security agent benchmark in:name,description`, ordenadas por actualización. Sus primeros resultados tuvieron pertinencia desigual; se preservaron en [github-search.json](evidence/github-search.json), sin elegirlos por popularidad. Los siete proyectos añadidos cubren necesidades concretas de reglas, datos, escenarios, coordinación, fuzzing y telemetría.

`pushed_at` puede reflejar actividad en otra rama y no acredita mantenimiento de la rama principal. Se registra por separado la fecha del commit fijado. Todas las estimaciones de integración son semanas-persona para un alcance pequeño, propuestas por esta auditoría; no son plazos publicados ni promesas.

## Matriz de existencia y actividad

| Repositorio | Último push UTC | Commit predeterminado: fecha UTC y evidencia | Mantenimiento observado | Licencia API |
|---|---|---|---|---|
| [vxcontrol/pentagi](https://github.com/vxcontrol/pentagi) | 2026-10-22T23:26:13Z | [cffed9485aa1](https://github.com/vxcontrol/pentagi/commit/cffed9485aa1cca70762cd08a57ac62a80406258) · 2026-10-22T23:20:46Z | No archivado en API; actividad no garantiza soporte | MIT |
| [usestrix/strix](https://github.com/usestrix/strix) | 2026-10-26T15:18:39Z | [200a501f15a2](https://github.com/usestrix/strix/commit/200a501f15a24a4152f92897a6b72af238ee69ef) · 2026-10-24T22:59:21Z | No archivado en API; actividad no garantiza soporte | Apache-2.0 |
| [pursuingf/CyberEvolver](https://github.com/pursuingf/CyberEvolver) | 2026-10-26T09:11:45Z | [948c5ad03b7f](https://github.com/pursuingf/CyberEvolver/commit/948c5ad03b7f7b9ebcd8d798845d48fd12267506) · 2026-06-04T07:32:53Z | No archivado en API; actividad no garantiza soporte | no identificada |
| [apache/caldera](https://github.com/apache/caldera) | 2026-10-12T19:16:34Z | [cb8d6a8160e0](https://github.com/apache/caldera/commit/cb8d6a8160e0f4990af323a7c071d5e8482b42c7) · 2026-08-27T00:46:57Z | No archivado en API; actividad no garantiza soporte | Apache-2.0 |
| [investigato/pentestgpt](https://github.com/investigato/pentestgpt) | 2026-07-14T12:58:31Z | [e8b1bb77d1ac](https://github.com/investigato/pentestgpt/commit/e8b1bb77d1ac00329675cec3b060aba971ec1ac8) · 2026-07-14T12:58:30Z | No archivado en API; actividad no garantiza soporte | MIT |
| [aliasrobotics/cai](https://github.com/aliasrobotics/cai) | 2026-08-22T13:05:58Z | [6dc79257777f](https://github.com/aliasrobotics/cai/commit/6dc79257777f5f1c9500b4d2319935d34a47412e) · 2026-08-22T13:02:17Z | Archivado en API | NOASSERTION |
| [0x4m4/hexstrike-ai](https://github.com/0x4m4/hexstrike-ai) | 2026-08-03T14:33:09Z | [d689933ff579](https://github.com/0x4m4/hexstrike-ai/commit/d689933ff579d839c676c82b231f8e98326c5f04) · 2026-08-03T14:33:01Z | No archivado en API; actividad no garantiza soporte | MIT |
| [SigmaHQ/sigma](https://github.com/SigmaHQ/sigma) | 2026-10-21T03:04:45Z | [05d1031f537e](https://github.com/SigmaHQ/sigma/commit/05d1031f537e830b76da53bae2400c5814c89165) · 2026-10-21T03:04:44Z | No archivado en API; actividad no garantiza soporte | NOASSERTION |
| [elastic/detection-rules](https://github.com/elastic/detection-rules) | 2026-10-26T15:08:09Z | [52ef7a06c71a](https://github.com/elastic/detection-rules/commit/52ef7a06c71a513f4209844c025ec8eb341871eb) · 2026-10-24T03:17:38Z | No archivado en API; actividad no garantiza soporte | NOASSERTION |
| [Infini-AI-Lab/SED](https://github.com/Infini-AI-Lab/SED) | 2026-09-15T11:24:52Z | [7cda0834511d](https://github.com/Infini-AI-Lab/SED/commit/7cda0834511df386b9e1c57632dfa537e1d0fdab) · 2026-09-15T11:24:50Z | No archivado en API; actividad no garantiza soporte | MIT |
| [modelscope/AgentEvolver](https://github.com/modelscope/AgentEvolver) | 2026-04-01T08:47:19Z | [a5a8db8689d6](https://github.com/modelscope/AgentEvolver/commit/a5a8db8689d6493b028107dcb4c27415441581dc) · 2026-03-28T04:58:20Z | No archivado en API; actividad no garantiza soporte | Apache-2.0 |
| [SigmaHQ/pySigma](https://github.com/SigmaHQ/pySigma) | 2026-10-18T11:33:14Z | [54e4077532d1](https://github.com/SigmaHQ/pySigma/commit/54e4077532d1ae654ed565b5a61b75e76e8f725a) · 2026-10-11T21:09:30Z | No archivado en API; actividad no garantiza soporte | LGPL-2.1 |
| [mitre-attack/attack-stix-data](https://github.com/mitre-attack/attack-stix-data) | 2026-10-13T19:11:18Z | [c2494b29dd26](https://github.com/mitre-attack/attack-stix-data/commit/c2494b29dd268c46a5d0ea0990553e4327cd2253) · 2026-10-13T19:11:14Z | No archivado en API; actividad no garantiza soporte | NOASSERTION |
| [redcanaryco/atomic-red-team](https://github.com/redcanaryco/atomic-red-team) | 2026-10-26T11:10:47Z | [1edbe50b8f49](https://github.com/redcanaryco/atomic-red-team/commit/1edbe50b8f49560110e6cf85161c1ef75a5732b7) · 2026-10-21T00:15:52Z | No archivado en API; actividad no garantiza soporte | MIT |
| [langchain-ai/langgraph](https://github.com/langchain-ai/langgraph) | 2026-10-26T15:02:11Z | [1adeafeaf32e](https://github.com/langchain-ai/langgraph/commit/1adeafeaf32ea5162790845ba82f51aba9899bea) · 2026-10-26T15:02:08Z | No archivado en API; actividad no garantiza soporte | MIT |
| [microsoft/autogen](https://github.com/microsoft/autogen) | 2026-10-16T16:14:37Z | [004749e6b710](https://github.com/microsoft/autogen/commit/004749e6b7102c8fd317b9b1499ce5807223c278) · 2026-10-16T16:14:36Z | API no archivado; README anuncia archivado/no mantenido: discrepancia | CC-BY-4.0 |
| [google/oss-fuzz](https://github.com/google/oss-fuzz) | 2026-10-26T11:24:15Z | [a6d41d14b529](https://github.com/google/oss-fuzz/commit/a6d41d14b5293cf2d06e5572813c5995e06c09c3) · 2026-10-26T11:24:11Z | No archivado en API; actividad no garantiza soporte | Apache-2.0 |
| [wazuh/wazuh](https://github.com/wazuh/wazuh) | 2026-10-26T15:05:16Z | [dcb722fae3e8](https://github.com/wazuh/wazuh/commit/dcb722fae3e85719eaaa006a45751c77819469e5) · 2026-10-22T11:31:22Z | No archivado en API; actividad no garantiza soporte | NOASSERTION |

## Matriz de utilidad y prioridades

Orden propuesto para NEXUS: 1 Sigma, 2 pySigma, 3 ATT&CK STIX, 4 Atomic Red Team, 5 PentAGI, 6 Strix, 7 OSS-Fuzz, 8 LangGraph, 9 SED, 10 Caldera. Es un ranking de utilidad condicionada a seguridad/licencia, no de capacidad global. AgentEvolver queda después por coste; CyberEvolver requiere permiso para reutilizar código; CAI no se incorpora comercialmente sin licencia.

En las fichas, «NEXUS aporta» identifica un diseño local aprovechable. **No se midió que NEXUS realice ninguna función mejor que esos proyectos bajo un benchmark comparable.** La conclusión honesta es una ventaja de integración potencial, no superioridad demostrada.

## vxcontrol/pentagi

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT; conservar copyright/licencia; herramientas, imágenes y edición empresarial se evalúan aparte. |
| Arquitectura/lenguajes | Go/TS; agentes especializados, Docker, PostgreSQL/pgvector y grafo opcional. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Sesiones, herramientas, memoria e informes; estudiar ejecución supervisada y trazas. |
| Dependencias | Go backend, UI React y servicios de datos; revisar backend/go.mod y frontend/package.json. |
| Pruebas/calidad | Backend con pruebas y workflows E2E; 362 archivos con patrón de test en árbol; no ejecutados. La calidad de código completa queda sin auditar. |
| Escalabilidad | Separación en servicios; escalado horizontal exige validar estado y runner, no inferirlo del README. No se midió carga externa. |
| Compatibilidad/coste | Afinidad Go alta; 4–8 semanas-persona para un adaptador acotado, no portar plataforma. |
| Qué conservar de NEXUS | NEXUS aporta gate de laboratorio/ANS-NHI y evaluación acotada de memoria; no se midió superioridad. |
| Recomendación | Estudiar y adaptar contratos de ejecución; no copiar el stack completo. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/vxcontrol/pentagi/commit/cffed9485aa1cca70762cd08a57ac62a80406258), [árbol fijado](https://github.com/vxcontrol/pentagi/tree/cffed9485aa1cca70762cd08a57ac62a80406258), [LICENSE](https://raw.githubusercontent.com/vxcontrol/pentagi/cffed9485aa1cca70762cd08a57ac62a80406258/LICENSE), [NOTICE](https://raw.githubusercontent.com/vxcontrol/pentagi/cffed9485aa1cca70762cd08a57ac62a80406258/NOTICE), [README.md](https://raw.githubusercontent.com/vxcontrol/pentagi/cffed9485aa1cca70762cd08a57ac62a80406258/README.md).

## usestrix/strix

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Apache-2.0; conservar licencia/avisos y cambios, revisar NOTICE y dependencias. |
| Arquitectura/lenguajes | Python, Go y TS; agentes, sandbox, herramientas, TUI y visor. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Pruebas web/código con validación y coordinación; ampliar revisión de hallazgos verificables. |
| Dependencias | openai-agents/LiteLLM, Docker, Pydantic; Python >=3.12 en manifiesto. |
| Pruebas/calidad | 130 archivos con patrón de test y CI de Python/TUI/visor; ejecución ajena no comprobada. La calidad de código completa queda sin auditar. |
| Escalabilidad | Aislamiento por ejecución; coste de modelos y contenedores crece con paralelismo. No se midió carga externa. |
| Compatibilidad/coste | Adaptador por proceso/API; 4–8 semanas-persona; evitar segundo runtime opaco. |
| Qué conservar de NEXUS | NEXUS conserva backend Go y gate explícito de laboratorio; comparar ambos en mismos casos. |
| Recomendación | Estudiar y adaptar verificación; no convertir sus claims de precisión en evidencia propia. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/usestrix/strix/commit/200a501f15a24a4152f92897a6b72af238ee69ef), [árbol fijado](https://github.com/usestrix/strix/tree/200a501f15a24a4152f92897a6b72af238ee69ef), [LICENSE](https://raw.githubusercontent.com/usestrix/strix/200a501f15a24a4152f92897a6b72af238ee69ef/LICENSE), [README.md](https://raw.githubusercontent.com/usestrix/strix/200a501f15a24a4152f92897a6b72af238ee69ef/README.md).

## pursuingf/CyberEvolver

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Sin licencia raíz identificada; componentes de benchmarks conservan licencias propias. No incorporar código sin permiso. |
| Arquitectura/lenguajes | Python; capas evolutivas, diagnóstico de trazas y búsqueda por poblaciones. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Revisiones del scaffold tras fallos; investigación de variantes, no servicio empresarial. |
| Dependencias | Manifiestos raíz y numerosos entornos/benchmarks; procedencia fragmentada. |
| Pruebas/calidad | 604 archivos con patrón de test, muchos vendorizados en datasets/dependencias: no prueban cobertura del motor. La calidad de código completa queda sin auditar. |
| Escalabilidad | Rollouts multiplicados por población y generaciones; coste alto y sandbox imprescindible. No se midió carga externa. |
| Compatibilidad/coste | Ideas transferibles a evolve; integración de código bloqueada por licencia, experimento original 4–8 semanas-persona. |
| Qué conservar de NEXUS | NEXUS tiene catálogo acotado, política y reversión; no demuestra su mejora de scaffolds. |
| Recomendación | Estudiar publicación y reimplementar ideas originales con evaluación independiente. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/pursuingf/CyberEvolver/commit/948c5ad03b7f7b9ebcd8d798845d48fd12267506), [árbol fijado](https://github.com/pursuingf/CyberEvolver/tree/948c5ad03b7f7b9ebcd8d798845d48fd12267506), [README.md](https://raw.githubusercontent.com/pursuingf/CyberEvolver/948c5ad03b7f7b9ebcd8d798845d48fd12267506/README.md).

## apache/caldera

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Apache-2.0 y NOTICE actuales; plugins/submódulos requieren análisis separado. |
| Arquitectura/lenguajes | Python/JS, plugins y agentes; planificación de emulación ATT&CK. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Operaciones, abilities y resultados; referencia para escenarios PURPLE. |
| Dependencias | aiohttp, cryptography, plugins; lista fijada requirements.txt. |
| Pruebas/calidad | 56 archivos con patrón de test y workflows de calidad/seguridad; no ejecutados. La calidad de código completa queda sin auditar. |
| Escalabilidad | Controlador y agentes; validar número de operaciones/hosts y aislamiento. No se midió carga externa. |
| Compatibilidad/coste | Conector de laboratorio con subconjunto aprobado, 4–8 semanas-persona. |
| Qué conservar de NEXUS | NEXUS aporta análisis pasivo y memoria evaluada; Caldera aporta emulación más madura, sin comparación experimental. |
| Recomendación | Integrar solo API/escenarios permitidos en laboratorio; no portar implantes. |
| Vulnerabilidades conocidas/limitaciones de comprobación | CVE-2025-27364 histórico: README pide v5.1.0+; no se afirma que el SHA revisado sea vulnerable. Revisar también plugins. [Aviso](https://github.com/apache/caldera). |

Fuentes inspeccionadas: [commit](https://github.com/apache/caldera/commit/cb8d6a8160e0f4990af323a7c071d5e8482b42c7), [árbol fijado](https://github.com/apache/caldera/tree/cb8d6a8160e0f4990af323a7c071d5e8482b42c7), [LICENSE](https://raw.githubusercontent.com/apache/caldera/cb8d6a8160e0f4990af323a7c071d5e8482b42c7/LICENSE), [NOTICE](https://raw.githubusercontent.com/apache/caldera/cb8d6a8160e0f4990af323a7c071d5e8482b42c7/NOTICE), [README.md](https://raw.githubusercontent.com/apache/caldera/cb8d6a8160e0f4990af323a7c071d5e8482b42c7/README.md).

## investigato/pentestgpt

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT según LICENSE.md; preservar autores y comprobar origen del fork. |
| Arquitectura/lenguajes | Python; pipeline por fases, persistencia, backends de agentes y modo legacy. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Gestión de sesiones/árbol de tareas; no atribuir automáticamente al fork resultados del paper original. |
| Dependencias | SDKs/modelos, CLI y herramientas de laboratorio; pyproject.toml. |
| Pruebas/calidad | 27 archivos con patrón de test y fixtures; incluye pruebas live opt-in; no ejecutados. La calidad de código completa queda sin auditar. |
| Escalabilidad | Sesiones por ejecución; no se verificó aislamiento multiempresa. No se midió carga externa. |
| Compatibilidad/coste | Referencia de planificación; 2–4 semanas-persona para un ensayo, no otro controlador productivo. |
| Qué conservar de NEXUS | NEXUS posee API/UI, evidencias y flujos de retest integrados; superioridad no medida. |
| Recomendación | Estudiar y adaptar conceptos; verificar procedencia con upstream citado por autores. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/investigato/pentestgpt/commit/e8b1bb77d1ac00329675cec3b060aba971ec1ac8), [árbol fijado](https://github.com/investigato/pentestgpt/tree/e8b1bb77d1ac00329675cec3b060aba971ec1ac8), [LICENSE.md](https://raw.githubusercontent.com/investigato/pentestgpt/e8b1bb77d1ac00329675cec3b060aba971ec1ac8/LICENSE.md), [README.md](https://raw.githubusercontent.com/investigato/pentestgpt/e8b1bb77d1ac00329675cec3b060aba971ec1ac8/README.md).

## aliasrobotics/cai

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT en src/cai/agents; añadidos Alias Robotics de investigación no comercial, requieren licencia comercial. |
| Arquitectura/lenguajes | Python; agentes, herramientas y caibench; repositorio archivado. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Evaluaciones y coordinación histórica; no confiar en mantenimiento futuro. |
| Dependencias | SDKs de agentes, herramientas y entornos heterogéneos; inspección parcial de pyproject. |
| Pruebas/calidad | Tests/workflows presentes; historial consolidado según README y sin validación ejecutada. La calidad de código completa queda sin auditar. |
| Escalabilidad | No seleccionar como runtime mantenido; coste de mantener fork recaería en NEXUS. No se midió carga externa. |
| Compatibilidad/coste | Integración comercial descartada sin permiso; estudio limitado de publicaciones. |
| Qué conservar de NEXUS | NEXUS tiene evolución acotada propia y ruta de mantenimiento local; no comparación de capacidad ofensiva. |
| Recomendación | Estudiar; descartar incorporación comercial de añadidos sin licencia. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/aliasrobotics/cai/commit/6dc79257777f5f1c9500b4d2319935d34a47412e), [árbol fijado](https://github.com/aliasrobotics/cai/tree/6dc79257777f5f1c9500b4d2319935d34a47412e), [LICENSE](https://raw.githubusercontent.com/aliasrobotics/cai/6dc79257777f5f1c9500b4d2319935d34a47412e/LICENSE), [LICENSE-MIT](https://raw.githubusercontent.com/aliasrobotics/cai/6dc79257777f5f1c9500b4d2319935d34a47412e/LICENSE-MIT), [README.md](https://raw.githubusercontent.com/aliasrobotics/cai/6dc79257777f5f1c9500b4d2319935d34a47412e/README.md).

## 0x4m4/hexstrike-ai

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT del wrapper; licencias de cada herramienta siguen aplicando. |
| Arquitectura/lenguajes | Python; servidor Flask y puente MCP a herramientas. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Catálogo de adaptadores; exposición amplia de ejecución no equivale a autonomía fiable. |
| Dependencias | Flask, requests, psutil, MCP, Selenium y ejecutables externos. |
| Pruebas/calidad | No se encontraron archivos de pruebas/CI por el patrón usado; calidad/seguridad de ejecución no auditadas. La calidad de código completa queda sin auditar. |
| Escalabilidad | Procesos externos y servidor concentran límites; medir colas y cuotas. No se midió carga externa. |
| Compatibilidad/coste | Portar pocos adaptadores con schema propio, 2–4 semanas-persona; riesgo elevado si se expone entero. |
| Qué conservar de NEXUS | NEXUS aporta gate, evidencia y autorización por trabajo en laboratorio; no presume contención integral. |
| Recomendación | Estudiar interfaces; reimplementar adaptadores mínimos, no integrar servidor completo. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/0x4m4/hexstrike-ai/commit/d689933ff579d839c676c82b231f8e98326c5f04), [árbol fijado](https://github.com/0x4m4/hexstrike-ai/tree/d689933ff579d839c676c82b231f8e98326c5f04), [LICENSE](https://raw.githubusercontent.com/0x4m4/hexstrike-ai/d689933ff579d839c676c82b231f8e98326c5f04/LICENSE), [README.md](https://raw.githubusercontent.com/0x4m4/hexstrike-ai/d689933ff579d839c676c82b231f8e98326c5f04/README.md).

## SigmaHQ/sigma

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Formato/especificación públicos; reglas DRL-1.1 con atribución en reglas y mensajes de coincidencia. |
| Arquitectura/lenguajes | YAML y Python de validación; catálogo de detecciones por fuente/técnica. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Reglas y metadatos; parsing exitoso no garantiza detección correcta. |
| Dependencias | Esquemas/validadores, taxonomía y herramientas Sigma; no es un SIEM. |
| Pruebas/calidad | tests/test_rules.py, tests/test_logsource.py y workflows; no reproducidos aquí. La calidad de código completa queda sin auditar. |
| Escalabilidad | Indexar fuentes y compilar reglas; el catálogo no prescribe motor de ejecución. No se midió carga externa. |
| Compatibilidad/coste | Alta utilidad; 2–4 semanas-persona para corpus fijado y validación, tras corregir semántica. |
| Qué conservar de NEXUS | NEXUS ejecuta observaciones y controla gates; Sigma ofrece catálogo más amplio, no evaluado localmente. |
| Recomendación | Integrar subconjunto versionado con autor/licencia y prueba por regla. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/SigmaHQ/sigma/commit/05d1031f537e830b76da53bae2400c5814c89165), [árbol fijado](https://github.com/SigmaHQ/sigma/tree/05d1031f537e830b76da53bae2400c5814c89165), [LICENSE](https://raw.githubusercontent.com/SigmaHQ/sigma/05d1031f537e830b76da53bae2400c5814c89165/LICENSE), [README.md](https://raw.githubusercontent.com/SigmaHQ/sigma/05d1031f537e830b76da53bae2400c5814c89165/README.md).

## elastic/detection-rules

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Elastic License 2.0 en reglas/código; preservar NOTICE y examinar límites de servicio gestionado. |
| Arquitectura/lenguajes | Python/TOML; reglas y utilidades KQL/EQL/ESQL de Elastic Security. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Validación, metadatos y ciclo de reglas; referencia de ingeniería de detección. |
| Dependencias | Python >=3.12, cliente Elasticsearch, parsers y librerías de esquema. |
| Pruebas/calidad | 35 archivos con patrón de test y numerosos workflows; no ejecutados. La calidad de código completa queda sin auditar. |
| Escalabilidad | Ejecución depende del backend Elastic; no extrapolar capacidad del repositorio al motor NEXUS. No se midió carga externa. |
| Compatibilidad/coste | Estudio del proceso; conector de SIEM 4–8 semanas-persona sujeto a licencia. |
| Qué conservar de NEXUS | NEXUS puede mantener reglas propias y control de laboratorio; no sustituye detección de Elastic. |
| Recomendación | Estudiar metodología; no copiar reglas/código ELv2 al núcleo sin revisión comercial. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/elastic/detection-rules/commit/52ef7a06c71a513f4209844c025ec8eb341871eb), [árbol fijado](https://github.com/elastic/detection-rules/tree/52ef7a06c71a513f4209844c025ec8eb341871eb), [LICENSE.txt](https://raw.githubusercontent.com/elastic/detection-rules/52ef7a06c71a513f4209844c025ec8eb341871eb/LICENSE.txt), [NOTICE.txt](https://raw.githubusercontent.com/elastic/detection-rules/52ef7a06c71a513f4209844c025ec8eb341871eb/NOTICE.txt), [README.md](https://raw.githubusercontent.com/elastic/detection-rules/52ef7a06c71a513f4209844c025ec8eb341871eb/README.md).

## Infini-AI-Lab/SED

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT raíz; benchmarks/dependencias externos con otras licencias, incluido OpenRT AGPL. |
| Arquitectura/lenguajes | Python; recuperación de políticas, memoria episódica y juez tras episodio. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Defensa adaptable sin cambio de pesos; estudiar cuarentena y escritura posterior al episodio. |
| Dependencias | Torch/Transformers/sentence-transformers, servicios Fireworks y benchmarks externos. |
| Pruebas/calidad | No encontrados tests unitarios por el patrón usado; sí runners de evaluación; claims no reproducidos. La calidad de código completa queda sin auditar. |
| Escalabilidad | Coste de chat, embeddings y juez por episodio; independencia del juez no viene garantizada. No se midió carga externa. |
| Compatibilidad/coste | Adaptar idea a memoria/políticas originales, 3–6 semanas-persona, sin dar autoridad al LLM. |
| Qué conservar de NEXUS | NEXUS dispone de promoción determinista de estrategias; SED investiga adaptación defensiva distinta. |
| Recomendación | Estudiar y adaptar memoria validada; no usar políticas generadas como control del host. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/Infini-AI-Lab/SED/commit/7cda0834511df386b9e1c57632dfa537e1d0fdab), [árbol fijado](https://github.com/Infini-AI-Lab/SED/tree/7cda0834511df386b9e1c57632dfa537e1d0fdab), [LICENSE](https://raw.githubusercontent.com/Infini-AI-Lab/SED/7cda0834511df386b9e1c57632dfa537e1d0fdab/LICENSE), [README.md](https://raw.githubusercontent.com/Infini-AI-Lab/SED/7cda0834511df386b9e1c57632dfa537e1d0fdab/README.md).

## modelscope/AgentEvolver

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Apache-2.0; revisar entornos, submódulos, datasets y checkpoints por separado. |
| Arquitectura/lenguajes | Python; servicios de entorno, experiencia y entrenamiento RL. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Auto-generación de tareas, navegación por experiencias y atribución de recompensa. |
| Dependencias | Ray, Torch, Accelerate, vLLM y stack CUDA según entorno; coste operativo alto. |
| Pruebas/calidad | 16 archivos con patrón de test, varios scripts de entorno/juegos; no es garantía de integración. La calidad de código completa queda sin auditar. |
| Escalabilidad | Entrenamiento distribuido y rollouts requieren presupuesto GPU y operación de servicios. No se midió carga externa. |
| Compatibilidad/coste | Baja prioridad actual; piloto 6–12 semanas-persona más cómputo, condicionado a benchmark independiente. |
| Qué conservar de NEXUS | NEXUS evalúa estrategias sin GPU ni entrenamiento; no equivale al aprendizaje RL del proyecto. |
| Recomendación | Estudiar; posponer integración de entrenamiento. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/modelscope/AgentEvolver/commit/a5a8db8689d6493b028107dcb4c27415441581dc), [árbol fijado](https://github.com/modelscope/AgentEvolver/tree/a5a8db8689d6493b028107dcb4c27415441581dc), [LICENSE](https://raw.githubusercontent.com/modelscope/AgentEvolver/a5a8db8689d6493b028107dcb4c27415441581dc/LICENSE), [README.md](https://raw.githubusercontent.com/modelscope/AgentEvolver/a5a8db8689d6493b028107dcb4c27415441581dc/README.md).

## SigmaHQ/pySigma

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | LGPL-2.1; conservar licencia, cambios y obligaciones sobre biblioteca; backends con licencias propias. |
| Arquitectura/lenguajes | Python; modelo, procesamiento y conversión de reglas Sigma. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Referencia independiente de sintaxis y semántica; no es detector SIEM completo. |
| Dependencias | pyproject.toml y plugins/backends; no incorporar automáticamente todos. |
| Pruebas/calidad | Suite de tests, backend/pipelines de prueba y CI; no ejecutados. La calidad de código completa queda sin auditar. |
| Escalabilidad | Conversión offline y caché; medir coste por corpus/versiones. No se midió carga externa. |
| Compatibilidad/coste | Usar proceso separado como oráculo de conformidad; 1–3 semanas-persona. |
| Qué conservar de NEXUS | NEXUS tiene matcher Go local; pySigma reduce riesgo de semántica inventada. |
| Recomendación | Integrar en validación de laboratorio, sujeto a obligaciones LGPL. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/SigmaHQ/pySigma/commit/54e4077532d1ae654ed565b5a61b75e76e8f725a), [árbol fijado](https://github.com/SigmaHQ/pySigma/tree/54e4077532d1ae654ed565b5a61b75e76e8f725a), [LICENSE](https://raw.githubusercontent.com/SigmaHQ/pySigma/54e4077532d1ae654ed565b5a61b75e76e8f725a/LICENSE), [README.md](https://raw.githubusercontent.com/SigmaHQ/pySigma/54e4077532d1ae654ed565b5a61b75e76e8f725a/README.md).

## mitre-attack/attack-stix-data

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Licencia propia MITRE permite uso comercial con copyright/licencia; respetar marcas, no llamarla MIT. |
| Arquitectura/lenguajes | JSON STIX y utilidades Python; versiones de conocimiento ATT&CK. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Técnicas, relaciones y metadatos; no aporta cobertura demostrada. |
| Dependencias | JSON/STIX; util/requirements.txt para utilidades. |
| Pruebas/calidad | Datos/versiones; no se verificó suite completa de validación. La calidad de código completa queda sin auditar. |
| Escalabilidad | Importación offline e índices por ID/versión; no enviar bundle completo al LLM. No se midió carga externa. |
| Compatibilidad/coste | Compatibilidad alta; 1–2 semanas-persona para ingestión versionada original. |
| Qué conservar de NEXUS | NEXUS añade evidencias de ejecución; ATT&CK es conocimiento, no verificador. |
| Recomendación | Integrar datos fijados con licencia; enlazar técnicas con evidencia. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/mitre-attack/attack-stix-data/commit/c2494b29dd268c46a5d0ea0990553e4327cd2253), [árbol fijado](https://github.com/mitre-attack/attack-stix-data/tree/c2494b29dd268c46a5d0ea0990553e4327cd2253), [LICENSE.txt](https://raw.githubusercontent.com/mitre-attack/attack-stix-data/c2494b29dd268c46a5d0ea0990553e4327cd2253/LICENSE.txt), [README.md](https://raw.githubusercontent.com/mitre-attack/attack-stix-data/c2494b29dd268c46a5d0ea0990553e4327cd2253/README.md).

## redcanaryco/atomic-red-team

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT principal; revisar archivos auxiliares y payloads por licencia. |
| Arquitectura/lenguajes | YAML, PowerShell, shell y Python de validación; pruebas por técnica. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Escenarios discretos y requisitos/limpieza para PURPLE. |
| Dependencias | Ejecutores, dependencias y SO varían por atomic; no descargar dependencias implícitamente. |
| Pruebas/calidad | Validación YAML/atomics y tests de esquemas; no se ejecutaron atomics. La calidad de código completa queda sin auditar. |
| Escalabilidad | Aislar por VM/escenario; concurrencia depende de sensores y efectos. No se midió carga externa. |
| Compatibilidad/coste | Subconjunto inocuo de laboratorio, 2–4 semanas-persona; no importar ejecución indiscriminada. |
| Qué conservar de NEXUS | NEXUS aporta pipeline/gates y evaluación de estrategias; carece de circuito completo de validación. |
| Recomendación | Integrar escenarios seleccionados tras inspección y autorización individual de su clase de acción. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/redcanaryco/atomic-red-team/commit/1edbe50b8f49560110e6cf85161c1ef75a5732b7), [árbol fijado](https://github.com/redcanaryco/atomic-red-team/tree/1edbe50b8f49560110e6cf85161c1ef75a5732b7), [LICENSE.txt](https://raw.githubusercontent.com/redcanaryco/atomic-red-team/1edbe50b8f49560110e6cf85161c1ef75a5732b7/LICENSE.txt), [README.md](https://raw.githubusercontent.com/redcanaryco/atomic-red-team/1edbe50b8f49560110e6cf85161c1ef75a5732b7/README.md).

## langchain-ai/langgraph

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | MIT del core; servicios comerciales y checkpointers se evalúan aparte. |
| Arquitectura/lenguajes | Python/TS; grafos, checkpoints y workflows con estado. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Reanudación y memoria; estudiar persistencia e interrupciones. |
| Dependencias | Paquetes por librería y adaptadores de almacenamiento; dependencias variables. |
| Pruebas/calidad | Amplia suite de tests y CI; no ejecutados ni comparados con NEXUS. La calidad de código completa queda sin auditar. |
| Escalabilidad | Persistencia/resume requieren idempotencia y backends; no equivalen a seguridad de efectos. No se midió carga externa. |
| Compatibilidad/coste | Reimplementar patrones sobre Go/PostgreSQL existente, 3–6 semanas-persona. |
| Qué conservar de NEXUS | NEXUS evita segundo runtime para coordinación; no demuestra igual resiliencia. |
| Recomendación | Estudiar checkpoints/outbox; no migrar todo el orquestador. |
| Vulnerabilidades conocidas/limitaciones de comprobación | GHSA-9rwj-6rc7-p77c: langgraph-checkpoint-sqlite <3.0.1, parche 3.0.1. No extrapolar al core actual. [Fuente](https://github.com/langchain-ai/langgraph/security/advisories/GHSA-9rwj-6rc7-p77c). |

Fuentes inspeccionadas: [commit](https://github.com/langchain-ai/langgraph/commit/1adeafeaf32ea5162790845ba82f51aba9899bea), [árbol fijado](https://github.com/langchain-ai/langgraph/tree/1adeafeaf32ea5162790845ba82f51aba9899bea), [LICENSE](https://raw.githubusercontent.com/langchain-ai/langgraph/1adeafeaf32ea5162790845ba82f51aba9899bea/LICENSE), [README.md](https://raw.githubusercontent.com/langchain-ai/langgraph/1adeafeaf32ea5162790845ba82f51aba9899bea/README.md).

## microsoft/autogen

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Código MIT en LICENSE-CODE, documentación CC-BY-4.0; API muestra solo la segunda. |
| Arquitectura/lenguajes | Python/.NET; Core, AgentChat y extensiones. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Patrones de mensajes, equipos y evaluación; README anuncia archivo y sucesor MAF. |
| Dependencias | Extensiones/proveedores y entornos de ejecución variados. |
| Pruebas/calidad | Tests Python/.NET presentes; no ejecutados; mantenimiento contradictorio entre API y README. La calidad de código completa queda sin auditar. |
| Escalabilidad | No adoptar sin aclarar soporte; estudiar interfaces y serialización. No se midió carga externa. |
| Compatibilidad/coste | Referencia histórica; nueva integración descartada por ahora. |
| Qué conservar de NEXUS | NEXUS mantiene API y core Go; resiliencia relativa no medida. |
| Recomendación | Estudiar; posponer integración y evaluar sucesor en otra investigación. |
| Vulnerabilidades conocidas/limitaciones de comprobación | GHSL-2024-025/026 sobre workflows: antecedente de cadena de suministro, no prueba de fallo del SHA actual. [Fuente](https://securitylab.github.com/advisories/GHSL-2024-025_GHSL-2024-026_AutoGen/). |

Fuentes inspeccionadas: [commit](https://github.com/microsoft/autogen/commit/004749e6b7102c8fd317b9b1499ce5807223c278), [árbol fijado](https://github.com/microsoft/autogen/tree/004749e6b7102c8fd317b9b1499ce5807223c278), [LICENSE](https://raw.githubusercontent.com/microsoft/autogen/004749e6b7102c8fd317b9b1499ce5807223c278/LICENSE), [LICENSE-CODE](https://raw.githubusercontent.com/microsoft/autogen/004749e6b7102c8fd317b9b1499ce5807223c278/LICENSE-CODE), [README.md](https://raw.githubusercontent.com/microsoft/autogen/004749e6b7102c8fd317b9b1499ce5807223c278/README.md).

## google/oss-fuzz

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Apache-2.0 de infraestructura; cada proyecto/harness/imagen conserva su licencia. |
| Arquitectura/lenguajes | Python/shell/C/C++; builds instrumentados, fuzzing y reproducción. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Patrones de harness, corpus, sanitizers, minimización y regresiones. |
| Dependencias | Docker, compiladores/sanitizers y runners; requisitos por proyecto. |
| Pruebas/calidad | Infraestructura y pruebas presentes; no ejecutadas en esta sesión. La calidad de código completa queda sin auditar. |
| Escalabilidad | Workers por target, cuotas CPU/disco/tiempo y costes de corpus. No se midió carga externa. |
| Compatibilidad/coste | Adaptar proceso al laboratorio zero_day, 2–4 semanas-persona. |
| Qué conservar de NEXUS | NEXUS aporta contexto de agentes; OSS-Fuzz aporta operación de fuzzing más amplia. |
| Recomendación | Estudiar e integrar prácticas y harnesses propios. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/google/oss-fuzz/commit/a6d41d14b5293cf2d06e5572813c5995e06c09c3), [árbol fijado](https://github.com/google/oss-fuzz/tree/a6d41d14b5293cf2d06e5572813c5995e06c09c3), [LICENSE](https://raw.githubusercontent.com/google/oss-fuzz/a6d41d14b5293cf2d06e5572813c5995e06c09c3/LICENSE), [README.md](https://raw.githubusercontent.com/google/oss-fuzz/a6d41d14b5293cf2d06e5572813c5995e06c09c3/README.md).

## wazuh/wazuh

| Dimensión | Evaluación |
|---|---|
| Licencia y obligaciones | Mixta: engine AGPL-3.0-or-later, bibliotecas indicadas duales GPL-2.0-only OR AGPL, resto GPLv2; avisos adicionales. |
| Arquitectura/lenguajes | C/C++/Python; agentes, manager, reglas y componentes de SIEM/XDR. |
| Funcionalidad disponible y oportunidad sobre NEXUS | Telemetría y detección endpoint; estudiar conector, no reemplazar BLUE por copia. |
| Dependencias | Bibliotecas nativas, indexación y servicios; revisar licencias de cada componente. |
| Pruebas/calidad | Pruebas de agentes/servidor y CI; no ejecutadas. La calidad de código completa queda sin auditar. |
| Escalabilidad | Manager/agentes e índices; dimensionar por eventos/retención y aislamiento. No se midió carga externa. |
| Compatibilidad/coste | Conector a instalación del operador, 4–8 semanas-persona, tras revisión de redistribución. |
| Qué conservar de NEXUS | NEXUS añade laboratorio/evaluación acotada; no tiene cobertura de sensores equivalente. |
| Recomendación | Estudiar/integrar protocolo bajo revisión; no copiar engine al núcleo. |
| Vulnerabilidades conocidas/limitaciones de comprobación | No se realizó un escaneo completo del grafo upstream. Sin conclusión sobre ausencia de vulnerabilidades; exigir SBOM/OSV y análisis de alcanzabilidad del SHA antes de integrar. |

Fuentes inspeccionadas: [commit](https://github.com/wazuh/wazuh/commit/dcb722fae3e85719eaaa006a45751c77819469e5), [árbol fijado](https://github.com/wazuh/wazuh/tree/dcb722fae3e85719eaaa006a45751c77819469e5), [LICENSE](https://raw.githubusercontent.com/wazuh/wazuh/dcb722fae3e85719eaaa006a45751c77819469e5/LICENSE), [README.md](https://raw.githubusercontent.com/wazuh/wazuh/dcb722fae3e85719eaaa006a45751c77819469e5/README.md).

## Decisiones

Mantener interfaces originales de NEXUS. Importar datos/reglas versionados bajo licencia y adaptar herramientas por procesos/API con política externa. No acumular runtimes de agentes ni ejecutar los scripts de instalación de los proyectos para evaluarlos. La siguiente comparación experimental debe usar tareas propias, iguales presupuestos y un verificador independiente; hasta entonces las ventajas observadas son oportunidades.

La [matriz de validación](VALIDATION_MATRIX.md) distingue las pruebas propias ejecutadas de todas las capacidades externas aún no reproducidas. Los resultados OSV del proyecto NEXUS se encuentran en [LICENSE_COMPLIANCE.md](LICENSE_COMPLIANCE.md); no deben confundirse con un escaneo de estos dieciocho repositorios.
