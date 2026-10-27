# NEXUS — Checklist de Lanzamiento del Website

Rastrea el avance hacia una revisión de lanzamiento (Directiva §20). Marca `[x]` al completar.

## Implementado y verificado (esta sesión)
- [x] Home de alta fidelidad fiel al diseño aprobado (hero, Capabilities, Platform
      Architecture, Trust Center, Request Demo, footer).
- [x] Imágenes del pack de marca integradas (hero, tarjetas, motor, secciones).
- [x] Fondos de vídeo en loop (hero + motor) con poster y `prefers-reduced-motion`.
- [x] Animaciones: contadores y reveal al scroll (reduced-motion aware).
- [x] Secciones: Inteligencia de Amenazas, La Plataforma, Evolution Lab, Industrias, About.
- [x] Formulario Request Demo **real** → guarda leads en PostgreSQL (verificado E2E).
- [x] Panel admin de leads con cambio de estado (`/system/leads`, requiere login).
- [x] Bilingüe ES/EN (toggle persistente).
- [x] **Menú móvil** (hamburguesa) funcional.
- [x] Config corporativa central `web/src/config/company.ts` (campos PENDIENTE).

## Pendiente de ingeniería (lo hago yo en próximos sprints)
- [ ] Rutas por idioma reales (`/en`, `/es`) con `hreflang`, canonical, sitemap, robots.
- [x] Subpáginas: Platform, Services, Research, About, Contact, Pricing. *(Early Access pendiente)*
- [ ] Páginas de capacidades (Red/Blue/Purple, Vuln Mgmt, Threat Detection, Agents, AI, Lab).
- [x] Gestor de cookies (necesarias/analítica/marketing) con registro de preferencias.
- [x] Páginas legales como **borrador "pendiente de revisión legal"**: Terms, Privacy,
      Cookies, AUP, Security, VDP publicadas en `/legal`. *(DPA, SLA, Subprocesadores, Aviso Legal: listados como pendientes)*
- [ ] Libro de Reclamaciones (SOLO si aplica la modalidad consumidor en Perú).
- [ ] Trust Center con clasificación de controles (Implementado/En desarrollo/Planificado).
- [ ] Endurecimiento de seguridad: CSP y cabeceras, revisión ASVS, (MFA si hay cuentas).
- [ ] Auditoría de accesibilidad WCAG 2.2 AA.
- [ ] Pruebas E2E (Playwright): enlaces, formularios, 404, idioma, cookies + visuales 1440/1280/768/390.
- [ ] Nitidez: re-encodear/regenerar imágenes de baja calidad (ver abajo).

## 🙋 LO QUE NECESITO DE TI (para no inventar nada)
### Datos legales/corporativos → van en `web/src/config/company.ts`
- [ ] Razón social registrada.
- [ ] País de constitución y número de registro / RUC.
- [ ] Dirección legal.
- [ ] Jurisdicción contractual (qué ley rige y tribunales).
- [ ] Emails reales: general, privacidad (DPO), seguridad, soporte.
- [ ] ¿Vende a **consumidores en Perú**? (decide si aplica Libro de Reclamaciones).
- [ ] Mercados objetivo reales (para decidir qué normas aplican: UE, Brasil, EE. UU., etc.).

### Decisiones de negocio
- [ ] Modalidad: ¿SaaS B2B, servicios profesionales de pentesting, o ambos?
- [ ] ¿Qué funciones son "disponibles" vs "Beta/Research" (para no prometer de más)?
- [ ] Canales de contacto auténticos (redes, teléfono/soporte) para el footer.

### Imágenes (nitidez) — regenerar en Codex a mayor calidad
Estas salieron con compresión fuerte y se ven blandas. Pídele a Codex la **misma imagen**
a mayor resolución/calidad (exportar **2560×1440**, WebP **calidad 90–95** o PNG, "maximum
detail, crisp UI text, sharp"):
- [ ] `platform-dashboard.webp` (la más importante: tiene texto/gráficas de UI).
- [ ] `industries.webp`
- [ ] `incident-response.webp`
Cuando las tengas en Descargas, las reemplazo.

### Revisión profesional
- [ ] Abogado/a para validar cada documento legal antes de publicarlo.
- [ ] (Opcional) Auditoría de seguridad externa antes de anunciar cualquier estándar.

## Regla de oro (cumplida)
Sin clientes, oficinas, certificaciones ni métricas inventadas. Lo experimental se marca
Research/Beta. Las imágenes conceptuales no se presentan como sedes/operaciones reales.
