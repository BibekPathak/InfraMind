# ADR-008: WebSocket for Live Stream

## Status

Accepted

## Context

InfraMind needs to push real-time updates (telemetry, alerts, health scores) to the frontend dashboard. Options included Server-Sent Events (SSE) and WebSocket.

SSE is simpler for one-way data flow but cannot send commands from the frontend to the backend. The roadmap includes operator actions (acknowledge alerts, restart devices, trigger diagnostics) that require bidirectional communication.

## Decision

Use WebSocket with a typed event envelope for all real-time frontend communication.

## Event Envelope

```json
{
  "type": "telemetry.updated",
  "timestamp": "2026-07-28T12:00:00Z",
  "asset_id": "tx-001",
  "payload": {}
}
```

Event types will grow: `telemetry.updated`, `alert.created`, `alert.resolved`, `device.connected`, `device.disconnected`, `incident.opened`, `workorder.created`, `ai.recommendation.generated`.

## Consequences

- One persistent connection handles all event types.
- Bidirectional — frontend can send commands in future phases.
- Backend publishes events to all subscribed WebSocket clients via an internal hub.
- Frontend uses a single connection with auto-reconnect logic.
- Slightly more complex than SSE for the MVP, but avoids a migration later.
