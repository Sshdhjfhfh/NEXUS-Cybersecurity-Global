#!/bin/bash
# Dev stack supervisor for NEXUS: keeps PostgreSQL, the Go backend and the Next
# dev server alive, restarting any of them if they die. Non-destructive: it never
# drops the database or rebuilds; it only (re)starts processes that are down.
#
#   nohup scripts/dev-supervise.sh >/tmp/nexus-supervisor.log 2>&1 &
#
# Stop it with:  pkill -f dev-supervise.sh
set -u
export PATH=/usr/local/go/bin:/home/saul/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

ROOT=/home/saul/nexus-real
PGBIN=/home/saul/.nexus-dev/pg/bin
PGDATA=/home/saul/.nexus-dev/pgdata
PGPORT=55432
export LD_LIBRARY_PATH=/home/saul/.nexus-dev/pg/lib:${LD_LIBRARY_PATH:-}
export NEXUS_PG_DSN="postgres://nexus@127.0.0.1:${PGPORT}/nexus?sslmode=disable"
# Dev only: lets the local test identity provider (plain http on 127.0.0.1)
# be used for SSO. Production issuers must be https on a public address.
export NEXUS_SSO_DEV_INSECURE=1
NEXUS_BIN=/home/saul/.nexus-dev/nexus-server
INTERVAL=${SUPERVISE_INTERVAL:-10}
# The backend listens on :8787. When something else holds that port (on Windows
# typically a stale `netsh interface portproxy` rule), it starts on the fallback
# port instead and Next is restarted to proxy to it, so the app keeps working.
PREFERRED_PORT=8787
FALLBACK_PORT=${NEXUS_DEV_FALLBACK_PORT:-8790}
PORT_FILE=/tmp/nexus-backend-port      # port the backend was started on
NEXT_PORT_FILE=/tmp/nexus-next-backend # backend port Next was started with

log() { echo "$(date '+%F %T') [supervise] $*"; }

pg_up()      { "$PGBIN/pg_ctl" -D "$PGDATA" status >/dev/null 2>&1; }
backend_port() { cat "$PORT_FILE" 2>/dev/null || echo "$PREFERRED_PORT"; }
backend_up() { [ "$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "http://127.0.0.1:$(backend_port)/api/health" 2>/dev/null)" = "200" ]; }
# port_free: could the backend bind this port on all interfaces right now? Like
# Go it sets SO_REUSEADDR, so sockets still closing (TIME_WAIT) do not count.
port_free()  { python3 -c 'import socket,sys; s=socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1); s.bind(("0.0.0.0", int(sys.argv[1])))' "$1" 2>/dev/null; }
next_up()    { ss -ltn 2>/dev/null | grep -q ':3000 '; }

start_pg() {
  log "PostgreSQL down -> starting"
  rm -f "$PGDATA/postmaster.pid" 2>/dev/null
  "$PGBIN/pg_ctl" -D "$PGDATA" -l /tmp/pg.log -o "-p ${PGPORT}" -w -t 60 start >/dev/null 2>&1
}

start_backend() {
  [ -x "$NEXUS_BIN" ] || return 0
  local port=$PREFERRED_PORT
  if ! port_free "$port"; then
    if ! port_free "$FALLBACK_PORT"; then
      log "backend down, but ports $PREFERRED_PORT and $FALLBACK_PORT are both in use -> not starting"
      return 0
    fi
    log "port $PREFERRED_PORT is held by another process -> using $FALLBACK_PORT." \
      "On Windows this is usually a stale portproxy rule; remove it in an admin PowerShell:" \
      "netsh interface portproxy delete v4tov4 listenport=$PREFERRED_PORT listenaddress=0.0.0.0"
    port=$FALLBACK_PORT
  fi
  log "backend down -> starting on :$port"
  echo "$port" >"$PORT_FILE"
  setsid bash -c "cd $ROOT && exec $NEXUS_BIN -addr :$port -data /home/saul/.nexus-dev/data -proxy=''" </dev/null >>/tmp/nexus.log 2>&1 &
  disown || true
}

start_next() {
  local port
  port=$(backend_port)
  log "next dev down -> starting (backend :$port)"
  export NEXUS_API="http://127.0.0.1:$port"
  export NEXT_PUBLIC_DEV_BACKEND_PORT="$port" # SSE connects to the backend directly in dev
  echo "$port" >"$NEXT_PORT_FILE"
  setsid bash -c "cd $ROOT/web && exec npx next dev -p 3000 -H 0.0.0.0" </dev/null >>/tmp/next.log 2>&1 &
  disown || true
}

log "supervisor started (interval ${INTERVAL}s)"
while true; do
  pg_up      || start_pg
  backend_up || start_backend
  if next_up && [ "$(cat "$NEXT_PORT_FILE" 2>/dev/null)" != "$(backend_port)" ]; then
    log "backend moved to :$(backend_port) -> restarting next dev to proxy to it"
    pkill -f "next dev -p 3000" 2>/dev/null
    # next-server (the child that holds the port) has a different command line.
    for pid in $(ss -ltnp 'sport = :3000' 2>/dev/null | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u); do
      kill "$pid" 2>/dev/null
    done
    sleep 3
  fi
  next_up    || start_next
  sleep "$INTERVAL"
done
