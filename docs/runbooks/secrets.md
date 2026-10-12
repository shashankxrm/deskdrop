# Runbook: secret injection (D5)

Secrets live in env or a Kubernetes Secret. They are never in git. Missing `DEVICE_TOKEN_PEPPER` or `DATABASE_URL` is a hard API exit ([cmd/api](../../cmd/api/main.go)).

## What is a secret

| Name | Where it is injected |
|---|---|
| `DEVICE_TOKEN_PEPPER` | API process |
| `DATABASE_URL` / `POSTGRES_PASSWORD` | API + Postgres |
| DuckDNS token | `infra/terraform.tfvars` (gitignored) |
| OCI API key | `~/.oci/oci_api_key.pem` |
| `deploy/prod/.env` | VM only, mode `600` |
| k3s Secret `deskdrop` | namespace `deskdrop` |
| Grafana Cloud token | same Secret / `.env` |
| GitHub `DEPLOY_SSH_KEY` | Actions only |

`.env.example` and `deploy/prod/env.example` stay empty. Local Compose may set `compose-local-pepper` in [`deploy/compose/compose.yaml`](../../deploy/compose/compose.yaml); production must not.

## Local

```bash
cp .env.example .env
# fill DEVICE_TOKEN_PEPPER and DATABASE_URL if you run the API on the host
chmod 600 .env
```

`docker compose up` does not need a root `.env` (A2 injects its own).

## Production Compose (rollback)

On the VM only:

```bash
cp deploy/prod/env.example deploy/prod/.env
openssl rand -hex 32   # POSTGRES_PASSWORD
openssl rand -hex 32   # DEVICE_TOKEN_PEPPER
chmod 600 deploy/prod/.env
# never rsync this file; CI excludes it
```

## k3s

Create or replace the Secret from the same values (do not commit the literals):

```bash
# on the VM, with deploy/prod/.env already filled
set -a
source ~/deskdrop/deploy/prod/.env
set +a
sudo k3s kubectl -n deskdrop create secret generic deskdrop \
  --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  --from-literal=DATABASE_URL="postgres://deskdrop:${POSTGRES_PASSWORD}@postgres:5432/deskdrop?sslmode=disable" \
  --from-literal=DEVICE_TOKEN_PEPPER="$DEVICE_TOKEN_PEPPER" \
  --from-literal=GRAFANA_CLOUD_RW_URL="${GRAFANA_CLOUD_RW_URL:-}" \
  --from-literal=GRAFANA_CLOUD_USER="${GRAFANA_CLOUD_USER:-}" \
  --from-literal=GRAFANA_CLOUD_TOKEN="${GRAFANA_CLOUD_TOKEN:-}" \
  --dry-run=client -o yaml | sudo k3s kubectl apply -f -
sudo k3s kubectl -n deskdrop rollout restart deploy/api
```

Pods that start without this Secret stay unready. `kubectl apply -k` does not create the Secret.

## GitHub Actions deploy

Repo Settings → Secrets: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`. Empty values skip deploy (job still green). They never belong in the workflow YAML.

## Rotate after a leak

1. Revoke the device in the dashboard (token).
2. New pepper: every existing token_hash becomes useless; users must pair again. Generate a new `DEVICE_TOKEN_PEPPER`, patch the Secret / `.env`, restart API.
3. New `POSTGRES_PASSWORD`: update Secret and Postgres user, or restore from D6.

## Verify

```bash
./scripts/check-secrets.sh
DEVICE_TOKEN_PEPPER= DATABASE_URL= go run ./cmd/api ; echo exit:$?
# non-zero; log missing required env
git ls-files '*.env' '*.tfvars' 'infra/backend.hcl' || true
# empty
```
