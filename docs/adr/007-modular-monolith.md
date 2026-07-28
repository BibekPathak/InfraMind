# ADR-007: Modular Monolith First

## Status

Accepted

## Context

InfraMind's architecture could be microservices or a monolith. Microservices add overhead (network, deployment, observability) that impedes early development velocity.

## Decision

Start with a modular monolith in Go, organized by domain package. Extract services later only where justified.

## Consequences

- One binary, one deployment, simple Docker Compose.
- Domain packages are decoupled by interface, not by network.
- EventBus provides the same decoupling as message queues.
- When a domain needs independent scaling (e.g., telemetry ingestion), extract it to a separate service with well-defined API boundaries.
- Cross-domain transactions are simpler in-process.
