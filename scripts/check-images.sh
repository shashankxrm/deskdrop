#!/usr/bin/env bash
# D2: image is non-root and does not ship .env or token files.
set -euo pipefail

fail=0
for img in "$@"; do
  user="$(docker inspect -f '{{.Config.User}}' "$img")"
  if [ -z "$user" ] || [ "$user" = "0" ] || [ "$user" = "root" ] || [ "$user" = "0:0" ]; then
    echo "FAIL $img user=$user (want non-root)" >&2
    fail=1
  else
    echo "ok $img user=$user"
  fi
  cid="$(docker create "$img")"
  if docker export "$cid" | tar t | grep -E '(^|/)\.env$|(^|/)DEVICE_TOKEN|\.pem$'; then
    echo "FAIL $img contains secret-like files" >&2
    fail=1
  else
    echo "ok $img no .env/token files in layers"
  fi
  docker rm "$cid" >/dev/null
done
exit "$fail"
