# k3s (D4)

A0 is **Ampere**, so production moves from Compose+Caddy to **single-node k3s** on the same VM (`137.131.14.30`, `https://shashank-dropdesk.duckdns.org`). Postgres stays ClusterIP-only. Two API replicas. cert-manager issues Let’s Encrypt. Grafana Cloud Alloy scrapes `api.deskdrop.svc:8080` in-cluster.

Manifests: [`deploy/k8s/`](../../deploy/k8s/). Cutover: [`docs/runbooks/migrate-compose-to-k3s.md`](../runbooks/migrate-compose-to-k3s.md). Decision: [ADR 0008](../adr/0008-compose-to-k3s.md).

## Layout

| Object | Notes |
|---|---|
| `deploy/postgres` | PVC 8Gi, `postgres:16-alpine`, Service ClusterIP `:5432` |
| `job/migrate` | `migrate/migrate` + ConfigMap of `migrations/` |
| `deploy/api` | 2 replicas, GHCR `api:v2`, `/healthz` `/readyz` |
| `deploy/dashboard` | GHCR `dashboard:v2` |
| Ingress | Traefik; `/v1` `/healthz` `/readyz` → API; `/` → dashboard; `/metrics` basic auth |
| NetworkPolicy | Postgres ingress only from `app=api` and `app=migrate` |
| `deploy/alloy` | optional; Grafana Cloud remote_write |

Secrets are created on the box (`kubectl -n deskdrop get secret deskdrop`). They are not in git.

## Install / apply (after cutover)

```bash
ssh -i ~/Downloads/ssh-key-2026-10-10.key ubuntu@137.131.14.30
sudo k3s kubectl -n deskdrop get pods
sudo k3s kubectl apply -k ~/deskdrop/deploy/k8s
sudo k3s kubectl apply -f ~/deskdrop/deploy/k8s/ingress.yaml
sudo k3s kubectl -n deskdrop rollout restart deploy/api
sudo k3s kubectl -n deskdrop get certificate
```

## Cutover

One-time, on the VM, repo at `~/deskdrop` matching this commit:

```bash
# METRICS_PASSWORD must be set in deploy/prod/.env (plaintext)
chmod +x scripts/k3s-cutover.sh
./scripts/k3s-cutover.sh
```

Expect a few minutes without HTTPS while Compose stops and Traefik binds 80/443. DuckDNS does not change. User row count must match before Compose is torn down.

## Verify or demo

```bash
curl -sfS https://shashank-dropdesk.duckdns.org/healthz
curl -sfS https://shashank-dropdesk.duckdns.org/readyz
sudo k3s kubectl -n deskdrop get pods
# all Ready, including api 2/2
sudo k3s kubectl -n deskdrop get certificate deskdrop-tls
# Ready=True
sudo k3s kubectl -n deskdrop get svc postgres -o jsonpath='{.spec.type}{"\n"}'
# ClusterIP
sudo k3s kubectl -n deskdrop delete pod -l app=api --field-selector=status.phase=Running | head
# share from the phone still delivers while one API pod is replaced
```

Postgres must not appear in `sudo k3s kubectl get svc -A | grep NodePort` for port 5432.

## Failure modes

- Empty-state k3s Postgres (no restore) loses pairing rows. Cutover dumps Compose first and aborts if `SELECT count(*) FROM users` changes.
- `update.k3s.io` / `get.k3s.io` may be unreachable from OCI; the script installs a pinned GitHub release (`K3S_VERSION`).
- Traefik vs Caddy on 80/443: install k3s with Traefik **disabled** until `compose down`.
- `apply -k` before Traefik exists: Ingress/Middleware CRDs missing. Ingress is applied after Traefik is enabled.
- kubectl kustomize cannot load files outside `deploy/k8s/`; SQL copies live in `deploy/k8s/migrations/` (keep in sync with `migrations/`).
- GHCR `:v2` is **linux/arm64** (CI `ubuntu-24.04-arm`). amd64-only images CrashLoop with `exec format error` on Ampere.
- `/metrics` 401: Traefik basic auth. Alloy does not need that path.
- 1 OCPU / 6 GB: keep Grafana in Cloud, not in-cluster Grafana.
- Per-pod rate limit is 2× after this ([ADR 0006](0006-per-pod-rate-limit.md)).
