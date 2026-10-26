# NEXUS — Arranque local verificado (FASE 0)

**Fecha:** 2026-10-26 · **Entorno:** WSL Ubuntu, sin Docker/root.

## Qué arranca y en qué puerto

| Servicio | Puerto | Estado verificado |
|---|---|---|
| PostgreSQL 18.6 (userspace, micromamba) | **55432** | ✅ acepta conexiones; base `nexus` con 41 tasks |
| Backend Go (`cmd/nexus`) | **8787** | ✅ `GET /api/health` → `{"ok":true,"service":"nexus-ia","version":"dev"}` |
| Frontend Next.js 16 (Turbopack) | **3000** | ✅ responde; proxy `/api`→`:8787` OK |

> Se usa Postgres en **55432** (no 5432) porque en WSL2 `127.0.0.1:5432`
> suele estar tomado por un proceso del lado de Windows (localhost se reenvía).

## Procedimiento reproducible (no destructivo)

```bash
bash scripts/dev-local-up.sh
```

Luego abrir **http://localhost:3000** en el navegador. En primer uso muestra
**"Inicializar contraseña"** (`/setup`) en español; el selector de idioma
soporta ES (default) e EN, más otros.

Parar todo:

```bash
pkill -f next-server; pkill -f nexus-dev/nexus-server
/home/saul/.nexus-dev/pg/bin/pg_ctl -D /home/saul/.nexus-dev/pgdata -m fast stop
```

## Evidencia (comprobado en esta sesión)

- Backend: `reloaded 41 task(s) from PG`, `listening on :8787`
  (ver `/home/saul/.nexus-dev/serve.log`).
- Frontend: `Next.js 16.4.1 (Turbopack) … Ready` (`web_dev.log`).
- Proxy: `curl http://127.0.0.1:3000/api/health` → 200.
- Navegador: carga `/setup` (consola NEXUS, ES). Captura tomada en sesión.

## Qué es esto y qué falta (honesto)

- Lo que corre es la **consola interna** de NEXUS (operador): dashboard,
  agents, tasks, findings, assets, logs, intercept, etc. Requiere setup/login.
- **No existe aún la web comercial pública** (FASE 1): la ruta `(external)`
  hoy solo redirige a la consola. Ese es el siguiente incremento.
- El motor LLM está **inactivo** (`no LLM provider configured`), esperado.

## Siguiente (FASE 1)

Construir la **web pública internacional** (Home + páginas, EN/ES, navy/cyan)
como nuevo grupo de rutas en el mismo proyecto Next, y la **captación real de
leads** (API Go + persistencia + vista admin). Criterio: abrir la web, cambiar
idioma, navegar y registrar una solicitud que aparezca en el panel.
