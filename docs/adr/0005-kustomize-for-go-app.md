# ADR-0005: Kustomize for the Go app's manifests, not raw YAML or Helm

## Status

Accepted — 2026-08-18.

## Context

The Go app's Kubernetes manifests need to:

- Reference an image that lives in the k3d built-in registry (`k3d-<name>-registry.localhost:5000/...`).
- Be `kubectl apply`-able from inside a workflow that doesn't have a Kubernetes templating engine installed.
- Be different across environments (today: only `dev`; tomorrow: probably `prod`).

Three options:

1. **Raw YAML** — simple, but the registry host and image tag leak into the workflow as literal strings.
2. **Kustomize overlays** — `images:` transformer rewrites `image:` deterministically per overlay. `kubectl apply -k` is built into stock kubectl.
3. **Helm chart** — full templating, values files, hooks. Overkill for two resources.

## Decision

Option 2 — Kustomize overlays. Layout:

```
k8s/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   └── service.yaml
└── overlays/
    └── dev/
        └── kustomization.yaml
```

The `images:` transformer in `overlays/dev/kustomization.yaml` rewrites the Go app's `image:` field to the k3d registry host.

## Consequences

- The registry path from ADR-0003 stops being a string the workflow templated inline; it's centralised in the overlay.
- Adding a `prod` overlay is one new directory; the base stays untouched.
- `kubectl apply -k k8s/overlays/dev` is the deploy command. Stock kubectl, no plugin.

## Alternatives considered

- **Raw YAML** — works, but bakes the registry host into the workflow file. That's the exact "infra detail leaking into the application of an infra detail" smell we wanted to avoid.
- **Helm** — for two resources, the values/templating overhead exceeds what it saves.