# ADR-0003: Use k3d's built-in registry for the deployed Go image

## Status

Accepted — 2026-08-18.

## Context

The workflow builds a Go image and then `kubectl apply`s a manifest referencing it. The `image:` field has to resolve from inside the cluster. Options:

- Push to `ghcr.io/<owner>/<repo>` and let the cluster pull from there.
- Push to k3d's built-in registry (`k3d-<name>.registry.localhost:5000`).
- Skip the registry and `kubectl apply` a manifest that references a local-only image.

## Decision

Use the **k3d built-in registry**. Image paths in the deployed manifest are templated to `k3d-<cluster>-registry.localhost:5000/<repo>:tag`. The workflow uses `docker push` to that registry after `docker build`.

## Consequences

- The whole demo loop is closed inside one cluster + one workstation. No external network calls for image distribution.
- The registry is destroyed with the cluster. We accept that the Go image has to be re-pushed on every cluster recreate.
- The `image:` field in the workflow's `kubectl apply` step is a string template that interpolates the registry host. We'll centralise that template.
- We do not currently model imagePullSecrets because the k3d registry is unauthenticated.

## Alternatives considered

- **GHCR.** Matches "real CI" patterns but requires registering a token in the runner's Secret and granting `ghcr.io` access to a k3d pod that may not always have outbound network.
- **No registry, local-only tarball.** Off-grid but doesn't survive a fresh pod schedule.