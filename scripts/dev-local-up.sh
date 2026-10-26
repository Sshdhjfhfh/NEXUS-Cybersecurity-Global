#!/usr/bin/env bash
# dev-local-up.sh — arranque local reproducible de NEXUS sin Docker (WSL).
#
# Levanta: PostgreSQL userspace (micromamba) + backend Go (:8787) + frontend
# Next.js (:3000, con proxy /api -> :8787). NO destruye datos. Idempotente.
#
# Requisitos ya presentes en este entorno de laboratorio:
#   - PostgreSQL 18.6 en $DEV/pg (data en $DEV/pgdata, superusuario rol "nexus")
#   - binario backend $DEV/nexus-server  (o se compila desde ./cmd/nexus)
#   - web/node_modules instalado (next dev)
#
# Nota: usamos el puerto 55432 para Postgres porque 127.0.0.1:5432 suele estar
# ocupado por un proceso del lado de Windows (WSL2 reenvía localhost).
set -u
DEV="${NEXUS_DEV_HOME:-/home/saul/.nexus-dev}"
REPO="$(cd "$(dirname "$0")/.." && pwd)"
PGPORT="${NEXUS_PG_PORT:-55432}"
PGBIN="$DEV/pg/bin"
export LD_LIBRARY_PATH="$DEV/pg/lib:${LD_LIBRARY_PATH:-}"
export PGCONNECT_TIMEOUT=5
DSN="postgres://nexus@127.0.0.1:${PGPORT}/nexus?sslmode=disable"

echo "▶ 1/3 PostgreSQL (:$PGPORT)"
if ! "$PGBIN/pg_isready" -h 127.0.0.1 -p "$PGPORT" -q 2>/dev/null; then
  rm -f "$DEV/pgdata/postmaster.pid" 2>/dev/null
  "$PGBIN/pg_ctl" -D "$DEV/pgdata" -l "$DEV/pg.log" -o "-p $PGPORT" -w -t 60 start \
    && echo "  started" || { echo "  ERROR: no arrancó PG (ver $DEV/pg.log)"; exit 1; }
else
  echo "  ya estaba corriendo"
fi

echo "▶ 2/3 Backend Go (:8787)"
if [ ! -x "$DEV/nexus-server" ]; then
  echo "  compilando backend..."; ( cd "$REPO" && go build -o "$DEV/nexus-server" ./cmd/nexus ) || exit 1
fi
pkill -f 'nexus-dev/nexus-server' 2>/dev/null; sleep 1
( cd "$REPO" && NEXUS_PG_DSN="$DSN" setsid "$DEV/nexus-server" -data "$DEV/data" -proxy "" \
    >"$DEV/serve.log" 2>&1 & )
for i in $(seq 1 25); do
  [ "$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8787/api/health 2>/dev/null)" = "200" ] \
    && { echo "  health 200 (${i}s)"; break; }; sleep 1
done

echo "▶ 3/3 Frontend Next.js (:3000)"
pkill -f 'next dev' 2>/dev/null; pkill -f 'next-server' 2>/dev/null; sleep 1
( cd "$REPO/web" && NEXUS_API=http://localhost:8787 setsid npx next dev -p 3000 -H 0.0.0.0 \
    >"$DEV/web_dev.log" 2>&1 & )
for i in $(seq 1 90); do
  c=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/ 2>/dev/null)
  case "$c" in 200|307|308) echo "  web $c (${i}s)"; break;; esac; sleep 1
done

echo ""
echo "✅ NEXUS local:"
echo "   Frontend : http://localhost:3000"
echo "   API      : http://localhost:8787/api/health"
echo "   PG DSN   : $DSN"
echo "   Logs     : $DEV/serve.log  ·  $DEV/web_dev.log  ·  $DEV/pg.log"
echo "Para parar:  pkill -f next-server; pkill -f nexus-dev/nexus-server; \"$PGBIN/pg_ctl\" -D \"$DEV/pgdata\" -m fast stop"
