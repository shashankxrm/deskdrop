#!/usr/bin/env bash
# B7: pair phone + laptop over HTTP, share, pull, ack once. Compose must be up.
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

hdr_code() {
  python3 -c 'import sys
lines=sys.stdin.read().splitlines()
code=""
for ln in lines:
    if ln.startswith("HTTP/"):
        parts=ln.split()
        if len(parts)>=2:
            code=parts[1]
print(code)
'
}

req_id() {
  python3 -c 'import sys
for ln in sys.stdin.read().splitlines():
    if ln.lower().startswith("x-request-id:"):
        print(ln.split(":",1)[1].strip())
        break
'
}

echo "waiting for $BASE/readyz"
wait_ready

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# 0. Bootstrap phone
curl -sS -D "$TMP/h0" -o "$TMP/b0" -X POST "$BASE/v1/pairings" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-Id: e2e-pair-phone' \
  -d '{"userId":"ignored"}'
test "$(hdr_code <"$TMP/h0")" = "201"
test "$(req_id <"$TMP/h0")" = "e2e-pair-phone"
PHONE_CODE="$(jget code <"$TMP/b0")"

curl -sS -D "$TMP/h1" -o "$TMP/b1" -X POST "$BASE/v1/pairings/claim" \
  -H 'Content-Type: application/json' \
  -d "{\"code\":\"$PHONE_CODE\",\"type\":\"phone\",\"name\":\"e2e-phone\"}"
test "$(hdr_code <"$TMP/h1")" = "201"
PHONE="$(jget token <"$TMP/b1")"

# Extra laptop on the same user
curl -sS -D "$TMP/h2" -o "$TMP/b2" -X POST "$BASE/v1/pairings" \
  -H "Authorization: Bearer $PHONE"
test "$(hdr_code <"$TMP/h2")" = "201"
LAP_CODE="$(jget code <"$TMP/b2")"

curl -sS -D "$TMP/h3" -o "$TMP/b3" -X POST "$BASE/v1/pairings/claim" \
  -H 'Content-Type: application/json' \
  -d "{\"code\":\"$LAP_CODE\",\"type\":\"laptop\",\"name\":\"e2e-mac\"}"
test "$(hdr_code <"$TMP/h3")" = "201"
LAPTOP="$(jget token <"$TMP/b3")"

KEY="e2e-$(date +%s)-$$"
URL="https://example.com/e2e-$KEY"

# 1–2. Share (userId ignored)
curl -sS -D "$TMP/h4" -o "$TMP/b4" -X POST "$BASE/v1/links" \
  -H "Authorization: Bearer $PHONE" \
  -H "Idempotency-Key: $KEY" \
  -H 'Content-Type: application/json' \
  -H 'X-Request-Id: e2e-share' \
  -d "{\"url\":\"$URL\",\"title\":\"e2e\",\"userId\":\"attacker\"}"
test "$(hdr_code <"$TMP/h4")" = "201"
test "$(req_id <"$TMP/h4")" = "e2e-share"
LINK_ID="$(jget id <"$TMP/b4")"
test "$(jget url <"$TMP/b4")" = "$URL"

# Replay: 200, same id, still one links row
curl -sS -D "$TMP/h5" -o "$TMP/b5" -X POST "$BASE/v1/links" \
  -H "Authorization: Bearer $PHONE" \
  -H "Idempotency-Key: $KEY" \
  -H 'Content-Type: application/json' \
  -d "{\"url\":\"$URL\",\"title\":\"e2e\"}"
test "$(hdr_code <"$TMP/h5")" = "200"
test "$(jget id <"$TMP/b5")" = "$LINK_ID"

# 3. Pull (twice): one pending, pull does not ack
curl -sS -D "$TMP/h6" -o "$TMP/b6" "$BASE/v1/links/pending" \
  -H "Authorization: Bearer $LAPTOP"
test "$(hdr_code <"$TMP/h6")" = "200"
test "$(jget links.0.id <"$TMP/b6")" = "$LINK_ID"
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert len(d["links"])==1' "$TMP/b6"

curl -sS -o "$TMP/b6b" "$BASE/v1/links/pending" -H "Authorization: Bearer $LAPTOP"
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert len(d["links"])==1' "$TMP/b6b"

# Phone cannot pull
curl -sS -D "$TMP/h6p" -o "$TMP/b6p" "$BASE/v1/links/pending" \
  -H "Authorization: Bearer $PHONE"
test "$(hdr_code <"$TMP/h6p")" = "403"

# 5. Ack, then empty pending, then ack again
curl -sS -D "$TMP/h7" -o "$TMP/b7" -X POST "$BASE/v1/links/$LINK_ID/ack" \
  -H "Authorization: Bearer $LAPTOP"
test "$(hdr_code <"$TMP/h7")" = "204"

curl -sS -o "$TMP/b8" "$BASE/v1/links/pending" -H "Authorization: Bearer $LAPTOP"
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); assert d["links"]==[]' "$TMP/b8"

curl -sS -D "$TMP/h9" -o /dev/null -X POST "$BASE/v1/links/$LINK_ID/ack" \
  -H "Authorization: Bearer $LAPTOP"
test "$(hdr_code <"$TMP/h9")" = "204"

N="$(docker compose exec -T postgres psql -U deskdrop -d deskdrop -tAc \
  "SELECT count(*) FROM link_deliveries WHERE link_id = '$LINK_ID'")"
test "$N" = "1"
ACKED="$(docker compose exec -T postgres psql -U deskdrop -d deskdrop -tAc \
  "SELECT count(*) FROM link_deliveries WHERE link_id = '$LINK_ID' AND acked_at IS NOT NULL")"
test "$ACKED" = "1"

MET="$(curl -sfS "$BASE/metrics")"
echo "$MET" | grep -q 'deskdrop_shares_total'
echo "$MET" | grep -q 'deskdrop_pulls_total'
echo "$MET" | grep -q 'deskdrop_acks_total'
echo "$MET" | grep -q 'deskdrop_pairings_total'

echo "e2e ok link_id=$LINK_ID deliveries=1"

echo "--- agent ---"
"$ROOT/tests/e2e/compose_agent.sh"
