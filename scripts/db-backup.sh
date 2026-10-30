#!/bin/sh
# NEXUS database backups. Takes a compressed pg_dump (custom format), checks
# that the archive is readable before keeping it, and deletes copies older than
# BACKUP_KEEP_DAYS. A failed or unreadable dump never replaces a good one.
#
#   db-backup.sh            one backup now
#   db-backup.sh --loop     one backup every BACKUP_INTERVAL_HOURS (the compose
#                           "backup" service runs this)
#
# Connection comes from the standard libpq variables: PGHOST, PGPORT, PGUSER,
# PGPASSWORD, PGDATABASE. Restore: see docs/engineering/BACKUP_RESTORE.md.
set -u

BACKUP_DIR=${BACKUP_DIR:-/backups}
KEEP_DAYS=${BACKUP_KEEP_DAYS:-14}
INTERVAL_HOURS=${BACKUP_INTERVAL_HOURS:-24}
DB=${PGDATABASE:-nexus}

log() { echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ') [db-backup] $*"; }

backup_once() {
  mkdir -p "$BACKUP_DIR" || return 1
  stamp=$(date -u '+%Y%m%d-%H%M%S')
  final="$BACKUP_DIR/$DB-$stamp.dump"
  tmp="$final.partial"
  if ! pg_dump --format=custom --compress=6 --no-owner --no-privileges --file="$tmp" "$DB"; then
    log "ERROR: pg_dump of $DB failed; previous backups kept"
    rm -f "$tmp"
    return 1
  fi
  # An archive pg_restore cannot list is not a backup.
  if ! pg_restore --list "$tmp" >/dev/null 2>&1; then
    log "ERROR: $tmp is not a readable archive; discarded"
    rm -f "$tmp"
    return 1
  fi
  mv "$tmp" "$final"
  chmod 600 "$final" 2>/dev/null
  log "ok: $final ($(du -h "$final" | cut -f1))"
  # Retention: only our own complete archives, never anything else in the dir.
  find "$BACKUP_DIR" -maxdepth 1 -type f -name "$DB-*.dump" -mtime +"$KEEP_DAYS" -print -delete |
    while read -r old; do log "expired: $old"; done
  rm -f "$BACKUP_DIR"/*.partial 2>/dev/null
  return 0
}

if [ "${1:-}" = "--loop" ]; then
  log "every ${INTERVAL_HOURS}h into $BACKUP_DIR, keeping ${KEEP_DAYS} days"
  while true; do
    backup_once
    sleep $((INTERVAL_HOURS * 3600))
  done
fi
backup_once
