# Delivery (share, pull, ack)

Share writes one `links` row and fans out `link_deliveries` to **currently paired, non-revoked laptops**. The agent pulls only its own unacked rows and acks them. See [ADR 0004](../adr/0004-at-least-once-ack.md).

## Why this way

- Unique `(user_id, idempotency_key)` so Android retries do not duplicate a share.
- Fan-out at share time keeps pull a cheap indexed select. Laptops paired later do not receive older links.
- Ack sets `acked_at`; a repeat ack is still 204. The row stays until B5 retention.

## Endpoints

| Method | Path | Who |
|---|---|---|
| POST | `/v1/links` | any authed device; `Idempotency-Key` required; body `{url, title?}` (`userId` ignored) |
| GET | `/v1/links/pending` | **laptop** only |
| POST | `/v1/links/{id}/ack` | **laptop** that has a delivery row |

`url` must be `http` or `https`, host required, max 2048 runes. Body cap 16KiB. Title optional, max 512 runes.

## Counters (`/metrics`)

- `deskdrop_shares_total` — new links only (replays are 200 and do not increment)
- `deskdrop_pulls_total` — successful pending GETs (including empty lists)
- `deskdrop_acks_total` — successful acks, including repeats

## Failure modes

| Case | Status |
|---|---|
| Missing/invalid/revoked token | 401 |
| Missing or oversized Idempotency-Key | 400 |
| Bad URL / invalid JSON | 400 |
| Body over 16KiB | 413 |
| Phone or web pulls or acks | 403 |
| Replay of the same key for that user | 200, same `id` |
| Ack of a link this laptop never received | 404 |
| Second ack | 204, still one `acked_at` |

## Verify or demo

Compose must be up (`docker compose up --build -d`). Pair a phone and a laptop on the same chain ([pairing.md](pairing.md)).

```bash
PHONE=...    # phone Bearer token
LAPTOP=...   # laptop Bearer token

# Share (userId in JSON must be ignored).
curl -sS -D - http://127.0.0.1:8080/v1/links \
  -H "Authorization: Bearer $PHONE" \
  -H "Idempotency-Key: demo-1" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/from-phone","title":"demo","userId":"ignored"}'
# HTTP 201, JSON id + url + title

# Same key: 200, same id, still one row.
curl -sS -D - http://127.0.0.1:8080/v1/links \
  -H "Authorization: Bearer $PHONE" \
  -H "Idempotency-Key: demo-1" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/from-phone","title":"demo"}'
# HTTP 200

# Phone cannot pull.
curl -sS -D - http://127.0.0.1:8080/v1/links/pending \
  -H "Authorization: Bearer $PHONE"
# HTTP 403

curl -sS http://127.0.0.1:8080/v1/links/pending \
  -H "Authorization: Bearer $LAPTOP"
# JSON {"links":[{id,url,title,created_at}]}

LINK_ID=...  # from the pending payload
curl -sS -D - -X POST http://127.0.0.1:8080/v1/links/$LINK_ID/ack \
  -H "Authorization: Bearer $LAPTOP"
# HTTP 204; repeat the same POST → still 204

curl -sS http://127.0.0.1:8080/v1/links/pending \
  -H "Authorization: Bearer $LAPTOP"
# {"links":[]}

curl -sS http://127.0.0.1:8080/metrics | grep -E 'deskdrop_(shares|pulls|acks)_total'

go test ./internal/httpserver/ -run 'TestShare_|TestAuth_'
```
