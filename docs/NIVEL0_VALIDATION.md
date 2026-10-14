# Validación de Nivel 0 — español

Fecha: 2026-10-14.

- `docker compose build orchestrator`: correcto.
- `docker compose up -d`: versión actualizada.
- PostgreSQL y orchestrator: health checks correctos.
- Catálogo español y copia pública idénticos: 5.325 entradas.
- 189 archivos TypeScript revisados: sin texto Han literal ni alias `tr`.
- `apply_translations.py --dry-run`: cero cambios pendientes, cero textos chinos.
- Pruebas de migración JSX, tipos, identificadores locales, extracción y dry run: correctas.
- Pruebas de inicialización, fallback offline, español predeterminado, RTL y aislamiento de campos de presentación: correctas.
- Chromium real, localhost:8787, sesión autenticada temporal: 20 rutas HTTP 200,
  sin texto chino visible, sin claves i18n sin resolver y sin errores de JavaScript.
- Comprobación visual adicional de la captura de inicio de sesión: nombre y logo nexus.

Rutas comprobadas: `/login`, `/dashboard`, `/chat`, `/function/tasks`,
`/function/assets`, `/function/findings`, `/function/commands`, `/function/traffic`,
`/function/llm-records`, `/function/sync`, `/function/workspace`, `/system/llm`,
`/system/agents`, `/system/mcp`, `/system/tools`, `/system/skills`, `/system/notify`,
`/system/intercept`, `/system/intercept/assets`, `/system/settings`.

Se localizaron también las etiquetas y descripciones integradas que entrega la
API. Los prompts, patrones de interceptación, identificadores internos y datos de
usuario desconocidos se conservan. No se modificó ningún módulo inmutable.
Los tokens y artefactos temporales del navegador no se incluyen en el repositorio.

Esta validación cubre el español y las rutas indicadas con los datos actuales;
no certifica otros nueve idiomas, contenido arbitrario del usuario ni todos los
estados posibles de cada diálogo.

## Referencia de tests Go preexistentes

La ejecución adicional de `go test ./...` detectó dos problemas anteriores a esta
migración (no se cambió código Go en Nivel 0): un nil HTTPClient en
`core/llm.TestDoStreamHonorsConfiguredRetries` y pruebas de integración del servidor
que requieren `NEXUS_PG_DSN`. El build completo Docker sí pasó. Estos problemas no
se presentan como tests aprobados ni se ocultan mediante skips.
