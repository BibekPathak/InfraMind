# ADR-002: PostgreSQL + TimescaleDB

## Status

Accepted

## Context

InfraMind stores both relational data (assets, devices, users) and time-series data (telemetry, health scores). Using separate databases adds operational complexity.

## Decision

Use PostgreSQL with the TimescaleDB extension as the single database.

## Consequences

- Relational and time-series data in one database, one backup strategy.
- TimescaleDB hypertables provide automatic partitioning by time.
- Continuous aggregates for downsampled views.
- Native compression reduces storage for old telemetry.
- Single connection pool simplifies backend code.
