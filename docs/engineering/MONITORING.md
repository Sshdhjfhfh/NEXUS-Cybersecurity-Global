# Monitorización y alertas

NEXUS vigila por sí mismo lo que suele romperse sin que nadie lo note y avisa
por email al equipo de operación. Lo único que no puede vigilar es su propia
caída: para eso hace falta un monitor externo (sección 3).

## 1. Vigilancia interna (backend)

Cada `NEXUS_OPS_INTERVAL` (5 min por defecto) el backend comprueba:

| Comprobación | Falla cuando |
|---|---|
| Base de datos | PostgreSQL no responde en 3 s. |
| Copias de seguridad | No hay ninguna copia, o la última es más antigua que `NEXUS_BACKUP_MAX_AGE_HOURS` (26 h). Lee el volumen de copias en solo lectura (`NEXUS_BACKUP_DIR`). |
| Espacio en disco | Queda menos de `NEXUS_DISK_MIN_FREE_PCT` (10 %) libre en datos o copias. |
| Errores del servidor | Hubo algún fallo interno (panic), o al menos 10 respuestas 5xx y un 5 % o más del tráfico desde la comprobación anterior. |
| Correo saliente | En producción (`NEXUS_PORTAL_BASE_URL` con `https://`) no hay `NEXUS_SMTP_HOST`; o hay emails que no se pudieron entregar en 24 h o que esperan desde hace más de 1 h. |
| Destino de las alertas | En producción no hay `NEXUS_OPS_EMAIL`. |

Avisos a `NEXUS_OPS_EMAIL`: un email cuando una comprobación empieza a fallar,
un recordatorio cada 24 h mientras siga fallando y otro cuando se resuelve. Se
envían directamente, sin pasar por la bandeja de salida (que puede ser lo que
falla). Todos los cambios quedan además en el log del backend como
`[ops-alert]`, también si el email no se puede enviar.

En la consola: **Sistema → Estado del servicio** (`/system/ops`) muestra el
estado actual y permite comprobar al momento. API (consola):
`GET /api/admin/ops/status`, `POST /api/admin/ops/check`.

## 2. Métricas

`/api/metrics` (formato Prometheus): peticiones, respuestas por clase (2xx–5xx),
duración, peticiones en curso, panics, peticiones descartadas por carga o por
límite de IP y tiempo en marcha. El proxy HTTPS la bloquea hacia Internet; se
consulta desde el propio servidor en `http://127.0.0.1:8787/api/metrics`.

## 3. Monitor externo (obligatorio en producción)

Si el servidor o el backend se caen, nada de lo anterior puede avisar. Da de
alta un monitor de disponibilidad externo (cualquier servicio de "uptime") que:

- consulte `https://<NEXUS_DOMAIN>/api/ready` cada 1–5 minutos;
- dé por caído cualquier código distinto de 200 (`/api/ready` responde 503 si
  la base de datos no está disponible);
- avise al mismo buzón que `NEXUS_OPS_EMAIL` (y, si se quiere, por SMS/chat).

Además, Docker reinicia los contenedores que terminan (`restart: unless-stopped`).

## 4. Qué hacer ante cada aviso

- **Base de datos**: `docker compose ps postgres`, `docker compose logs postgres`;
  espacio en disco del volumen `nexus_pgdata`.
- **Copias de seguridad**: `docker compose logs backup` (busca `ERROR`); probar a
  mano con `docker compose exec backup /bin/sh /usr/local/bin/db-backup.sh`.
  Ver `BACKUP_RESTORE.md`.
- **Espacio en disco**: `df -h`; borrar imágenes viejas (`docker image prune`);
  revisar `BACKUP_KEEP_DAYS`.
- **Errores del servidor**: `docker compose logs orchestrator | grep -E "\[panic\]|\[http\].* 5[0-9][0-9]"`;
  cada línea lleva un `id=` que el usuario ve como `X-Request-ID`.
- **Correo saliente**: revisar `NEXUS_SMTP_*`, credenciales y SPF/DKIM/DMARC del
  dominio; los envíos y sus errores: `GET /api/admin/portal/mail-outbox` (consola).
