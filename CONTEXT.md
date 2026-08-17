# Context

The glossary for `kube-runner`. Terms are defined as they are used in this project; if a term conflicts with its general Kubernetes / GitHub Actions meaning, the project meaning wins here.

## Scope

A self-hosted GitHub Actions runner that lives **inside a Kubernetes cluster** and uses that cluster to **deploy and verify a small Go application** via CI/CD. The end state is a closed loop: a push to the repo triggers a workflow, the workflow runs on a runner pod inside the cluster, and the workflow then deploys and smoke-tests a Go app on the same cluster.

## Terms

**Runner** — A self-hosted GitHub Actions agent that polls GitHub for jobs assigned to it and executes them. In this project, a Runner is *always* a Kubernetes Pod, not a VM or bare-metal host. "The runner" without qualification means the whole system that registers and runs Runner pods; "a runner pod" means one pod.

**Ephemeral Runner** — A Runner pod that exists for the lifetime of a single GitHub Actions job and is torn down when the job ends. The opposite of "persistent." The project uses ephemeral runners exclusively.

**ARC (Actions Runner Controller)** — GitHub's official Kubernetes operator for self-hosted runners. Installed as **two** Helm charts: the controller (`gha-runner-scale-set-controller`) and the runner-scale-set chart (`gha-runner-scale-set-runner-scale-set`). See ADR-0001.

**AutoscalingListener** — The long-lived ARC CR that holds the GitHub App credentials and watches GitHub for queued jobs. (Replaces the old "Listener" concept; the name change is part of the chart split.)

**AutoscalingRunnerSet** — The ARC CR that owns ephemeral Runner pods and defines `minRunners` / `maxRunners`. The scaling parameters live here, not on the Listener.

**RunnerSet (umbrella install)** — The Helm sub-chart that installs both an `AutoscalingListener` and an `AutoscalingRunnerSet` for the same GitHub target. One Helm install == one runner pool.

**Runner Image** — The Docker image the Runner Pod runs from. In this project: official `ghcr.io/actions/actions-runner:latest` **plus** the project's toolchain overlay (Go, Docker CLI, kubectl, anything else baked in via a separate Dockerfile that `FROM`s the official image). See ADR-0002.

**Custom Runner Image** — The image this project builds on top of the official runner image. Lives at the repo root as `Dockerfile.runner` (name to be confirmed in a later round). NOT to be confused with the App Image.

**GitHub App** — The credential the project uses to register runners with GitHub. Replaces the deprecated PAT-based registration. App credentials are stored in a Kubernetes Secret.

**Target Cluster** — The Kubernetes cluster the runner deploys *into*. In this project this is the **same** cluster the Runner Pod lives in (no remote cluster, no kubeconfig file).

**k3d Built-in Registry** — The Docker registry that ships with a `k3d registry create` setup. Image paths are `k3d-<cluster-name>-registry.localhost:5000/<repo>:<tag>`. Used as the deployment target for the Go app image. See ADR-0003.

**App Image** — The Docker image of the Go App, built by the workflow and pushed to the k3d Built-in Registry. NOT the same as the Runner Image.

**ServiceAccount (runner)** — The Kubernetes ServiceAccount the Runner Pod uses. Lives in the `arc-runners` namespace. Bound to an RBAC Role in the `app` namespace granting apply-only verbs (`get, list, create, patch` on `deployments`, `services`; `get, list` on `pods`; `create` on `pods/exec`). **No `delete`.**

**Namespace `arc-systems`** — Long-lived ARC controller + `AutoscalingListener`. Holds the GitHub App credentials Secret. See ADR-0004.

**Namespace `arc-runners`** — Ephemeral Runner Pods. No long-lived resources.

**Namespace `app`** — Deployed Go app. The only namespace the runner pod's ServiceAccount has RBAC for. See ADR-0004.

**Listener Mode** — The runner pool is configured `runnerListenerType: polling`, meaning the listener polls GitHub's Actions API itself rather than receiving pushes via an in-cluster Service. See ADR-0006.

**Kustomize Overlay** — The Go app's manifests live under `k8s/base/` and `k8s/overlays/dev/`. The overlay's `images:` transformer rewrites the registry host to the k3d built-in registry. `kubectl apply -k k8s/overlays/dev` is the deploy command. See ADR-0005.

**Workflow** — A YAML file under `.github/workflows/` that GitHub Actions executes. The project has at least one workflow: build → test → deploy → verify.

**App / Go App / Target Application** — The small `net/http` Go binary the workflow deploys. Exposes `/` and `/healthz`. The word "app" in this project always means this binary; not the runner itself, not ARC, not anything else.

**Health Check (`/healthz`)** — The endpoint the verification step curls to confirm the deployment succeeded. See ADR-0007 for the exact contract.

**Healthz Endpoint Contract** — `/healthz` returns HTTP 200 with body exactly `ok`. Both conditions are checked; either failing fails the workflow. See ADR-0007.

**Verification Step** — The final step of the workflow. Runs a single `curl` from the Runner Pod to `http://kube-runner-app.app.svc.cluster.local:8080/healthz`, asserts both status 200 and body `ok`. The single source of truth for "is the deployment successful."

## Out of scope (for v1)

- Multi-cluster deployments (target cluster == runner cluster)
- Multi-repo / multi-org runner pools (one repo: `github.com/minhajul/kube-runner`)
- Persistent runners, runner caching beyond what ARC provides
- Production hardening of the runner image (non-root, distroless, supply-chain signing) — recorded as a follow-up
- Canary / blue-green rollouts — plain rolling Deployment is enough for v1

## Open decisions

See `docs/adr/` for decisions that have been committed to. Anything not in an ADR is still open.