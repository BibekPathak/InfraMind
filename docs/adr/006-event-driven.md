# ADR-006: Event-Driven Architecture

## Status

Accepted

## Context

InfraMind services need to react to state changes: telemetry arrives, health scores update, alerts fire. Tightly coupled request-response patterns don't scale with complexity.

## Decision

Use an event-driven architecture with an in-process EventBus and standardized event format.

## Consequences

- Domain packages publish events; other packages subscribe without direct imports.
- Easy to add new event handlers without modifying existing code.
- In-process bus keeps latency low and avoids distributed complexity during Phase 1.
- Event schema (`id`, `type`, `source`, `timestamp`, `data`) enforced across all domains.
- Can replace with Redis Streams or Kafka later without changing domain logic.
