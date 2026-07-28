# InfraMind — Engineering Handbook

## Project Vision

Autonomous Infrastructure Intelligence Platform. Monitor, predict, and maintain industrial assets autonomously.

## Architecture Principles

- **Event-driven**: All state changes emit events. Services communicate asynchronously.
- **Modular Monolith**: Domain packages, not microservices. Extract later if needed.
- **Domain-Driven Design**: Each business capability is a self-contained package.
- **CQRS-ready**: Separate read and write models where complexity demands it.
- **API-first**: Define contracts in OpenAPI before writing handlers.

## Coding Standards

- **Go**: `gofmt` + `staticcheck` enforced. No global variables. `context.Context` as first param.
- **Python**: PEP 8, type hints everywhere. FastAPI for HTTP.
- **TypeScript**: Strict mode, no `any`. Preact patterns in Next.js.
- **No global state**: Config, DB pools, loggers are injected.
- **Wrap errors**: Always annotate with context using `fmt.Errorf("…: %w", err)`.
- **Structured logging only**: Use `log/slog`. No `fmt.Println` in production code.
- **Unit tests required**: `_test.go` alongside every package. Table-driven tests preferred.

## Repository Structure

```
infraMind/
├── AGENTS.md              # This file
├── docker-compose.yml     # Single command startup
├── docs/
│   └── adr/               # Architecture Decision Records
├── backend/               # Go — API, ingestion, domain logic
│   ├── cmd/server/
│   ├── internal/          # Domain packages (asset/, device/, telemetry/, …)
│   ├── pkg/               # Shared utilities
│   └── spec/              # OpenAPI 3.1
├── frontend/              # Next.js + TypeScript
│   ├── app/               # Pages
│   ├── components/        # UI components
│   └── lib/               # API client, utilities
├── ai/                    # Python + FastAPI
│   └── engines/           # Health score, anomaly detection, etc.
├── simulator/             # Go — telemetry generator
│   ├── device/
│   ├── telemetry/
│   └── scenarios/
└── deployments/
    └── docker/            # Dockerfiles per service
```

## Naming Conventions

- **Go packages**: lowercase, single word (`asset`, `device`, `telemetry`)
- **Go files**: lowercase, underscore for compound (`health_score.go`)
- **REST endpoints**: plural nouns, kebab-case (`/api/v1/health-scores`)
- **JSON fields**: camelCase (`deviceId`, `temperature`)
- **Database columns**: snake_case (`device_id`, `firmware_version`)
- **MQTT topics**: lowercase, forward-slash hierarchy (`telemetry/{device_id}/data`)
- **Environment variables**: UPPER_SNAKE_CASE (`INFRA_DB_URL`)
- **Git branches**: `type/description` (`feat/health-score`, `fix/telemetry-batch`)

## API Guidelines

- **Versioned**: `/api/v1/…`
- **RESTful**: Resources, not actions. Use HTTP methods correctly.
- **JSON**: Request and response bodies. `Content-Type: application/json`.
- **Idempotent**: PUT and DELETE are idempotent. POST creates.
- **Pagination**: `?page=1&limit=50` with `X-Total-Count` header.
- **Errors**: `{ "error": { "code": "NOT_FOUND", "message": "…", "details": {} } }`
- **Timestamps**: RFC 3339, always UTC.

## Database Guidelines

- **UUID v7**: Primary keys. Time-ordered, better index locality than random UUIDs.
- **Soft delete**: `deleted_at TIMESTAMPTZ NULL` on all entity tables.
- **Timestamps**: `created_at`, `updated_at` on every entity table.
- **Migrations only**: Never edit schema manually. All changes via Goose.
- **Indexes**: On `device_id`, `asset_id`, `time` (hypertable), `status`.
- **Foreign keys**: Enforced at DB level. Cascade rules explicit.

## Logging Standards

- **Format**: JSON structured logs. `slog.JSONHandler`.
- **Levels**: `DEBUG` (development), `INFO` (normal ops), `WARN` (expected anomalies), `ERROR` (unexpected failures).
- **Context**: Include `request_id`, `device_id`, `asset_id`, `correlation_id` in every log line.
- **No sensitive data**: Never log passwords, tokens, or PII.

## Testing Philosophy

- **Unit tests**: Required for all business logic. Table-driven in Go.
- **Integration tests**: Test against real DB/MQTT in CI. `testcontainers-go` recommended.
- **No mocks for external services**: Use real Docker containers for integration tests.
- **Coverage target**: >80% for `internal/` packages.

## Event Bus Conventions

- **In-process**: EventBus dispatches to registered handlers synchronously.
- **Event type**: `{ "id": "uuid", "type": "device.telemetry", "source": "simulator", "timestamp": "…", "data": {} }`
- **Naming**: `{domain}.{action}` — `device.registered`, `telemetry.ingested`, `alert.raised`

## Security Requirements

- **JWT**: Short-lived access tokens (15 min). Refresh tokens (7 days).
- **TLS**: All external endpoints. mTLS for inter-service.
- **Input validation**: All API inputs validated. SQL injection prevented via parameterized queries.
- **Rate limiting**: Per-IP and per-token.
- **No secrets in code**: Environment variables or Vault.

## Code Review Checklist

- [ ] No global variables
- [ ] Context passed correctly
- [ ] Errors wrapped with context
- [ ] Logs are structured, no fmt.Print
- [ ] SQL uses parameterized queries
- [ ] UUID v7 used for primary keys
- [ ] Migrations included for schema changes
- [ ] Unit tests for new logic
- [ ] OpenAPI spec updated
- [ ] No secrets or hardcoded values

## Verification Commands

```bash
# Backend
cd backend && go build ./... && go test ./... && staticcheck ./...

# Frontend
cd frontend && npm run lint && npm run typecheck

# AI
cd ai && ruff check . && mypy .

# All services
docker compose --profile sim up --build
```

## Future Roadmap

- Phase 1: Foundation & Architecture (current)
- Phase 1.5: Domain Model Design
- Phase 2: Edge Telemetry Platform
- Phase 3: Cloud Platform
- Phase 4: Digital Twin
- Phase 5: AI Engine
- Phase 6: Infrastructure Operations
- Phase 7: Multi-Asset Intelligence
- Phase 8: Autonomous Infrastructure
- Phase 9: Enterprise Platform
- Phase 10: Production & Scale
