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
NEXUS_BIN=/home/saul/.nexus-dev/nexus-server
INTERVAL=${SUPERVISE_INTERVAL:-10}

log() { echo "$(date '+%F %T') [supervise] $*"; }

pg_up()      { "$PGBIN/pg_ctl" -D "$PGDATA" status >/dev/null 2>&1; }
backend_up() { [ "$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 http://127.0.0.1:8787/api/health 2>/dev/null)" = "200" ]; }
next_up()    { ss -ltn 2>/dev/null | grep -q ':3000 '; }

start_pg() {
  log "PostgreSQL down -> starting"
  rm -f "$PGDATA/postmaster.pid" 2>/dev/null
  "$PGBIN/pg_ctl" -D "$PGDATA" -l /tmp/pg.log -o "-p ${PGPORT}" -w -t 60 start >/dev/null 2>&1
}

start_backend() {
  log "backend down -> starting"
  [ -x "$NEXUS_BIN" ] || return 0
  setsid bash -c "cd $ROOT && exec $NEXUS_BIN -data /home/saul/.nexus-dev/data -proxy=''" </dev/null >>/tmp/nexus.log 2>&1 &
  disown || true
}

start_next() {
  log "next dev down -> starting"
  export NEXUS_API='http://127.0.0.1:8787'
  setsid bash -c "cd $ROOT/web && exec npx next dev -p 3000 -H 0.0.0.0" </dev/null >>/tmp/next.log 2>&1 &
  disown || true
}

log "supervisor started (interval ${INTERVAL}s)"
while true; do
  pg_up      || start_pg
  backend_up || start_backend
  next_up    || start_next
  sleep "$INTERVAL"
done
