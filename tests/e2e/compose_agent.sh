#!/usr/bin/env bash
# C3: drive the agent binary against compose. Assert one open + one delivery.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
BASE="${DESKDROP_API:-http://127.0.0.1:8080}"

jget() {
  python3 -c 'import json,sys
d=json.load(sys.stdin)
cur=d
for k in sys.argv[1].split("."):
    if isinstance(cur, list):
        cur=cur[int(k)]
    else:
        cur=cur[k]
if cur is None:
    print("")
else:
    print(cur)
' "$1"
}

wait_ready() {
  local i
  for i in $(seq 1 60); do
    if curl -sfS "$BASE/readyz" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "readyz never 200 on $BASE" >&2
  return 1
}

echo "waiting for $BASE/readyz"
wait_ready

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

go build -o "$TMP/deskdrop-agent" ./cmd/agent

curl -sS -o "$TMP/b0" -X POST "$BASE/v1/pairings" -H 'Content-Type: application/json' -d '{}'
PHONE_CODE="$(jget code <"$TMP/b0")"
curl -sS -o "$TMP/b1" -X POST "$BASE/v1/pairings/claim" \
  -H 'Content-Type: application/json' \
  -d "{\"code\":\"$PHONE_CODE\",\"type\":\"phone\",\"name\":\"e2e-phone-agent\"}"
PHONE="$(jget token <"$TMP/b1")"

curl -sS -o "$TMP/b2" -X POST "$BASE/v1/pairings" -H "Authorization: Bearer $PHONE"
LAP_CODE="$(jget code <"$TMP/b2")"

"$TMP/deskdrop-agent" pair -api "$BASE" -code "$LAP_CODE" -name e2e-agent-mac -config "$TMP/config.json"
python3 -c 'import os,stat,sys
p=sys.argv[1]
m=stat.S_IMODE(os.stat(p).st_mode)
assert m==0o600, oct(m)
d=stat.S_IMODE(os.stat(os.path.dirname(p)).st_mode)
assert d==0o700, oct(d)
' "$TMP/config.json"

KEY="e2e-agent-$(date +%s)-$$"
URL="https://example.com/agent-$KEY"
curl -sS -o "$TMP/b4" -D "$TMP/h4" -X POST "$BASE/v1/links" \
  -H "Authorization: Bearer $PHONE" \
  -H "Idempotency-Key: $KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"url\":\"$URL\",\"title\":\"agent-e2e\"}"
LINK_ID="$(jget id <"$TMP/b4")"
test -n "$LINK_ID"

OPENED="$TMP/opened.txt"
"$TMP/deskdrop-agent" run -config "$TMP/config.json" -once -open-file "$OPENED"
test "$(cat "$OPENED")" = "$URL"

# Relaunch must not open again (already acked).
"$TMP/deskdrop-agent" run -config "$TMP/config.json" -once -open-file "$OPENED"
test "$(wc -l <"$OPENED" | tr -d ' ')" = "1"

N="$(docker compose exec -T postgres psql -U deskdrop -d deskdrop -tAc \
  "SELECT count(*) FROM link_deliveries WHERE link_id = '$LINK_ID'")"
test "$N" = "1"
ACKED="$(docker compose exec -T postgres psql -U deskdrop -d deskdrop -tAc \
  "SELECT count(*) FROM link_deliveries WHERE link_id = '$LINK_ID' AND acked_at IS NOT NULL")"
test "$ACKED" = "1"

echo "e2e agent ok link_id=$LINK_ID opened=1"
