# ADR-0001: Use the new ARC split chart, not the legacy single chart

## Status

Accepted — 2026-08-18.

## Context

GitHub Actions Runner controllerships as two Helm charts today:

- `gha-runner-scale-set-controller` (the controller)
- `gha-runner-scale-set-runner-scale-set` (the listener + runner-set pair)

The legacy umbrella chart `actions-runner-controller` still installs but uses the older `RunnerDeployment` / `RunnerSet` / `HorizontalRunnerAutoscaler` CRDs, which are in maintenance.

## Decision

Use the new split chart shape and the new `AutoscalingListener` / `AutoscalingRunnerSet` CRDs.

## Consequences

- All manifests, RBAC, and docs target the new API.
- The ARC community's recent examples, blog posts, and the official quickstart are aligned with this choice.
- If the chart is ever renamed again (it has been renamed twice), the migration will be a one-time YAML rename plus a CRD conversion.

## Alternatives considered

- **Legacy single chart.** Works, more Stack Overflow answers exist, but actively in maintenance. Rejected for a greenfield project where we don't yet have any legacy state to preserve.