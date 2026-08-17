# ADR-0002: Custom runner image is a tooling overlay on the bundled image

## Status

Accepted — 2026-08-18.

## Context

The official bundled runner image is `ghcr.io/actions/actions-runner:latest`. The project brief calls for a "custom Docker image for the runner," so something has to be customized. Three patterns are common:

1. Pull the bundled image as-is and add tooling per-job via `actions/setup-*` actions.
2. `FROM ghcr.io/actions/actions-runner:latest` and bake the toolchain in with `RUN`.
3. `FROM mcr.microsoft.com/actions/runner` (upstream open-source runner) and copy ARC's `entrypoint.sh` into it. Leanest image.

## Decision

Pattern **(2)**: layer the Go toolchain, Docker CLI, and any other required tools on top of `ghcr.io/actions/actions-runner:latest`.

## Consequences

- The Dockerfile for the runner image is short, obvious, and reproducible: `FROM` + `RUN apt-get install …` + (optionally) a non-root user.
- Image is large but cold-start is fast — no per-job installs.
- The image has to be rebuilt and re-pushed when tool versions change. ARC itself does *not* auto-rebuild this; we own that loop.
- We do **not** take on pattern (3)'s complexity for v1. Pattern (3) is the right answer at scale but is not justified for a single-repo learning project.

## Alternatives considered

- **Pattern 1, no custom image.** Drops the "custom Dockerfile" requirement from the brief. Rejected: the brief specifically asks for one.
- **Pattern 3, from upstream runner.** Leanest, but adds an entrypoint to maintain. Rejected as overkill for v1.