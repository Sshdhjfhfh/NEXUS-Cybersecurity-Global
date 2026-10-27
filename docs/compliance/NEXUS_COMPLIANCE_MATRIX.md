# NEXUS Cybersecurity Global — Matriz de Cumplimiento

> **Estado del documento:** borrador de trabajo. Ningún requisito está "validado
> legalmente" por el hecho de existir código. La validación jurídica la realiza un
> profesional habilitado una vez definida la modalidad comercial y los mercados.

**Leyenda de estado:** `NA` No aplicable (con justificación) · `ANÁLISIS` Pendiente de
análisis · `IMPL` Pendiente de implementación · `HECHO` Implementado (técnico) ·
`PROBADO` Implementado y probado · `LEGAL-OK` Validado legalmente.

| # | Norma / requisito | Jurisdicción | Aplicabilidad | Fuente oficial | Obligación (resumen) | Implementación técnica | Evidencia | Estado | Responsable | Riesgo / revisión legal |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | GDPR / RGPD | UE/EEE | ANÁLISIS (depende de si se ofrecen servicios a residentes UE) | Reglamento (UE) 2016/679 | Bases jurídicas, derechos ARCO+, info transparente, DPA con encargados, transferencias | Consentimiento cookies, formulario con consentimiento, config central `company.ts` | — | ANÁLISIS | Negocio + Legal | Alta |
| 2 | Ley de Protección de Datos Personales | Perú | ANÁLISIS (probable si hay entidad/clientes en Perú) | Ley N.º 29733 | Deberes de información, consentimiento, registro de bancos de datos, derechos | Formulario con consentimiento; falta registro de bancos ante APDP | — | ANÁLISIS | Negocio + Legal | Alta |
| 3 | Reglamento LPDP | Perú | ANÁLISIS | D.S. 016-2024-JUS | Desarrolla obligaciones de la Ley 29733 | Pendiente | — | ANÁLISIS | Legal | Alta |
| 4 | Cookies / ePrivacy | UE / global | IMPL | Directiva 2002/58/CE (ePrivacy) | Consentimiento previo para cookies no esenciales, revocable | Gestor de cookies (pendiente de implementar) | — | IMPL | Ingeniería | Media |
| 5 | Libro de Reclamaciones | Perú | ANÁLISIS (solo si venta a consumidores en Perú) | Ley 29571, Ley 32495 y reglamento vigente | Enlace visible, formato legal, correlativo, plazos | Pendiente; decidir modalidad comercial primero | — | ANÁLISIS | Negocio + Legal | Media |
| 6 | LGPD | Brasil | NA / ANÁLISIS (si se entra al mercado BR) | Lei 13.709/2018 | Similar a GDPR | Pendiente | — | ANÁLISIS | Legal | Baja |
| 7 | CCPA / CPRA | EE. UU. (California) | NA / ANÁLISIS | Cal. Civ. Code §1798.100+ | Derechos de consumidores, opt-out de venta | Pendiente | — | ANÁLISIS | Legal | Baja |
| 8 | UK GDPR | Reino Unido | NA / ANÁLISIS | DPA 2018 | Equivalente a GDPR | Pendiente | — | ANÁLISIS | Legal | Baja |
| 9 | PIPEDA | Canadá | NA / ANÁLISIS | PIPEDA 2000 | Consentimiento, acceso | Pendiente | — | ANÁLISIS | Legal | Baja |
| 10 | Autorización de pruebas ofensivas | Contractual | IMPL (parcial) | Buenas prácticas + contrato | Verificar autoridad sobre activos antes de testear | `lab/engagement` (scope gate server-side) ya aplica alcance firmado fuera del LLM | Código `lab/engagement/engagement.go` | HECHO (núcleo) | Ingeniería + Legal | Media |
| 11 | OWASP ASVS 5.0 | Seguridad web | ANÁLISIS | OWASP ASVS v5.0 | Controles verificables (authn, sesiones, CSRF, XSS, CSP, headers…) | Auth JWT + rate-limit leads + honeypot; faltan CSP/headers/MFA | — | ANÁLISIS | Ingeniería | Media |
| 12 | Accesibilidad | Global | IMPL | WCAG 2.2 AA | Teclado, foco, contraste, semántica, reduced-motion | Reduced-motion en animaciones; falta auditoría completa | — | IMPL | Ingeniería | Media |
| 13 | SEO internacional | Global | IMPL | — | hreflang, canonical, sitemap, robots, metadata, OG | Pendiente (requiere rutas /en /es) | — | IMPL | Ingeniería | Baja |

## Notas
- La **aplicabilidad real** de cada norma depende de: (a) país de constitución de la
  empresa, (b) dónde están los clientes/usuarios, (c) modalidad comercial (B2B SaaS,
  servicios profesionales, venta a consumidores). Esos datos van en
  `web/src/config/company.ts` y hoy están en `PENDIENTE`.
- Nada aquí debe comunicarse públicamente como certificación (SOC 2, ISO 27001, etc.)
  hasta obtenerla y auditarla.
