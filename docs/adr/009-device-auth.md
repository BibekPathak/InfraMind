# ADR-009: Device Authentication (MVP)

## Status

Accepted

## Context

Devices (simulators, ESP32s) connecting to EMQX need authentication. Options: EMQX built-in username/password, HTTP auth hook calling the backend, or mTLS with X.509 certificates.

HTTP auth couples MQTT broker availability to backend availability. mTLS adds certificate management overhead early. The MVP needs something simple and reliable.

## Decision

Use EMQX built-in username/password authentication for the MVP. Backend provisions credentials via the EMQX REST API when devices are registered.

## Evolution Path

```
MVP                     → Username + Password (EMQX built-in)
Phase 3+                → HTTP Auth (dynamic, tenant-aware)
Phase 7+ (Enterprise)   → mTLS / X.509 Device Certificates
Phase 9+ (Scale)        → Hardware-backed Identity (TPM)
```

## Consequences

- EMQX broker operates independently of the backend — devices can still connect if the backend is down.
- Backend registers MQTT credentials atomically during device registration.
- One set of credentials per device, stored in the `certificates` JSONB column.
- ACLs ensure each device can only publish to its own topic (`telemetry/{device_id}/#`).
- Migration to HTTP auth is additive: EMQX supports multiple auth backends simultaneously.
