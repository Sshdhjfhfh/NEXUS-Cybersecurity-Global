# NEXUS — Escaneo de seguridad (dependencias y secretos)

Resultados de los escáneres ejecutados sobre `nexus-real`, y su automatización en CI
(`.github/workflows/security.yml`: govulncheck, osv-scanner, gitleaks, npm audit en cada push/PR).

## Go — govulncheck + osv-scanner
**Hallazgo (corregido):**
- `GO-2026-6860` — complejidad cuadrática al unir nodos de texto en el parser HTML de
  `golang.org/x/net`. **Alcanzable** desde `core/tool/websearch.go` (`html.Parse`).
  **Fix aplicado:** `golang.org/x/net` **v0.61.0 → v0.62.0**.
- Tras el fix: **govulncheck = "0 vulnerabilidades; tu código no está afectado"**. Build + vet
  + tests OK; backend reconstruido y en marcha.
- Queda 1 aviso en un módulo requerido **no llamado por nuestro código** (no alcanzable).

## Web — npm audit
`npm audit` reporta **18 vulnerabilidades** (1 crítica, 11 altas, 5 moderadas, 1 baja), casi
todas **transitivas a través del paquete `shadcn`** (incluye una copia anidada antigua de
`zod` en `node_modules/shadcn/node_modules/zod`). No se arreglan a la fuerza porque
`npm audit fix --force` propone un cambio **rompedor** (`shadcn@1.0.0`).

**Recomendaciones (deliberadas, para no romper el build):**
1. Mover `shadcn` de `dependencies` a **`devDependencies`** (es una CLI de scaffolding, no se
   envía al navegador) y, si no se usa en runtime, considerar eliminarla.
2. Revisar el aviso de `zod`; fijar/actualizar a una versión no vulnerable y re-verificar el build.
3. Ejecutar `npm audit fix` (sin `--force`) en una rama y validar que la app compila y corre
   antes de mezclar.
4. El job `web-audit` de CI mantiene esto visible en cada cambio.

> Nota: muchas de estas advertencias no afectan al bundle de producción (son de la cadena de
> herramientas de desarrollo), pero se tratan con rigor: se documentan y se rastrean, no se
> declaran "resueltas" sin verificación.
