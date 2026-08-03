# Disaster Recovery Runbook

## Overview

This runbook covers backup, restore, and recovery procedures for InfraMind.
Target recovery objectives:

| Metric | Target |
|--------|--------|
| RPO (Recovery Point Objective) | ≤ 1 hour (hourly DB backups) |
| RTO (Recovery Time Objective) | ≤ 30 minutes (compose restart) |
| Backup retention | 7 days on-site |

## Services & State

| Service | State | Backup method |
|---------|-------|---------------|
| TimescaleDB | Persistent (volume `pgdata`) | `pg_dump` custom format |
| Redis | Persistent (RDB) + Redis Streams | `SAVE` + copy `dump.rdb` |
| EMQX | Persistent (volume `emqx-data`) | tar of `/opt/emqx/data` |
| Backend/Frontend/AI/Simulator | Stateless | none (rebuild from image) |

---

## 1. Taking a Backup

Run the backup script on any host with access to the services:

```bash
cd deployments/backup
BACKUP_DIR=/srv/backups ./backup.sh
```

Output layout:

```
/srv/backups/20260101-120000/
├── postgres.dump          # TimescaleDB custom-format dump
├── redis-dump.rdb         # Redis snapshot
└── emqx-data.tar.gz       # EMQX users/ACLs/config
```

### Scheduling (cron example)

```cron
0 * * * * cd /srv/inframind/deployments/backup && BACKUP_DIR=/srv/backups ./backup.sh >> /var/log/inframind-backup.log 2>&1
```

### Verification

Periodically test restore into a scratch database:

```bash
createdb infra_restore_test
DB_URL=postgres://infra:infra@localhost:5432/infra_restore_test ./restore.sh /srv/backups/<latest>
```

---

## 2. Restoring

```bash
cd deployments/backup
./restore.sh /srv/backups/20260101-120000
```

Then restart affected services so they pick up restored state:

```bash
docker compose restart backend redis emqx
```

Notes:
- `restore.sh` requires the target database to exist and be empty for a clean
  restore (`pg_restore --clean --if-exists` will drop existing objects).
- Redis is restored by staging `dump.rdb` and restarting the container.
- EMQX data is restored into `/opt/emqx/data` and requires an EMQX restart.

---

## 3. Recovery Scenarios

### 3.1 Single service crash (backend, frontend, ai)

```bash
docker compose up -d --build backend frontend ai
```

Stateless services recover from their Docker images. No data loss.

### 3.2 Database corruption / accidental data loss

1. Stop the backend to prevent writes: `docker compose stop backend`
2. Restore the DB: `./restore.sh /srv/backups/<latest>`
3. Restart: `docker compose up -d backend`

### 3.3 Full host failure (all services lost)

1. Provision a new host with Docker + Docker Compose.
2. Clone the repo and copy the backup directory onto the host.
3. Start the stateless + infrastructure services first:
   ```bash
   docker compose up -d emqx timescaledb redis
   ```
4. Restore data (see section 2).
5. Start the remaining services:
   ```bash
   docker compose up -d backend frontend ai
   ```
6. Verify: `curl localhost:8080/api/v1/health` and check a device's telemetry.

### 3.4 Redis Streams loss (cross-instance event replay)

Redis Streams events are a durability enhancement, not the source of truth.
If the stream is lost:
- New events resume being written on restart.
- Domain data (alerts, work orders, actions, twins) lives in TimescaleDB and
  is recovered via the DB restore.
- No backfill of historical stream events is required for correctness.

---

## 4. Off-site / Additional Resilience (recommended)

- Copy `/srv/backups` to object storage (S3/GCS) or another host nightly.
- Enable TimescaleDB continuous backups via `pg_basebackup` + WAL archiving
  for near-zero RPO in production.
- Enable Redis AOF (`appendonly yes`) for finer-grained Redis recovery.
- Document a secondary/DR region with a warm standby (replica) once running
  multiple replicas in Phase 10 (Kubernetes).

---

## 5. Incident Response Checklist

1. Determine blast radius: DB, Redis, EMQX, or stateless services?
2. If data loss: stop writers (backend) immediately.
3. Select the most recent *verified* backup.
4. Restore and restart per sections above.
5. Verify data integrity with a smoke test (login, device list, telemetry query).
6. Record the incident and adjust backup frequency if RPO was missed.
