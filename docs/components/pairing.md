# Pairing

Bootstrap creates a user and a short-lived code. Claim returns a device token once. Extra devices need a Bearer token from an already-paired device. See [ADR 0002](../adr/0002-pairing-hmac.md).

## Why this way

- One pairing chain = one user. No email, no development bearer.
- Codes and tokens stored as HMAC-SHA256, never plaintext.
- Failed claims are rate-limited per IP so codes cannot be guessed quickly.

## Failure modes

| Claim | Status |
|---|---|
| Unknown code | 404 (counts toward 429) |
| Already used | 409 |
| Expired (~3 min) | 410 |
| 6th failed claim / IP / minute | 429 |
| 6th pairing **create** / IP / minute (token bucket, B6) | 429 |
| Bad or missing Bearer on extra-device create | 401 |

## Verify or demo

Compose must be up (`docker compose up --build -d`). Pepper is injected by compose, not by Go defaults.

```bash
# Bootstrap (ignore any userId in the body).
curl -sS -D - http://127.0.0.1:8080/v1/pairings \
  -H 'Content-Type: application/json' \
  -d '{"userId":"ignored"}'
# HTTP 201, JSON code like ABCD-EFGH

# Claim as a laptop. Save the token; it is shown once.
curl -sS -D - http://127.0.0.1:8080/v1/pairings/claim \
  -H 'Content-Type: application/json' \
  -d '{"code":"ABCD-EFGH","type":"laptop","name":"mac"}'
# HTTP 201, token + device_id. Repeat the same claim → 409.

# Second device for the same user:
curl -sS http://127.0.0.1:8080/v1/pairings \
  -H "Authorization: Bearer $TOKEN"

# Wrong claims from one client: 6th in a minute is 429.
for i in 1 2 3 4 5 6; do
  curl -sS -o /tmp/claim.body -w "$i %{http_code}\n" \
    http://127.0.0.1:8080/v1/pairings/claim \
    -H 'Content-Type: application/json' \
    -d '{"code":"ZZZZ-ZZZZ","type":"phone"}'
done
# 1–5: 404; 6: 429

curl -sS http://127.0.0.1:8080/metrics | grep deskdrop_pairings_total
# counter bumped after a successful claim

go test ./internal/httpserver/ ./internal/tokhash/
```
