# Estado real del proyecto — 2026-10-14

Este estado distingue implementación existente de integración y validación.
Los recuentos de líneas de código no equivalen a porcentaje de producto terminado.

| Área | Estado | Falta para terminar |
|---|---|---|
| Interfaz española | Publicada y verificada en 20 pantallas; catálogo de 5.325 entradas | Cobertura continua para nuevos estados y contenido; no se certifican aquí los otros nueve idiomas. |
| Docker y backend heredado | Compilan y arrancan con dos servicios healthy | Validación operativa de todas las integraciones nuevas. |
| BLUE, predicción y swarm | Base funcional con tests y race detector | Conectar trabajadores, evidencias, autorización real y persistencia con el runtime. |
| Laboratorios | Proxy/enrich existentes y planes de escenarios aislados | Configuración e imágenes probadas de DVWA, Metasploitable, VulnHub y AD; adaptadores de VM cuando correspondan. |
| Seguridad y gobernanza existentes | Paquetes implementados con tests | Conexión y validación de extremo a extremo en el proceso principal. |
| Protección pendiente | `opsec` y `circuit_breaker` ausentes; `watchdog`, `llmjacking_guard`, `oracle_defense` vacíos | No se modifican ni sustituyen: están sujetos a la regla de inmutabilidad del usuario. |
| Nivel 3 | `evolve`, `memory`, `knowledge_bus` vacíos | Implementación, integración y pruebas. |
| Nivel 4 | `graph_rag`, `vckg`, `zero_day`, `symbolic_ai`, `ai_scientist`, `cti_feed`, `digital_twin`, `xai`, `federated` vacíos | Implementar módulos propios; aprovechar los grafos heredados existentes en otros paquetes. |
| Validación final | Compilación y suite actual correctas con entorno de pruebas aislado | Medir recuperación, confinamiento, autenticación por acción, precisión y rendimiento exigidos por la spec. |

La siguiente entrega debe cerrar la integración de Nivel 1–2 antes de avanzar a
los niveles superiores. No se declara NEXUS v6.0 completo ni se afirma un
porcentaje o fecha de terminación sin una planificación de los pendientes.

Detalles: [Nivel 0](NIVEL0_VALIDATION.md), [Nivel 1–2](NIVEL12_STATUS.md) y
[especificación](NEXUS_v6_SPEC.md).
