# Metrics Service

This package is the feature boundary for metrics delivery.

It owns the HTTP-facing behavior for metric snapshots and streaming updates,
while delegating hardware sampling and sensor reads to `internal/sensors`.

## Scope

- Exposes route handlers for metrics endpoints.
- Builds normalized API responses from sampler snapshots.
- Orchestrates dependencies injected from application startup.

## Responsibilities and Boundaries

Belongs in `services/metrics`:

- Handler registration targets and transport behavior (`GET /metrics`, `GET /metrics/ws`).
- Response shaping and cross-sampler aggregation for API output.
- Service-level orchestration that composes sampler data into a single payload.

Belongs in `internal/sensors`:

- Hardware/system access (`lm-sensors`, sysfs, platform-specific readers).
- Sampling cadence internals, caching, and low-level parsing.
- Reusable collector logic that should not be exposed as a service boundary.

## Public Routes Owned by This Service

- `GET /metrics`: returns the current snapshot as JSON.
- `GET /metrics/ws`: streams periodic snapshots over WebSocket.

## Dependency Direction

`services/metrics` depends on `internal/sensors`.

`internal/sensors` must not depend on `services/metrics`.

This keeps transport/service concerns separate from low-level sensor logic.
