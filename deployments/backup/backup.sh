#!/usr/bin/env bash
set -euo pipefail

# InfraMind backup script.
# Backs up TimescaleDB (full dump), Redis (RDB), and EMQX data.
#
# Usage:
#   ./backup.sh                 # full backup to $BACKUP_DIR/YYYYMMDD-HHMMSS
#   BACKUP_DIR=/srv/backups ./backup.sh
#
# Environment overrides:
#   BACKUP_DIR      destination directory (default ./backups)
#   DB_URL          postgres URL (default local dev)
#   REDIS_CLI       redis-cli binary (default redis-cli)
#   EMQX_DATA_DIR   emqx data dir (default /opt/emqx/data)

BACKUP_DIR="${BACKUP_DIR:-./backups}"
DB_URL="${DB_URL:-postgres://infra:infra@localhost:5432/inframind?sslmode=disable}"
REDIS_CLI="${REDIS_CLI:-redis-cli}"
EMQX_DATA_DIR="${EMQX_DATA_DIR:-/opt/emqx/data}"

STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="$BACKUP_DIR/$STAMP"
mkdir -p "$DEST"

log() { echo "[backup] $*"; }

log "starting backup -> $DEST"

# 1. TimescaleDB logical backup (full schema + data, compressible)
log "dumping TimescaleDB..."
pg_dump "$DB_URL" --format=custom --compress=9 \
  --file="$DEST/postgres.dump"
log "  -> postgres.dump ($(du -h "$DEST/postgres.dump" | cut -f1))"

# 2. Redis snapshot (RDB) via SAVE; copy the dump file
log "snapshotting Redis..."
"$REDIS_CLI" SAVE >/dev/null
RDB_FILE=$("$REDIS_CLI" CONFIG GET dir | sed -n 2p)/dump.rdb
if [ -f "$RDB_FILE" ]; then
  cp "$RDB_FILE" "$DEST/redis-dump.rdb"
  log "  -> redis-dump.rdb ($(du -h "$DEST/redis-dump.rdb" | cut -f1))"
else
  log "  WARN: could not locate redis dump.rdb, skipping"
fi

# 3. EMQX data directory (config, auth users, ACLs)
if [ -d "$EMQX_DATA_DIR" ]; then
  tar -czf "$DEST/emqx-data.tar.gz" -C "$EMQX_DATA_DIR" .
  log "  -> emqx-data.tar.gz ($(du -h "$DEST/emqx-data.tar.gz" | cut -f1))"
else
  log "  WARN: $EMQX_DATA_DIR not found, skipping"
fi

# 4. Retention: keep last N backups
RETENTION="${RETENTION:-7}"
log "pruning backups older than $RETENTION"
ls -1dt "$BACKUP_DIR"/*/ 2>/dev/null | tail -n +"$((RETENTION + 1))" | xargs -r rm -rf

log "backup complete -> $DEST"
