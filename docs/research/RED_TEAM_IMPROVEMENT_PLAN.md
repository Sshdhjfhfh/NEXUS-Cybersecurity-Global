# Plan de mejora RED

2026-10-26. Diagnóstico y propuestas; solo se implementó el arreglo de carga Sigma
descrito en [VALIDATION_MATRIX.md](VALIDATION_MATRIX.md).

## Punto de partida

`agents/red` contiene sesiones LLM reales, planificación, workers, registro y
retest de hallazgos. `core/tool` aporta archivos, shell, web y MCP; `core/plan`
gestiona modo de planificación/aprobación, no un planificador de ataques.
`scan.Scanner` ejecuta httpx/nuclei/katana con un `lab.Authorizer`, pero las pruebas
actuales demuestran principalmente parseo y rechazo antes de ejecutar. Las
entradas son strings de destino; `lab/security` exige IP privada literal. Hace
falta un contrato canónico de activos para conectar ambos con garantías.
`adversarial_rt` simula y `ctf_engine.Start` no resuelve desafíos. `zero_day` es un
laboratorio sintético de fuzzing, no un motor de descubrimiento ya demostrado.

## Entregas ordenadas

| Prioridad | Cambio | Dependencias | Aceptación verificable |
|---|---|---|---|
| P0 | Contrato de alcance: organización/tarea, activos canónicos, puertos/protocolos, expiración y permisos por acción | Inventario y adaptadores existentes | Pruebas negativas para URL ambigua, IPv6, DNS cambiante y redirección; cero conexiones fuera del permiso registrado. |
| P0 | Runner de herramientas aislado con egreso impuesto por host, límites CPU/RAM/PIDs/tiempo y captura de salida | Política independiente del LLM | Fixture de herramienta que intenta otro destino: bloqueo por red y evidencia del rechazo, aunque el proceso ignore instrucciones. |
| P1 | Catálogo fijado de binarios y plantillas, digest y procedencia; presupuestos de solicitudes y exclusión de operaciones destructivas | Runner | Mismo inventario produce mismo contrato; plantilla no aprobada no se ejecuta; sin actualizaciones implícitas. |
| P1 | Adaptadores con resultado estructurado y separación `observed / hypothesis / verified / rejected` | Evidencia | Salida corrupta/truncada produce error explícito; hallazgo no se marca verificado sin reproducción independiente. |
| P1 | Priorización basada en activos, exposición, precondiciones y confianza de evidencia | Modelo de activos | Ranking reproducible ante datos faltantes; no inventar CVE/CVSS; guardar versión y motivos. |
| P2 | SAST/DAST sobre copia y objetivo propios; correlación CWE y metodología OWASP | Runner y licencias | Casos vulnerables y corregidos, reportes reproducibles y negativos representativos. |
| P2 | Informe y retest: petición normalizada, respuesta redactada, hash, versión de herramienta, alcance y corrección | Almacén de evidencia | Un validador distinto reproduce en laboratorio y el caso corregido deja de fallar. |

## Razonamiento sobre resultados

Mantener un estado explícito de hipótesis y evidencias con dependencias. Después
de cada herramienta: validar esquema y procedencia; identificar qué hipótesis
apoya o contradice; seleccionar la siguiente observación con mayor utilidad dentro
del presupuesto; detener caminos sin evidencia o con autorización vencida.
El LLM propone acciones y explicaciones; un ejecutor valida el contrato y un
verificador diferente establece si el hallazgo está demostrado. No ejecutar texto
libre del modelo como si fuera una autorización. Conservar la API de findings y
añadir campos opcionales mediante adaptadores, sin reemplazar el runtime heredado.

Medir por escenarios: hallazgos reproducibles/emitidos, falsos positivos, cobertura
de activos autorizados, peticiones, tiempo y coste por hallazgo confirmado.
Comparar herramientas sin LLM, agente actual y variante con el mismo presupuesto.
Una subida de alertas no cuenta como mejora de precisión.

## Investigación aplicable y límites

[PentestGPT, USENIX 2024](https://www.usenix.org/conference/usenixsecurity24/presentation/deng)
apoya separar contexto, planificación y ejecución.
[Strix](https://github.com/usestrix/strix) y [PentAGI](https://github.com/vxcontrol/pentagi)
aportan patrones de herramientas y sandbox a estudiar; sus resultados no validan
NEXUS. Usar [OWASP WSTG versionada](https://owasp.org/projects/web-security-testing-guide)
para etiquetar casos, conservando sus licencias si se reproduce contenido.
No portar implantes, persistencia ni propagación de plataformas de emulación.

Riesgos: una plantilla puede realizar acciones intrusivas; autorizar el destino
inicial no controla lo que el subproceso contacte después; capturas pueden contener
datos sensibles. Reversión futura: desactivar el adaptador nuevo y volver al
catálogo anterior; nunca desactivar el gate o el registro para recuperar rendimiento.
Pendiente inmediato: prueba local adversarial de egreso y de redirecciones antes
de ampliar las funciones del scanner.
