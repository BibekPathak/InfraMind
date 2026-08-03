# InfraMind Load Testing

A Go-based load generator for validating telemetry ingestion and API latency
under sustained load.

## Build

```bash
cd deployments/loadtest
go build -o loadtest .
```

## MQTT ingestion test

Publishes telemetry to the broker at a fixed rate and measures publish latency:

```bash
./loadtest -mode mqtt -broker mqtt://localhost:1883 -rate 200 -seconds 30
```

Options:
- `-topic` — MQTT topic (default `telemetry/tx-001/data`)
- `-devices` — number of distinct device IDs to cycle through (default 1)

## API query test

Samples the live-telemetry endpoint and reports response latency percentiles:

```bash
./loadtest -mode api -url http://localhost:8080/api/v1 -rate 50 -seconds 20
```

## Output

```
=== Load Test Results ===
Total:            6000
Successful:       5995
Errors:           5
Avg latency:      0.012s
p50:              0.011s
p95:              0.019s
p99:              0.031s
Max:              0.240s
Throughput:       199.8 req/s
```

## Metrics to watch (Grafana)

While running, check:

- `infra_telemetry_ingested_total` — ingestion throughput vs. publish rate
- `infra_telemetry_batch_size` — batch flush distribution (should batch up to ~1000)
- `infra_http_request_duration_seconds` — API p95/p99 latency
- `infra_eventbus_publish_total` — event volume

## Tuning levers

| Knob | Where | Effect |
|------|-------|--------|
| `batchSize` / `batchTimeout` | `backend/internal/telemetry/ingester.go` | Larger batches = higher ingest throughput |
| DB pool size | `backend/internal/db/pool.go` (`pgxpool.New`) | More concurrent inserts under load |
| Rate limiter | `INFRA_RATE_LIMIT` (10.4) | API throughput ceiling |
| Hypertable chunk interval | migration 003 | Tune chunk size for retention/query mix |
