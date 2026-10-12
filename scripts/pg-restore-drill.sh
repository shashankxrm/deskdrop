#!/usr/bin/env bash
# Safe restore drill: dump → Object Storage → restore into deskdrop_restore
# (not the live DB) → compare user counts → drop the sidecar database.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
NS="${K8S_NS:-deskdrop}"
K=(sudo k3s kubectl)
export DESKDROP_ENV="${DESKDROP_ENV:-$ROOT/deploy/prod/.env}"

KEY="$("$ROOT/scripts/pg-backup.sh")"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

ENV_FILE="$DESKDROP_ENV"
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a
export AWS_ACCESS_KEY_ID="$OCI_S3_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$OCI_S3_SECRET_KEY"
export AWS_DEFAULT_REGION="${OCI_S3_REGION:-us-phoenix-1}"
ENDPOINT="https://${OCI_S3_NAMESPACE}.compat.objectstorage.${AWS_DEFAULT_REGION}.oraclecloud.com"
BUCKET="${OCI_S3_BUCKET:-deskdrop-backups}"

aws s3 cp "s3://${BUCKET}/${KEY}" "$TMP" --endpoint-url "$ENDPOINT"

LIVE="$("${K[@]}" -n "$NS" exec deploy/postgres -- psql -U deskdrop -d deskdrop -tAc 'SELECT count(*) FROM users')"
"${K[@]}" -n "$NS" exec deploy/postgres -- \
  psql -U deskdrop -d postgres -v ON_ERROR_STOP=1 \
  -c "DROP DATABASE IF EXISTS deskdrop_restore;" \
  -c "CREATE DATABASE deskdrop_restore OWNER deskdrop;"
gzip -dc "$TMP" | "${K[@]}" -n "$NS" exec -i deploy/postgres -- \
  psql -U deskdrop -d deskdrop_restore -v ON_ERROR_STOP=1 >/dev/null
DRILL="$("${K[@]}" -n "$NS" exec deploy/postgres -- psql -U deskdrop -d deskdrop_restore -tAc 'SELECT count(*) FROM users')"
"${K[@]}" -n "$NS" exec deploy/postgres -- \
  psql -U deskdrop -d postgres -v ON_ERROR_STOP=1 \
  -c "DROP DATABASE deskdrop_restore;"

echo "live_users=$LIVE restore_users=$DRILL object=$KEY"
if [[ "$LIVE" != "$DRILL" ]]; then
  echo "FAIL: user count mismatch" >&2
  exit 1
fi
WHEN="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "drill_ok $WHEN"
echo "Paste into docs/runbooks/restore-postgres.md:"
echo "| ${WHEN} | ${LIVE} | ${KEY} | sidecar DB deskdrop_restore dropped |"
