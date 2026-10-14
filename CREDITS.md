# Créditos y Atribuciones

Nexus IA v6.0 es un proyecto derivado. Este documento lista los proyectos upstream de los que se ha absorbido código, con la atribución correspondiente según AGPL-3.0.

---

## ARTEX

- **Repositorio**: https://github.com/mhtsec/ARTEX
- **Autor original**: mhtsec
- **Licencia**: AGPL-3.0
- **Descripción**: AI 自主渗透测试系统 (Sistema de pentesting autónomo con IA). Ganador del concurso "Agent+" de Baidu.
- **Uso en Nexus**: Base de `orchestrator/`, `graph_rag/`, `llm_gateway/`, `ethical_boundary/`, `governance/`, `lab/`, `evidence/`, `observability/`, `selfimprove/`, `agents/red/`, `swarm/`, `web/`.
- **LOC absorbidas**: ~93.282 (Go) + ~56.088 (Next.js)

## norma

- **Repositorio**: https://github.com/Autumn-27/norma
- **Autor original**: Autumn-27
- **Versión absorbida**: v0.4.3
- **Licencia**: AGPL-3.0 (restaurada desde proxy de Go)
- **Descripción**: SDK de agentes LLM con sistema de tools, memoria, permisos, harness.
- **Uso en Nexus**: Base de `core/` (agent, tool, harness, compaction, skill, memory, transcript, mcp, subagent, coordinator, permission, hook, plan).
- **LOC absorbidas**: ~16.000 (Go)

---

## Licencia del proyecto

Nexus IA v6.0 se distribuye bajo **GNU Affero General Public License v3.0 (AGPL-3.0)**, heredada de ARTEX.

Cualquier modificación o derivado de Nexus debe:
1. Mantener la licencia AGPL-3.0.
2. Publicar el código fuente si se ofrece como servicio de red.
3. Mantener esta atribución a ARTEX y norma.

Ver `LICENSE` para los términos completos.

---

## Uso permitido

Nexus IA v6.0 **SOLO** puede usarse en:
- Laboratorios aislados.
- Investigación académica.
- Análisis de código fuente.
- Simulaciones internas con autorización explícita.

**PROHIBIDO**:
- Uso contra sistemas reales sin autorización.
- Uso en producción.
- Uso ofensivo fuera del laboratorio.
- Cualquier actividad ilegal.