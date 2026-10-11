#!/usr/bin/env bash
# Run on the Ampere VM as ubuntu from ~/deskdrop. Stops Compose, restores
# Postgres into k3s, then Traefik+cert-manager own 80/443. DuckDNS stays.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE=(docker compose -f "$ROOT/deploy/prod/compose.yaml" --env-file "$ROOT/deploy/prod/.env")
K=(sudo k3s kubectl)
NS=deskdrop
HOST=shashank-dropdesk.duckdns.org
ENV_FILE="$ROOT/deploy/prod/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing $ENV_FILE" >&2
  exit 1
fi
# shellcheck disable=SC1090
set -a
# shellcheck source=/dev/null
source "$ENV_FILE"
set +a
: "${POSTGRES_PASSWORD:?}"
: "${DEVICE_TOKEN_PEPPER:?}"
: "${CADDY_EMAIL:?}"
: "${METRICS_USER:=scrape}"
if [[ -z "${METRICS_PASSWORD:-}" ]]; then
  echo "set METRICS_PASSWORD in deploy/prod/.env (plaintext for Traefik basic auth)" >&2
  exit 1
fi

need() { command -v "$1" >/dev/null || { echo "need $1" >&2; exit 1; }; }
need docker
need curl
need openssl

USERS_BEFORE="$("${COMPOSE[@]}" exec -T postgres psql -U deskdrop -d deskdrop -tAc 'SELECT count(*) FROM users')"
echo "users before: $USERS_BEFORE"

if [[ ! -x /usr/local/bin/k3s ]]; then
  curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik" sh -
fi
sudo mkdir -p /home/ubuntu/.kube
sudo cp /etc/rancher/k3s/k3s.yaml /home/ubuntu/.kube/config
sudo chown ubuntu:ubuntu /home/ubuntu/.kube/config
chmod 600 /home/ubuntu/.kube/config

"${K[@]}" apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.2/cert-manager.yaml
"${K[@]}" -n cert-manager rollout status deploy/cert-manager --timeout=180s
"${K[@]}" -n cert-manager rollout status deploy/cert-manager-webhook --timeout=180s
"${K[@]}" -n cert-manager rollout status deploy/cert-manager-cainjector --timeout=180s

"${K[@]}" create namespace "$NS" --dry-run=client -o yaml | "${K[@]}" apply -f -

HTPASSWD="${METRICS_USER}:$(openssl passwd -apr1 "$METRICS_PASSWORD")"

"${K[@]}" -n "$NS" create secret generic deskdrop \
  --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  --from-literal=DATABASE_URL="postgres://deskdrop:${POSTGRES_PASSWORD}@postgres:5432/deskdrop?sslmode=disable" \
  --from-literal=DEVICE_TOKEN_PEPPER="$DEVICE_TOKEN_PEPPER" \
  --from-literal=GRAFANA_CLOUD_RW_URL="${GRAFANA_CLOUD_RW_URL:-}" \
  --from-literal=GRAFANA_CLOUD_USER="${GRAFANA_CLOUD_USER:-}" \
  --from-literal=GRAFANA_CLOUD_TOKEN="${GRAFANA_CLOUD_TOKEN:-}" \
  --dry-run=client -o yaml | "${K[@]}" apply -f -

"${K[@]}" -n "$NS" create secret generic metrics-auth \
  --from-literal=users="$HTPASSWD" \
  --dry-run=client -o yaml | "${K[@]}" apply -f -

# Freeze writes, dump compose Postgres, restore into k3s.
"${COMPOSE[@]}" stop api dashboard caddy alloy || true
DUMP=/tmp/deskdrop-cutover.sql
"${COMPOSE[@]}" exec -T postgres pg_dump -U deskdrop --clean --if-exists deskdrop >"$DUMP"

"${K[@]}" apply -f "$ROOT/deploy/k8s/postgres.yaml"
"${K[@]}" -n "$NS" rollout status deploy/postgres --timeout=180s
"${K[@]}" -n "$NS" exec -i deploy/postgres -- psql -U deskdrop -d deskdrop <"$DUMP"
"${K[@]}" -n "$NS" delete job migrate --ignore-not-found
"${K[@]}" apply -k "$ROOT/deploy/k8s"
"${K[@]}" -n "$NS" wait --for=condition=complete job/migrate --timeout=180s
"${K[@]}" -n "$NS" rollout status deploy/api --timeout=180s
"${K[@]}" -n "$NS" rollout status deploy/dashboard --timeout=180s

USERS_AFTER="$("${K[@]}" -n "$NS" exec deploy/postgres -- psql -U deskdrop -d deskdrop -tAc 'SELECT count(*) FROM users')"
echo "users after restore: $USERS_AFTER"
if [[ "$USERS_BEFORE" != "$USERS_AFTER" ]]; then
  echo "user count changed ($USERS_BEFORE -> $USERS_AFTER); stopping before compose down" >&2
  exit 1
fi

"${COMPOSE[@]}" down
# Traefik needs 80/443. Reinstall k3s without --disable=traefik (keeps data).
curl -sfL https://get.k3s.io | sh -
"${K[@]}" apply -k "$ROOT/deploy/k8s"
"${K[@]}" apply -f "$ROOT/deploy/k8s/ingress.yaml"
sed "s/ACME_EMAIL_PLACEHOLDER/${CADDY_EMAIL}/" "$ROOT/deploy/k8s/cert-issuer.yaml" | "${K[@]}" apply -f -

if [[ -n "${GRAFANA_CLOUD_TOKEN:-}" ]]; then
  "${K[@]}" apply -f "$ROOT/deploy/k8s/alloy.yaml"
fi

echo "waiting for cert..."
for i in $(seq 1 36); do
  READY="$("${K[@]}" -n "$NS" get certificate deskdrop-tls -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
  if [[ "$READY" == True ]]; then
    break
  fi
  sleep 5
done

curl -fsS --retry 12 --retry-delay 5 "https://$HOST/healthz"
echo
"${K[@]}" -n "$NS" get pods
echo "cutover ok users=$USERS_AFTER"
