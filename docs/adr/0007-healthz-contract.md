# ADR-0007: Verification contract — `/healthz` returns 200 with body `ok`

## Status

Accepted — 2026-08-18.

## Context

The project's end-to-end success criterion is "deploy and verify a Go application in Kubernetes." `verify` is the word carrying the load. Without an explicit contract, "verified" can mean anything from "the pod scheduled" to "the business logic behaves correctly."

## Decision

Verification is a single HTTP call from the Runner Pod to the in-cluster Service URL:

```
GET http://kube-runner-app.app.svc.cluster.local:8080/healthz
```

The contract is:

- HTTP status `200`
- Response body equals `ok` (the literal three-character string, no trailing whitespace, no JSON wrapper)

If either condition fails, the workflow step fails and the run is red.

## Consequences

- Catches "container started but the binary crashed" (no listener → connect failure).
- Catches "container is up but the route isn't wired" (5xx).
- Catches "the process started but the health route wasn't registered" (returns 404 or empty body).
- Does **not** test business behaviour, only readiness. That's intentional.
- The contract is a single line of `curl` plus a body check; it lives in the workflow as the last step. Any future change to the contract is an ADR-worthy change.

## Alternatives considered

- **Liveness only** (`curl --fail`) — wouldn't catch "binary up, route not wired."
- **Behavioural smoke test** (`curl /`) — too coupled to the Go app's content; if we change the app's homepage the deploy breaks for the wrong reason.
- **End-to-end with side effects** — out of scope for a CI/deploy smoke test.