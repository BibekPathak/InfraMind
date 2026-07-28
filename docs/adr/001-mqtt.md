# ADR-001: MQTT for Messaging

## Status

Accepted

## Context

InfraMind needs a messaging protocol for communication between edge devices, simulators, backend services, and the cloud. Requirements: pub/sub model, IoT-friendly, lightweight, QoS levels, broad ecosystem support.

## Decision

Use MQTT (via EMQX broker) as the primary messaging protocol.

## Consequences

- Pub/sub decouples producers (simulator/devices) from consumers (backend/AI).
- MQTT 5 provides session expiry, message expiry, and improved error codes.
- EMQX provides clustering, auth, ACLs, and a dashboard out of the box.
- OPC-UA bridging can be added later via EMQX gateways.
