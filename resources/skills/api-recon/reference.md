# Referencia de inventario de API

Los scripts son plantillas: sus parámetros deben corresponder al objetivo autorizado. Guarda las copias y salidas dentro del directorio de trabajo de la tarea.

| Script | Función |
|---|---|
| `harvest_static.py` | Recoger HTML y recursos JavaScript para el inventario estático. |
| `spider_mpa.py` | Recorrer páginas del mismo sitio con límites de profundidad y cantidad. |
| `extract_route_map.py` | Extraer mapas de rutas del JavaScript descargado. |
| `build_perm_tree.py` | Reconstruir la estructura de navegación a partir del código; sus respuestas simuladas no prueban permisos del backend. |
| `runtime_harvest.js` | Observar solicitudes de Chromium y producir artefactos de navegación. |
| `preload.js` | Instrumentación del navegador para observación y simulaciones locales del frontend. |

## Configuración del navegador

El archivo JSON del runtime admite `baseUrl`, `outDir`, `chromium`, `proxy`, `routes`, `apiPattern`, `waitUntil`, `routeTimeout`, `waitMs`, `perRouteMs` y `headless`. Revisa los valores y argumentos documentados en la cabecera del script antes de iniciar una campaña. `CHROMIUM` selecciona el ejecutable; Nexus lo configura automáticamente.

El proxy puede obtenerse de `HTTP_PROXY` o `HTTPS_PROXY`. Respeta el proxy y los controles proporcionados por Nexus. No los elimines para eludir un bloqueo.

`forward: false` permite estudiar el frontend con respuestas simuladas; `forward: true` envía solicitudes al servidor y requiere autorización para esas acciones. `cookies` y `localStorage` solo deben contener material autorizado para la tarea. Las opciones `neutralize`, `stubs` y `loginUrlPattern` alteran el comportamiento local del frontend; registra expresamente su uso y nunca presentes esa simulación como autenticación real.

## Rastreo de páginas

`spider_mpa.py` recibe `base` y `outdir`, y permite `--max`, `--depth`, `--delay`, `--include`, `--exclude` y `--cookie`. Mantén una lista de exclusión de acciones con efectos; un método GET no garantiza que una ruta sea inocua.

## Informe

Incluye método, ruta, parámetros conocidos, fuente, evidencia, nivel de confianza, estado de autenticación y limitaciones. Redacta secretos. Las rutas inferidas y los registros simulados deben distinguirse de las peticiones reales.
