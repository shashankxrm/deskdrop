#!/usr/bin/env bash
# Dump k3s Postgres and upload gzip to OCI Object Storage (S3-compat).
# Run on the Ampere VM. Does not talk to Postgres over the cluster network
# (NetworkPolicy: only API/migrate). Uses kubectl exec.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${DESKDROP_ENV:-$ROOT/deploy/prod/.env}"
NS="${K8S_NS:-deskdrop}"
K=(sudo k3s kubectl)

if [[ ! -f "$ENV_FILE" ]]; then
  echo "missing $ENV_FILE" >&2
  exit 1
fi
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

: "${OCI_S3_ACCESS_KEY:?set OCI_S3_ACCESS_KEY in deploy/prod/.env}"
: "${OCI_S3_SECRET_KEY:?set OCI_S3_SECRET_KEY in deploy/prod/.env}"
: "${OCI_S3_NAMESPACE:?set OCI_S3_NAMESPACE (oci os ns get)}"
OCI_S3_REGION="${OCI_S3_REGION:-us-phoenix-1}"
OCI_S3_BUCKET="${OCI_S3_BUCKET:-deskdrop-backups}"

command -v aws >/dev/null || {
  echo "install awscli: sudo apt-get install -y awscli" >&2
  exit 1
}

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
KEY="pg/deskdrop-${STAMP}.sql.gz"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

echo "dumping deskdrop…" >&2
"${K[@]}" -n "$NS" exec deploy/postgres -- \
  pg_dump -U deskdrop --clean --if-exists --no-owner deskdrop | gzip -c >"$TMP"

export AWS_ACCESS_KEY_ID="$OCI_S3_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$OCI_S3_SECRET_KEY"
export AWS_DEFAULT_REGION="$OCI_S3_REGION"
ENDPOINT="https://${OCI_S3_NAMESPACE}.compat.objectstorage.${OCI_S3_REGION}.oraclecloud.com"

aws s3 cp "$TMP" "s3://${OCI_S3_BUCKET}/${KEY}" \
  --endpoint-url "$ENDPOINT" \
  --region "$OCI_S3_REGION" >&2

BYTES="$(wc -c <"$TMP" | tr -d ' ')"
echo "uploaded s3://${OCI_S3_BUCKET}/${KEY} bytes=${BYTES}" >&2
echo "$KEY"
