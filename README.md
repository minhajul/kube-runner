# kube-runner

Self-hosted GitHub Actions runner inside a Kubernetes cluster, used to
**deploy and verify a small Go application** on the same cluster via
CI/CD.

A push to `main` triggers `.github/workflows/deploy.yml`, which runs on
a Runner Pod spawned inside the cluster. The runner builds the Go app,
pushes the image to the cluster's built-in registry, applies a Kustomize
overlay, waits for rollout, and then asserts the deployment is healthy.

The end state is a closed loop: the runner that builds the app *is*
itself a pod in the cluster the app runs in.

## Status

**Code is in place; the loop has not yet been run end-to-end.** Every
file in this repo is a buildable starting point. The bootstrap steps
below have not been executed against a real cluster as of this commit,
and several of them (chart versions, image tags, GitHub App permissions)
will need verification against current ARC documentation before they
work on your machine.

Treat this README as a **checklist**, not a verified procedure. If a
step fails, the failure is in the procedure, not your setup.

## Architecture

| Layer | Choice | Decision |
|---|---|---|
| Runner architecture | GitHub Actions Runner Controller (ARC), ephemeral pods, split chart | [ADR-0001](docs/adr/0001-arc-split-chart.md), [ADR-0006](docs/adr/0006-runner-listener-polling.md) |
| Runner image | Tooling overlay on `ghcr.io/actions/actions-runner:latest` | [ADR-0002](docs/adr/0002-runner-image-overlay.md) |
| Cluster | k3d with built-in registry | [ADR-0003](docs/adr/0003-image-registry.md) |
| Namespaces | `arc-systems`, `arc-runners`, `app` | [ADR-0004](docs/adr/0004-namespace-layout.md) |
| App manifests | Kustomize overlays | [ADR-0005](docs/adr/0005-kustomize-for-go-app.md) |
| Verification | `GET /healthz` → 200 + body `ok` | [ADR-0007](docs/adr/0007-healthz-contract.md) |
| RBAC | Apply-only Role in `app` namespace; ServiceAccount wired into Runner Pod | [CONTEXT.md](CONTEXT.md) §ServiceAccount (runner) |

The glossary in [CONTEXT.md](CONTEXT.md) defines the project's vocabulary
("runner", "ephemeral", "listener", etc.). Read it if any term below is
ambiguous.

## Repository layout

```
.
├── app/                      # Go application source
│   ├── main.go               # /  and  /healthz
│   ├── main_test.go          # TestHealthzContract pins the ADR-0007 contract
│   ├── go.mod
│   └── Dockerfile            # multi-stage build → distroless/static
├── arc/                      # ARC install + RBAC
│   ├── namespaces.yaml
│   ├── controller-values.yaml
│   ├── runner-scale-set-values.yaml
│   ├── runner-pod-rbac.yaml  # ServiceAccount, Role, RoleBinding
│   └── github-app-secret.example.yaml
├── k8s/                      # Kustomize manifests for the deployed Go app
│   ├── base/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── kustomization.yaml
│   └── overlays/
│       └── dev/
│           └── kustomization.yaml    # registry host lives here (ADR-0005)
├── .github/
│   └── workflows/
│       └── deploy.yml        # push-to-main, single job, ends in verification curl
├── Dockerfile.runner         # FROM ghcr.io/actions/actions-runner:latest, +Go/Docker/kubectl
├── docs/adr/                 # 0001..0007 — the design decisions
├── CONTEXT.md                # project glossary
└── README.md                 # this file
```

## Prerequisites

You will need, locally:

- **macOS or Linux** with Docker installed.
- **Go 1.25** if you want to run the unit tests locally.
- **k3d** ≥ 5.x — `brew install k3d` or [install script](https://k3d.io).
- **kubectl** ≥ 1.27 (any recent build).
- **Helm** ≥ 3.x.
- **Docker CLI access** to `k3d-kube-runner-registry.localhost:5000`
  (this works out of the box once the cluster is up).

You will need, on GitHub:

- A GitHub App with these permissions, **installed on this repository**:
  - Repository → **Administration: Read & Write**
  - Repository → **Actions: Read & Write**
  - Metadata: Read-only (default)
- The App's numeric `app_id`, the `installation_id` for this repo, and
  the App's private key (a PEM file).

The App's webhook does **not** need to be configured — the listener
uses polling ([ADR-0006](docs/adr/0006-runner-listener-polling.md)).

## Bootstrap (step-by-step)

### 1. Create the k3d cluster with a built-in registry

The registry is created *with* the cluster; it cannot be added later.

```bash
k3d cluster create kube-runner \
  --registry-create kube-runner \
  --agents 1
```

Confirm `kubectl cluster-info` resolves and `docker ps` shows a
container named `k3d-kube-runner-registry`.

### 2. Apply namespaces

```bash
kubectl apply -f arc/namespaces.yaml
```

You should now have `arc-systems`, `arc-runners`, and `app`.

### 3. Create the GitHub App credentials Secret

Get from the GitHub App settings page:

- App ID (numeric)
- Installation ID for this repo (numeric; found at the end of the
  "Install App" URL or via the App's installation list)
- The private key PEM file

Then:

```bash
kubectl create secret generic gha-rs-github-secret \
  --namespace=arc-systems \
  --from-literal=github_app_id="<APP_ID>" \
  --from-literal=github_app_installation_id="<INSTALLATION_ID>" \
  --from-file=github_app_private_key=/path/to/private-key.pem
```

Verify:

```bash
kubectl get secret gha-rs-github-secret -n arc-systems
```

The PEM file is sensitive. The `.gitignore` already excludes
`*github-app-secret*.yaml` (real Secret YAML) and `*.pem`. The
`arc/github-app-secret.example.yaml` file in the repo is a template
only; its values are placeholders.

### 4. Build and push the custom runner image

The workflow relies on a custom runner image (Go + Docker CLI + kubectl)
built on top of `ghcr.io/actions/actions-runner:latest`.

```bash
docker build -f Dockerfile.runner \
  -t ghcr.io/minhajul/kube-runner-runner:dev .
docker push ghcr.io/minhajul/kube-runner-runner:dev
```

If you prefer a different registry or tag, update
`arc/runner-scale-set-values.yaml` (`runnerImage:`) and re-helm.

### 5. Install ARC

Two Helm charts, in order. Check
[the ARC releases page](https://github.com/actions/actions-runner-controller/releases)
for the current chart versions before running.

```bash
# Controller (the CRD reconciler)
helm install arc-controller \
  oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set-controller \
  --version <CHART_VERSION> \
  --namespace arc-systems \
  --create-namespace \
  --values arc/controller-values.yaml

# Runner scale set (the Listener + RunnerSet pair)
helm install runner-scale-set \
  oci://ghcr.io/actions/actions-runner-controller-charts/gha-runner-scale-set-runner-scale-set \
  --version <CHART_VERSION> \
  --namespace arc-systems \
  --values arc/runner-scale-set-values.yaml
```

Watch the listener come up:

```bash
kubectl get pods -n arc-systems -w
```

You should see one `arc-controller-*` pod and one
`runner-scale-set-*` listener pod go to `Running`. The Listener will
register itself with GitHub; a `kubectl logs -n arc-systems
runner-scale-set-<hash>` should show GitHub API polling activity.

### 6. Wire up the runner pod's RBAC

The Role + RoleBinding + ServiceAccount were committed in
`arc/runner-pod-rbac.yaml`. The ServiceAccount *must* be created
*before* the chart starts spawning Runner Pods, otherwise pods from the
first job will use a default SA and the RBAC will not apply.

```bash
kubectl apply -f arc/runner-pod-rbac.yaml
```

The chart's `template.spec.serviceAccountName` in
`arc/runner-scale-set-values.yaml` already points at this SA, so new
Runner Pods will pick it up automatically.

### 7. Push to `main`

That's it. The next push to `main` will:

1. Wake the ARC listener (polling, ~30 s wake-up).
2. Spawn one Runner Pod in `arc-runners`.
3. The pod checks out this repo, runs `go test ./...` in `app/`,
   builds the app image, pushes it to
   `k3d-kube-runner-registry.localhost:5000/kube-runner-app:<sha>`.
4. The pod copies the dev overlay to a tmpdir, stamps the SHA into the
   image tag, and runs `kubectl apply -k`.
5. The pod waits for the Deployment to roll out (120 s timeout).
6. The pod curls `http://kube-runner-app.app.svc.cluster.local:8080/healthz`
   and asserts 200 + body `ok`.

If the workflow run is green, the loop is closed.

## Local development

Run the unit tests:

```bash
go -C app test ./...
```

Render the Kustomize overlay (without applying):

```bash
kubectl kustomize k8s/overlays/dev
```

Manually apply after stamping a SHA (mirrors what the workflow does):

```bash
tmp="$(mktemp -d)"
cp -r k8s/base "$tmp/base"
cp -r k8s/overlays/dev/. "$tmp/"
sed -i.bak "s|newTag: dev$|newTag: <SHA>|" "$tmp/kustomization.yaml"
rm "$tmp/kustomization.yaml.bak"
kubectl apply -k "$tmp"
rm -rf "$tmp"
```

## What this README is *not*

- It is **not** a verified procedure. Every step has been reasoned
  through but not run end-to-end against a cluster as of this commit.
- It is **not** a security review. The runner image runs as `runner`
  (the official image's default user); the deployment runs as
  `nonroot:nonroot` with `readOnlyRootFilesystem: true`. RBAC is
  apply-only. None of this has been audited.
- It is **not** a production guide. Persistent runners, multi-repo
  pools, image-signing, and NetworkPolicies are all out of scope for
  v1 (see [CONTEXT.md](CONTEXT.md) §Out of scope).

## Troubleshooting

The first job usually fails for one of these reasons:

- **`secret "gha-rs-github-secret" not found`** — the Secret in step 3
  wasn't created before the chart was installed. Reapply the Secret and
  the listener will reconnect.
- **`ImagePullBackOff` on the runner pod** — the `runnerImage:` in
  `arc/runner-scale-set-values.yaml` doesn't exist. Re-push in step 4.
- **`ImagePullBackOff` on the deployed app** — the k3d registry
  container isn't running, or the hostname doesn't resolve. Check
  `docker ps | grep k3d-kube-runner-registry` and confirm
  `k3d-kube-runner-registry.localhost` resolves from inside a pod.
- **`Forbidden` on `kubectl apply` in the workflow** — the SA wasn't
  wired into the runner pod. Re-check `template.spec.serviceAccountName`
  in `arc/runner-scale-set-values.yaml` and re-helm.
- **`/healthz` returns 5xx in the verification step** — usually a typo
  in the verification URL, or the Service selector not matching the
  Pod labels. `kubectl get pods -n app --show-labels` should show
  `app.kubernetes.io/name=kube-runner-app` on the app pod.

## See also

- [CONTEXT.md](CONTEXT.md) — glossary
- [docs/adr/](docs/adr/) — architectural decisions 0001..0007
- [`.github/workflows/deploy.yml`](.github/workflows/deploy.yml) — the
  workflow itself, with comments referencing the ADRs it implements