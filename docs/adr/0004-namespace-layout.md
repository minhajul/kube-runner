# ADR-0004: Three namespaces — `arc-systems`, `arc-runners`, `app`

## Status

Accepted — 2026-08-18.

## Context

ARC installs its controller and a listener; ephemeral runner pods land in whatever namespace the chart targets; and the Go app deploys somewhere. They could all share one namespace. Splitting them up costs YAML but pays back in RBAC clarity and operational reasoning.

## Decision

Three namespaces, single-purpose each:

- **`arc-systems`** — long-lived ARC controller + the `AutoscalingListener` for the runner-scale-set pool. GitHub App credentials Secret lives here too.
- **`arc-runners`** — ephemeral `Runner Pod`s, spawned by the `AutoscalingRunnerSet`. No long-lived resources.
- **`app`** — the deployed Go app (Deployment, Service, ServiceAccount, RBAC Role + RoleBinding). The runner pod does *not* live here.

The Runner Pod's ServiceAccount is bound to a `Role` in the `app` namespace only — never to anything in `arc-systems` or `arc-runners`. That is what makes "the runner can only touch the Go app" a sentence that's true at audit time, not just aspirational.

## Consequences

- NetworkPolicies and ResourceQuotas can be set per-namespace later without affecting the others.
- `kubectl get all -n arc-systems`, `-n arc-runners`, `-n app` are the three queries you need to know for debugging.
- Destroying the `app` namespace removes the deployed app without disturbing the runner pool. Destroying `arc-runners` is a no-op because it has no long-lived resources.
- The cost is four extra YAML files (Namespace × 3, plus the RoleBinding). Acceptable.

## Alternatives considered

- **One namespace** — rejected for inability to give a precise RBAC answer.
- **Two namespaces** (`arc-systems`, `arc-runners`; app co-located) — rejected because running the Go app alongside ephemeral runner pods makes the RBAC story fuzzier.