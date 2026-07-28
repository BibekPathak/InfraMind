# Domain Model Design — InfraMind

## Bounded Contexts

InfraMind is organized into the following bounded contexts:

```
┌─────────────────────────────────────────────────────┐
│                   InfraMind                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │  Asset   │  │  Device  │  │   Telemetry      │  │
│  │  Mgmt   │──│  Mgmt   │──│   Ingestion      │  │
│  └──────────┘  └──────────┘  └──────────────────┘  │
│       │              │               │              │
│       ▼              ▼               ▼              │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │  Alert   │  │  Health  │  │   Work Order     │  │
│  │  Engine  │──│  Service │──│    (future)      │  │
│  └──────────┘  └──────────┘  └──────────────────┘  │
│       │              │                              │
│       ▼              ▼                              │
│  ┌──────────┐  ┌──────────┐                         │
│  │  Auth /  │  │  Org /   │                         │
│  │  Access  │  │  Tenant  │                         │
│  └──────────┘  └──────────┘                         │
└─────────────────────────────────────────────────────┘
```

---

## 1. Domain Glossary

### Asset

A physical or logical entity being monitored. An Asset is the top-level entity
in the domain hierarchy.

| Attribute | Type | Description |
|-----------|------|-------------|
| id | UUID v7 | Primary key |
| name | string | Human-readable name |
| type | string | Asset kind: `transformer`, `pump`, `motor`, `generator`, `hvac` |
| location | JSONB | Geographic/plant location (lat/lng, site, zone) |
| metadata | JSONB | Arbitrary key-value metadata |
| createdAt | TIMESTAMPTZ | Creation timestamp |
| updatedAt | TIMESTAMPTZ | Last update timestamp |
| deletedAt | TIMESTAMPTZ | Soft delete timestamp |

**Invariants:**
- Asset name is required and unique within an organization
- Asset type must be one of the predefined types
- An Asset can have zero or more Devices

**Behaviors:**
- `register(name, type, location, metadata)` → Asset
- `decommission(id)` → void (soft delete)
- `listByType(type)` → Asset[]
- `getLocation(id)` → Location

---

### Device

An edge hardware device attached to an Asset, producing telemetry.

| Attribute | Type | Description |
|-----------|------|-------------|
| id | UUID v7 | Primary key |
| assetId | UUID v7 | Parent Asset (FK) |
| firmwareVersion | string | Installed firmware version |
| status | string | `online`, `offline`, `error`, `maintenance` |
| location | JSONB | Device-specific location (may differ from Asset) |
| lastHeartbeat | TIMESTAMPTZ | Last communication time |
| createdAt | TIMESTAMPTZ | Registration timestamp |
| updatedAt | TIMESTAMPTZ | Last update timestamp |
| deletedAt | TIMESTAMPTZ | Soft delete timestamp |

**Invariants:**
- Device must belong to exactly one Asset
- Device ID is globally unique
- Status transitions: `offline → online → error → offline`

**Behaviors:**
- `register(assetId, firmwareVersion)` → Device
- `handleHeartbeat(deviceId)` → void (updates status + timestamp)
- `markOffline(deviceId)` → void (after heartbeat timeout)
- `configure(deviceId, config)` → void

---

### Telemetry

Time-series measurements produced by a Device.

| Attribute | Type | Description |
|-----------|------|-------------|
| time | TIMESTAMPTZ | Measurement timestamp (UTC) |
| deviceId | UUID v7 | Source Device (FK) |
| temperature | FLOAT | °C |
| current | FLOAT | Amperes |
| voltage | FLOAT | Volts |
| humidity | FLOAT | Relative humidity % |

**Invariants:**
- telemetry is immutable — never updated or deleted
- time is always UTC
- Each telemetry point has exactly one deviceId

**Behaviors:**
- `ingest(deviceId, metrics[])` → void (batch insert)
- `queryByDevice(deviceId, from, to)` → Telemetry[]
- `queryLatest(deviceId)` → Telemetry
- `aggregate(deviceId, window, function)` → AggregatedTelemetry

---

### Alert

A notification generated when a rule or AI analysis detects an anomaly.

| Attribute | Type | Description |
|-----------|------|-------------|
| id | UUID v7 | Primary key |
| deviceId | UUID v7 | Source Device (FK) |
| severity | string | `info`, `warning`, `critical` |
| rule | string | Rule name or ID that triggered the alert |
| message | string | Human-readable summary |
| status | string | `open`, `acknowledged`, `resolved` |
| createdAt | TIMESTAMPTZ | When alert was raised |
| updatedAt | TIMESTAMPTZ | Last status change |

**Invariants:**
- Alert status transitions: `open → acknowledged → resolved`
- A resolved alert cannot be reopened

**Behaviors:**
- `raise(deviceId, severity, rule, message)` → Alert
- `acknowledge(alertId)` → void
- `resolve(alertId)` → void
- `listByDevice(deviceId, status)` → Alert[]

---

### HealthScore

A computed 0-100 health metric for a Device, derived from telemetry.

| Attribute | Type | Description |
|-----------|------|-------------|
| deviceId | UUID v7 | Source Device (PK) |
| score | FLOAT | 0-100 health score |
| level | string | `healthy`, `warning`, `critical` |
| factors | JSONB | Contributing factors with impacts |
| computedAt | TIMESTAMPTZ | Calculation timestamp |

**Invariants:**
- Only the latest HealthScore per device is stored (1:1 with Device)
- Score calculation is deterministic and auditable

**Behaviors:**
- `calculate(deviceId, telemetry)` → HealthScore
- `getLatest(deviceId)` → HealthScore

---

### Organization (stub — Phase 9)

Multi-tenant container for Assets, Devices, and Users.

| Attribute | Type | Description |
|-----------|------|-------------|
| id | UUID v7 | Primary key |
| name | string | Organization name |
| settings | JSONB | Org-level configuration |
| createdAt | TIMESTAMPTZ | Creation timestamp |

**Invariants:**
- An Organization owns all Assets within it
- Users belong to exactly one Organization

---

### WorkOrder (stub — Phase 6)

A maintenance task generated by alerts or manual creation.

| Attribute | Type | Description |
|-----------|------|-------------|
| id | UUID v7 | Primary key |
| assetId | UUID v7 | Related Asset |
| alertId | UUID v7 | Source Alert (nullable) |
| type | string | `inspection`, `repair`, `replacement` |
| priority | string | `low`, `medium`, `high`, `critical` |
| status | string | `open`, `assigned`, `in_progress`, `completed`, `cancelled` |
| assignedTo | string | Engineer ID or name |
| estimatedCost | FLOAT | Cost estimate |
| createdAt | TIMESTAMPTZ | Creation timestamp |

---

## 2. Event Catalog

All events follow this schema:

```json
{
  "id": "uuid",
  "type": "{domain}.{action}",
  "source": "{service_name}",
  "timestamp": "2026-07-28T12:00:00Z",
  "data": { }
}
```

### Defined Events

| Event Type | Source | Trigger | Payload |
|-----------|--------|---------|---------|
| `asset.created` | backend | POST /assets | Asset object |
| `asset.deleted` | backend | DELETE /assets/{id} | `{ "id": "..." }` |
| `device.registered` | backend | POST /devices | Device object |
| `device.heartbeat` | backend | POST /devices/{id}/heartbeat | `{ "deviceId": "..." }` |
| `device.status_changed` | backend | Internal (heartbeat timeout) | `{ "deviceId": "...", "from": "online", "to": "offline" }` |
| `device.configuration_updated` | backend | PUT /devices/{id}/config | `{ "deviceId": "...", "config": {} }` |
| `telemetry.ingested` | ingester | MQTT message processed | `{ "deviceId": "...", "time": "...", "scenario": "..." }` |
| `telemetry.batch_ingested` | ingester | Batch insert | `{ "deviceId": "...", "count": 10 }` |
| `alert.raised` | alert engine | Rule evaluation | Alert object |
| `alert.acknowledged` | alert | PATCH /alerts/{id}/acknowledge | `{ "alertId": "..." }` |
| `alert.resolved` | alert | PATCH /alerts/{id}/resolve | `{ "alertId": "..." }` |
| `health.calculated` | health | Telemetry ingested | HealthScore object |
| `mqtt.message` | backend | Incoming MQTT | `{ "topic": "...", "payload": "..." }` |

### Future Events

| Event Type | Phase | Trigger |
|-----------|-------|---------|
| `workorder.created` | 6 | Alert resolved or manual creation |
| `workorder.assigned` | 6 | Engineer assigned |
| `workorder.completed` | 6 | Work order closed |
| `user.invited` | 9 | User invited to org |
| `user.role_changed` | 9 | RBAC update |

---

## 3. Aggregate Boundaries

### Entity Relationships

```
Organization (future)
    │
    ├── Asset (1:N)
    │       │
    │       ├── Device (1:N)
    │       │       │
    │       │       ├── Telemetry (1:N, hypertable)
    │       │       ├── HealthScore (1:1 latest)
    │       │       ├── Alert (1:N)
    │       │       └── Config (1:1)
    │       │
    │       └── WorkOrder (1:N, future)
    │
    └── User (1:N, future)
```

### Ownership Rules

| Parent | Child | Cascade | Notes |
|--------|-------|---------|-------|
| Asset | Device | CASCADE | Deleting an Asset deletes all its Devices |
| Device | Telemetry | NO ACTION | Telemetry is never cascade-deleted (immutable history) |
| Device | Alert | CASCADE | Alerts are tied to device lifecycle |
| Device | HealthScore | CASCADE | Health score is derived, follows device |
| Asset | WorkOrder | RESTRICT | Cannot delete asset with open work orders |

### Transactional Boundaries

| Operation | Aggregate | Notes |
|-----------|-----------|-------|
| Register Device | Device | Requires Asset to exist (FK validation) |
| Ingest Telemetry | Telemetry | Requires Device to exist (FK validation) |
| Raise Alert | Alert | References Device, but no transactional coupling |
| Calculate Health | HealthScore | Eventually consistent with telemetry ingestion |

---

## 4. API Contract Review

### Current Endpoints vs Domain Model

| Endpoint | Domain | Status | Notes |
|----------|--------|--------|-------|
| `GET /api/v1/health` | System | ✅ | No change needed |
| `POST /api/v1/assets` | Asset | ✅ | Returns full asset |
| `GET /api/v1/assets` | Asset | ✅ | Pagination needed later |
| `GET /api/v1/assets/{id}` | Asset | ✅ | Include device count? |
| `DELETE /api/v1/assets/{id}` | Asset | ✅ | Soft delete |
| `POST /api/v1/devices` | Device | ✅ | Validate assetId exists |
| `GET /api/v1/devices/{id}` | Device | ✅ | Include health score? |
| `POST /api/v1/devices/{id}/heartbeat` | Device | ✅ | Auto sets status=online |
| `GET /api/v1/assets/{id}/devices` | Device | ✅ | Filters by asset |
| `GET /api/v1/devices/{id}/telemetry` | Telemetry | ✅ | Pagination default limit |
| `GET /api/v1/telemetry/live` | Telemetry | ✅ | Add batch query endpoint later |
| `GET /api/v1/health/{deviceId}` | Health | ✅ | Query params for metrics |
| `GET /api/v1/alerts` | Alert | ✅ | Stub — no rules yet |

### Missing Endpoints (Future Phases)

| Endpoint | Phase | Priority |
|----------|-------|----------|
| `PATCH /api/v1/alerts/{id}/acknowledge` | 3 | High |
| `PATCH /api/v1/alerts/{id}/resolve` | 3 | High |
| `PUT /api/v1/devices/{id}/config` | 2 | High |
| `GET /api/v1/telemetry/aggregate` | 3 | Medium |
| `POST /api/v1/work-orders` | 6 | Medium |
| `GET /api/v1/organizations` | 9 | Low |

---

## 5. Database Schema Review

### Current Tables vs Domains

| Table | Domain | Status | Notes |
|-------|--------|--------|-------|
| `assets` | Asset | ✅ | Matches domain model exactly |
| `devices` | Device | ✅ | Add `certificates` and `config` JSONB columns later |
| `telemetry` | Telemetry | ✅ | Hypertable — correct design |
| *(missing)* | Alert | ❌ | Not yet created — add in Phase 3 |
| *(missing)* | HealthScore | ❌ | Not yet created — store latest in Redis for now |

### Recommended Alert Table (Phase 3)

```sql
CREATE TABLE alerts (
    id UUID PRIMARY KEY,
    device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    rule TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'resolved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 6. Key Design Decisions

### Decision 1: Device owns Telemetry, not Asset

Rationale: Telemetry is always emitted by a specific Device. Querying by Asset
requires a join through Devices. This keeps the telemetry table simple and
allows a Device to be reassigned to a different Asset without data migration.

### Decision 2: HealthScore is 1:1 with Device (latest only)

Rationale: Historical health scores can be reconstructed from telemetry replay.
Storing only the latest reduces write pressure and simplifies the read model.
If historical health tracking is needed later, add a `health_scores` hypertable.

### Decision 3: Alerts reference Device, not Asset

Rationale: Multiple Devices can belong to one Asset. Alerting at the Device
level gives finer granularity. Asset-level alerts are derived by aggregating
child Device alerts.

### Decision 4: Soft delete on Assets and Devices

Rationale: Prevents orphaned telemetry and historical data loss. Reports and
analytics often need to reference decommissioned entities.

---

## 7. Next Actions

| # | Action | Phase |
|---|--------|-------|
| 1 | Add `certificates` and `config` columns to devices table | 2 |
| 2 | Create alerts table migration | 3 |
| 3 | Add pagination to list endpoints | 2 |
| 4 | Implement alert rule evaluation engine | 3 |
| 5 | Create WorkOrder aggregate | 6 |
| 6 | Create Organization aggregate | 9 |
| 7 | Add event sourcing for critical events (telemetry, alerts) | 4 |
