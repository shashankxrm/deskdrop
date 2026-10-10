# Auth and devices

Every authenticated request is a **device**. User id is loaded from `devices` via the token hash, never from JSON. See [ADR 0003](../adr/0003-cookie-vs-bearer.md).

## Why this way

- Fixes v1’s `userId` in the body.
- Revoke is an immediate `revoked_at` — no waiting for a JWT to expire.
- Web gets an httpOnly cookie; agents get Bearer.

## Endpoints

| Method | Path | Auth |
|---|---|---|
| GET | `/v1/devices` | Bearer or cookie |
| PATCH | `/v1/devices/{id}` | same user; body `{name}` (`userId` ignored) |
| DELETE | `/v1/devices/{id}` | unpair (sets `revoked_at`) |
| POST | `/v1/links` | auth required; **501 until B4** |
| GET | `/v1/links/pending` | auth required; **501 until B4** |

## Failure modes

- Missing/invalid/revoked token: 401.
- Rename/revoke a device that is not yours: 404 (no leak).
- Revoked token cannot list, share, or pull.

## Verify or demo

```bash
# Pair a laptop (see pairing.md), then:
TOKEN=...
DEVICE_ID=...

curl -sS http://127.0.0.1:8080/v1/devices -H "Authorization: Bearer $TOKEN"
# JSON devices list

curl -sS -D - -X PATCH http://127.0.0.1:8080/v1/devices/$DEVICE_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"kitchen mac","userId":"ignored"}'
# 204

curl -sS -D - -X DELETE http://127.0.0.1:8080/v1/devices/$DEVICE_ID \
  -H "Authorization: Bearer $TOKEN"
# 204

curl -sS -D - http://127.0.0.1:8080/v1/links/pending \
  -H "Authorization: Bearer $TOKEN"
# 401

go test ./internal/httpserver/ -run 'TestAuth_|TestPairing_'
```
