# ADR-004: Next.js for Frontend

## Status

Accepted

## Context

InfraMind needs a web dashboard with real-time charts, device management, and asset visualization. The framework must support SSR, TypeScript, and a rich ecosystem for data visualization.

## Decision

Use Next.js with TypeScript for the frontend application.

## Consequences

- React ecosystem with strong tooling and community.
- SSR for performance and SEO (admin dashboards still benefit from fast initial load).
- App Router provides file-based routing aligned with our domain structure.
- ECharts integration for complex industrial visualizations.
- API client layer (`lib/api.ts`) keeps backend coupling centralized.
