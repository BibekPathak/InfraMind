# ADR-005: Python for AI Engine

## Status

Accepted

## Context

InfraMind's AI layer (health scoring, anomaly detection, failure prediction) requires numerical computing, ML libraries, and rapid iteration. Go's ML ecosystem is immature.

## Decision

Use Python with FastAPI for the AI service.

## Consequences

- Access to NumPy, scikit-learn, PyTorch, and the Python ML ecosystem.
- FastAPI provides async, OpenAPI auto-docs, and Pydantic validation.
- AI service is a separate HTTP service — clean API boundary from the Go backend.
- Can be scaled independently when ML workloads increase.
- Type hints via Pydantic ensure contract alignment with the backend.
