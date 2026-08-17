# ADR-0006: ARC listener uses `polling`, not `http`

## Status

Accepted — 2026-08-18.

## Context

ARC's `AutoscalingListener` can be configured two ways:

- `runnerListenerType: http` — the controller pushes job events to the listener via an in-cluster Service.
- `runnerListenerType: polling` — the listener periodically polls GitHub's Actions API itself.

## Decision

Polling. The chart values set `runnerListenerType: polling`.

## Consequences

- No need to ensure the controller can reach the listener Service — there is no controller-to-listener call.
- No NetworkPolicy writeup for the listener Service in the README.
- Cold-start for the first job after idle is bounded by GitHub's API poll interval (default ~30 s) plus pod scheduling. Acceptable for a learning project; if this ever needs to be faster we revisit.
- GitHub rate limits apply. Acceptable for one repo's traffic.

## Alternatives considered

- **`http`** — slightly faster wake-up, requires the listener Service to be reachable from the controller pod. Adds an inbound connection to reason about. Rejected for v1.