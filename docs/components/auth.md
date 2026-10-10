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
| POST | `/v1/links` | auth required; see [delivery.md](delivery.md) |
| GET | `/v1/links/pending` | laptop only; see [delivery.md](delivery.md) |
| POST | `/v1/links/{id}/ack` | laptop only; see [delivery.md](delivery.md) |
| GET | `/v1/links` | history; see [delivery.md](delivery.md) |
| DELETE | `/v1/account` | deletes the user row; FKs cascade; web cookie cleared |

## Failure modes

- Missing/invalid/revoked token: 401.
- Rename/revoke a device that is not yours: 404 (no leak).
- Revoked token cannot list, share, or pull.
- After `DELETE /v1/account`, every device token for that user is 401 and no `users`/`devices`/`links` rows remain.

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
# 401 after revoke; a live laptop token gets 200 `{"links":[]}` if nothing is pending

curl -sS -D - -X DELETE http://127.0.0.1:8080/v1/account \
  -H "Authorization: Bearer $TOKEN"
# 204; the same token then 401s on /v1/devices

go test ./internal/httpserver/ -run 'TestAuth_|TestPairing_|TestAccount_'
```
