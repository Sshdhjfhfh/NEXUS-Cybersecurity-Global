# Investigación científica y transferencia a NEXUS

Consulta: 2026-10-26. Referencias reales verificadas mediante editor, autores o
arXiv; no se reprodujeron los experimentos publicados. Esta es una revisión
priorizada, no una revisión sistemática exhaustiva. Se contrastaron principalmente
metadatos, resúmenes y documentación de artefactos; no se afirma lectura íntegra
de todos los artículos ni verificación de sus tablas experimentales.

Los resultados descritos a continuación pertenecen a sus autores. Las aplicaciones,
costes asintóticos aproximados y prioridades son hipótesis de ingeniería propias.
No se trasladan porcentajes externos a NEXUS. Un identificador arXiv no implica
revisión por pares. Cada enlace es la fuente de la ficha correspondiente.

## Agentes, memoria y planificación

| Referencia verificable | Método, evidencia y limitación | Aplicación propuesta y recursos estimados |
|---|---|---|
| **ReAct: Synergizing Reasoning and Acting in Language Models**. Shunyu Yao, Jeffrey Zhao, Dian Yu, Nan Du, Izhak Shafran, Karthik Narasimhan, Yuan Cao. 2022/2023. [arXiv:2210.03629](https://arxiv.org/abs/2210.03629). | Intercala razonamiento y acciones con feedback de entornos. Evaluación experimental en QA y entornos interactivos; no demuestra confinamiento ni exactitud de hallazgos de seguridad. | Estado explícito acción→observación→hipótesis en RED. Coste aproximadamente proporcional a pasos × tokens procesados; limitar pasos y tamaño de evidencia. Prioridad alta. |
| **PentestGPT: Evaluating and Harnessing Large Language Models for Automated Penetration Testing**. Gelei Deng, Yi Liu, Víctor Mayoral-Vilches, Peng Liu, Yuekang Li, Yuan Xu, Tianwei Zhang, Yang Liu, Martin Pinzger, Stefan Rass. USENIX Security 2024, pp. 847–864. [Editor](https://www.usenix.org/conference/usenixsecurity24/presentation/deng); preprint [2308.06782](https://arxiv.org/abs/2308.06782). | Tres módulos para gestionar contexto y tareas de pentesting; evaluación en objetivos/CTFs y artefacto reconocido por USENIX. El resultado del paper no se atribuye a un fork moderno ni al uso productivo general. | Comparar planificación y memoria de NEXUS con un baseline bajo igual presupuesto; máquinas aisladas y API LLM o modelo local. O(pasos × coste de inferencia + herramientas). Prioridad alta. |
| **Reflexion: Language Agents with Verbal Reinforcement Learning**. Noah Shinn, Federico Cassano, Edward Berman, Ashwin Gopinath, Karthik Narasimhan, Shunyu Yao. 2023. [arXiv:2303.11366](https://arxiv.org/abs/2303.11366). | Guarda reflexión textual episódica, sin ajustar pesos. Experimentos y ablaciones en decisión, código y razonamiento. El feedback simulado puede ser incorrecto; reflexión no equivale a verificación. | Notas de fallos redactadas y aprobadas por un evaluador, sin convertirlas en política. Coste crece con reintentos y memoria recuperada. Prioridad media tras procedencia y aislamiento. |
| **Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks**. Patrick Lewis, Ethan Perez, Aleksandra Piktus, Fabio Petroni, Vladimir Karpukhin, Naman Goyal, Heinrich Küttler, Mike Lewis, Wen-tau Yih, Tim Rocktäschel, Sebastian Riedel, Douwe Kiela. NeurIPS 2020. [arXiv:2005.11401](https://arxiv.org/abs/2005.11401). | Combina recuperación densa y generación, con evaluación de tareas de conocimiento. Mejor recuperación no certifica verdad ni permisos del documento recuperado. | RAG privado por organización con citas y evaluación de consultas distintas de títulos. Búsqueda exhaustiva O(Nd); índices aproximados con memoria/recall medidos; embeddings pueden ejecutarse por CPU/API/GPU. Prioridad media. |
| **Generative Agents: Interactive Simulacra of Human Behavior**. Joon Sung Park, Joseph C. O'Brien, Carrie J. Cai, Meredith Ringel Morris, Percy Liang, Michael S. Bernstein. 2023. [arXiv:2304.03442](https://arxiv.org/abs/2304.03442); DOI ACM 10.1145/3586183.3606763. | Memoria, reflexión y planificación para simulación social, con evaluación de plausibilidad y ablaciones. Plausibilidad humana no es seguridad ni éxito operativo. ACM devolvió 403; se verificó el resumen de autores en arXiv. | Separar memoria episódica de hechos verificados. Evitar trasladar métricas de comportamiento social a ciberseguridad. Recuperación lineal o indexada y varias llamadas de reflexión; prioridad de investigación baja. |
| **Voyager: An Open-Ended Embodied Agent with Large Language Models**. Guanzhi Wang, Yuqi Xie, Yunfan Jiang, Ajay Mandlekar, Chaowei Xiao, Yuke Zhu, Linxi Fan, Anima Anandkumar. 2023. [arXiv:2305.16291](https://arxiv.org/abs/2305.16291). | Currículo, biblioteca ejecutable de habilidades y feedback en Minecraft; evaluación y transferencia dentro del dominio. No prueba habilidades de seguridad ni que código generado sea seguro. | Skills originales con contrato, test y procedencia antes de promoción. Coste de exploración y almacenamiento proporcional a intentos/habilidades; runner aislado obligatorio. Prioridad media/baja. |

## Robustez, autoevolución y evaluación

| Referencia verificable | Método, evidencia y limitación | Aplicación propuesta y recursos estimados |
|---|---|---|
| **AgentDojo: A Dynamic Environment to Evaluate Prompt Injection Attacks and Defenses for LLM Agents**. Edoardo Debenedetti, Jie Zhang, Mislav Balunović, Luca Beurer-Kellner, Marc Fischer, Florian Tramèr. 2024. [arXiv:2406.13352](https://arxiv.org/abs/2406.13352). | Entornos/herramientas con tareas legítimas y contenido adversarial; evalúa utilidad y éxito de ataques. No cubre todas las herramientas ni sustituye un control de permisos. | Fixtures de outputs no fiables que intenten cambiar objetivo/alcance; verificar acciones y utilidad por separado. O(tareas × ataques × variantes × pasos). CPU más inferencia; prioridad alta. |
| **Automated Design of Agentic Systems**. Shengran Hu, Cong Lu, Jeff Clune. 2024/ICLR 2025. [arXiv:2408.08435](https://arxiv.org/abs/2408.08435). | Meta Agent Search genera agentes en código y usa archivo de diseños; experimentos en código, ciencia y matemáticas con transferencias. No demuestra mejora ilimitada ni seguridad de ejecución. | Variantes de workflows en sandbox, sin acceso al evaluador. O(generaciones × población × casos × coste de rollout); GPU opcional con APIs, coste de inferencia significativo. Posponer a fase 8. |
| **CyberEvolver: Structured Self-Evolution for Cybersecurity Agents On the Fly**. Yihe Fan, Changyi Li, Lichen Xu, Xudong Pan, Jiarun Dai, Hong Geng, Min Yang. 2026. [arXiv:2605.26195](https://arxiv.org/abs/2605.26195). | Cuatro capas, diagnóstico de trazas y beam search para revisar scaffolds. Evaluación en CTF, vulnerabilidades conocidas y pentesting con cuatro modelos abiertos. Evolución por objetivo; transferencia a nuevos objetivos y descubrimiento abierto no quedan demostrados. | Taxonomía propia de fallos y variantes; no importar código sin licencia raíz. O(generaciones × ancho × tareas × pasos). VMs/contenedores y varios rollouts; prioridad de estudio alta, integración acotada posterior. |
| **Self-Evolving Defense: Continual Security Policy Learning for LLM Agents**. Minh Nhat Le, Nisarga Gondi, Yibo Peng, Ronghao Ni, Limin Jia, Beidi Chen, Haizhong Zheng. 2026. [arXiv:2609.36603](https://arxiv.org/abs/2609.36603). | Extrae políticas de trayectorias dañinas y recupera memoria para episodios posteriores, sin actualizar pesos. Experimentos publicados con tres modelos y ocho benchmarks. Es un preprint reciente; juez y contaminación de memoria requieren evaluación propia. | Escribir memoria candidata al terminar episodio y verificar antes de activarla. O(episodios × coste de juez/generador + búsqueda en memoria). El repo usa servicios externos e incluye stack ML pesado; piloto con corpus sintético. Prioridad media/alta. |
| **AgentEvolver: Towards Efficient Self-Evolving Agent System**. Yunpeng Zhai, Shuchang Tao, Cheng Chen, Anni Zou, Ziqian Chen, Qingxu Fu, Shinji Mai, Li Yu, Jiaji Deng, Zouying Cao, Zhaoyang Liu, Bolin Ding, Jingren Zhou. 2025. [arXiv:2511.10395](https://arxiv.org/abs/2511.10395). | Generación de tareas, reutilización de experiencias y atribución de recompensas para RL. El resumen comunica experimentos preliminares; no asumir transferencia a NEXUS ni recursos exactos sin reproducir configuración. | Investigar atribución de errores/recompensa después de benchmarks ciegos. Coste de rollouts más entrenamiento/gradientes; stack Ray/Torch/vLLM y GPU según modelo. No presupuestar entrenamiento antes del piloto. |

## Detección e investigación defensiva

| Referencia verificable | Método, evidencia y limitación | Aplicación propuesta y recursos estimados |
|---|---|---|
| **KAIROS: Practical Intrusion Detection and Investigation using Whole-system Provenance**. Zijun Cheng, Qiujian Lv, Jinyuan Liang, Yan Wang, Degang Sun, Thomas Pasquier, Xueyuan Han. IEEE S&P 2024. [Autores/editorial](https://spg.cs.ubc.ca/publication/2024-sp/); [arXiv:2308.05034](https://arxiv.org/abs/2308.05034). | Encoder-decoder sobre grafos temporales de procedencia y reconstrucción de ataques; experimentos en datasets de investigación. Depende de sensores, datos representativos y umbrales; no hay evidencia aquí de rendimiento en telemetría NEXUS. | Primero grafo determinista de evento→activo→proceso→hallazgo, luego comparar modelo. Coste por aristas/ventana, dimensiones y pasos GNN; RAM para ventanas y GPU si el entrenamiento lo exige. Prioridad posterior a ingesta. |
| **UNICORN: Runtime Provenance-Based Detector for Advanced Persistent Threats**. Xueyuan Han, Thomas Pasquier, Adam Bates, James Mickens, Margo Seltzer. NDSS 2020. DOI **10.14722/ndss.2020.24046**. [arXiv:2001.01525](https://arxiv.org/abs/2001.01525). | Resumen incremental de grafos para comportamiento prolongado; evaluación experimental. Un sketch pierde información y la anomalía no identifica por sí sola una amenaza. | Investigar correlación de bajo consumo frente a heurísticas simples. Procesamiento por aristas y rondas de vecindario; tamaño de sketch fijado, sensor/corpus de procedencia necesarios. CPU primero; prioridad media. |
| **A comprehensive survey on intrusion detection systems with advances in machine learning, deep learning and emerging cybersecurity challenges**. Ali Hozouri, Abbas Mirzaei, Mehdi Effatparvar. 2025. DOI **10.1007/s44163-025-00578-1**. [Springer](https://link.springer.com/article/10.1007/s44163-025-00578-1). | Revisión de IDS, arquitecturas y datasets; no es una reproducción independiente de todos los resultados ni evidencia de un detector NEXUS. | Selección crítica de datos y comparación con reglas/estadística antes de DL. Recursos dependen del algoritmo; no hay complejidad única que atribuir al survey. Prioridad bibliográfica. |

## Referencias normativas, distintas de evidencia experimental

| Referencia | Autores/año e identificador | Aplicación/límite |
|---|---|---|
| [SSDF 1.1](https://csrc.nist.gov/pubs/sp/800/218/final) | Murugiah Souppaya, Karen Scarfone, Donna Dodson, 2022; DOI 10.6028/NIST.SP.800-218 | Desarrollo, procedencia, verificación y respuesta a vulnerabilidades. Guía, no benchmark ni certificación de NEXUS. |
| [Incident Response Recommendations and Considerations for Cybersecurity Risk Management](https://csrc.nist.gov/pubs/sp/800/61/r3/final) | Alexander Nelson, Sanjay Rekhi, Murugiah Souppaya, Karen Scarfone, 2025; DOI 10.6028/NIST.SP.800-61r3 | Investigación y recuperación dentro de CSF 2.0; no confundir automatización de contención con autorización. |
| [MITRE ATT&CK Data & Tools](https://attack.mitre.org/resources/attack-data-and-tools/) | MITRE, recurso vivo consultado 2026-10-26; dataset a fijar por versión/SHA | Taxonomía y STIX para PURPLE; mapear no significa emular/detectar. Coste de importar/indexar JSON, sin GPU requerida. |
| [OWASP Web Security Testing Guide](https://owasp.org/projects/web-security-testing-guide) | OWASP WSTG Team, versión publicada 4.2; desarrollo 5.0 indicado por la fuente | Metodología y trazabilidad de casos web, con licencias de contenido propias. No es evaluador autónomo. |

## Cobertura de búsqueda y límites de acceso

Se buscaron/consultaron IEEE Xplore, ACM DL, USENIX, NDSS, arXiv, Google Scholar,
Springer, ScienceDirect, NIST, MITRE y OWASP. IEEE devolvió fichas de surveys y
se confirmó KAIROS con la página de los autores. ACM y ScienceDirect devolvieron
403 al abrir los artículos; Google Scholar no fue accesible por la herramienta.
No se presentan contenidos bloqueados como leídos.
El candidato de ScienceDirect
[Deep learning methods in network intrusion detection: A survey and an objective comparison](https://www.sciencedirect.com/science/article/pii/S1084804520302411)
queda pendiente de metadatos/texto completo antes de incluir conclusiones detalladas.

## Decisión de investigación

Primer experimento: conformidad Sigma con oráculo independiente y fixtures que
incluyan negativos; después robustez a contenido de herramientas no fiable,
recuperación de memoria y correlación determinista. Los grafos aprendidos, RL y
generación de código quedan detrás de datos, aislamiento y presupuesto.
Ninguna referencia prueba hoy autoevolución recursiva general de NEXUS.
La evaluación propuesta está en [RECURSIVE_IMPROVEMENT_RESEARCH.md](RECURSIVE_IMPROVEMENT_RESEARCH.md).
