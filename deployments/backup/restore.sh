#!/usr/bin/env bash
set -euo pipefail

# InfraMind restore script.
# Restores a backup directory produced by backup.sh.
#
# Usage:
#   ./restore.sh ./backups/20260101-120000
#
# Environment overrides:
#   DB_URL      postgres URL (target database)
#   REDIS_CLI   redis-cli binary

BACKUP="$1"
DB_URL="${DB_URL:-postgres://infra:infra@localhost:5432/inframind?sslmode=disable}"
REDIS_CLI="${REDIS_CLI:-redis-cli}"

if [ -z "$BACKUP" ] || [ ! -f "$BACKUP/postgres.dump" ]; then
  echo "usage: $0 <backup-dir>  (must contain postgres.dump)" >&2
  exit 1
fi

log() { echo "[restore] $*"; }

log "restoring from $BACKUP"

# 1. Restore TimescaleDB (requires an existing empty target database)
log "restoring TimescaleDB..."
pg_restore "$DB_URL" --clean --if-exists --no-owner --no-privileges \
  "$BACKUP/postgres.dump"
log "  -> postgres.dump restored"

# 2. Restore Redis RDB snapshot
if [ -f "$BACKUP/redis-dump.rdb" ]; then
  log "restoring Redis snapshot..."
  "$REDIS_CLI" FLUSHALL >/dev/null
  RDB_DIR=$("$REDIS_CLI" CONFIG GET dir | sed -n 2p)
  cp "$BACKUP/redis-dump.rdb" "$RDB_DIR/dump.rdb"
  "$REDIS_CLI" SHUTDOWN NOSAVE >/dev/null 2>&1 || true
  # redis will reload dump.rdb on restart (e.g. via compose restart redis)
  log "  -> redis-dump.rdb staged (restart redis to load)"
else
  log "  WARN: no redis snapshot found, skipping"
fi

# 3. EMQX data
if [ -f "$BACKUP/emqx-data.tar.gz" ]; then
  log "restoring EMQX data..."
  tar -xzf "$BACKUP/emqx-data.tar.gz" -C /opt/emqx/data
  log "  -> emqx-data.tar.gz restored (restart emqx to apply)"
fi

log "restore complete. Restart affected services via: docker compose restart backend redis emqx"
