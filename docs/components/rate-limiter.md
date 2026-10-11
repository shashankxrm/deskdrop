# Rate limits and CORS

Write paths are limited **per API process, per client IP**. Pairing **claims** use a sliding failure window (B2). Pairing **creates** and **shares** use a token bucket (B6). A global Postgres limiter is Post-v2. See [ADR 0006](../adr/0006-per-pod-rate-limit.md).

## Why this way

- Guessing pairing codes and flooding bootstrap users would be cheap without a cap.
- Two replicas after D4 each have their own buckets — documented, not a silent miss.
- CORS is an exact Origin allowlist from `CORS_ORIGINS`. Empty list means no `Access-Control-Allow-Origin` (agents and curl do not send Origin).

## Limits (per IP, per pod)

| Path | Rule | 429 when |
|---|---|---|
| `POST /v1/pairings/claim` | 5 failed claims / minute (B2) | 6th **wrong** claim |
| `POST /v1/pairings` | token bucket burst 5, refill 5/min | 6th **create** in a burst |
| `POST /v1/links` | token bucket burst 30, refill 30/min | 31st share in a burst |

Successful claims are not counted toward the claim window. Claim is not the create bucket.

## CORS and headers

- `CORS_ORIGINS` — comma-separated exact Origins (scheme + host + port).
- Allowed Origin: `Access-Control-Allow-Origin` echo, credentials, methods GET/POST/PATCH/DELETE/OPTIONS, headers `Authorization, Content-Type, Idempotency-Key, X-Request-Id`.
- Disallowed Origin: request still runs; **no ACAO**.
- Every response: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Cache-Control: no-store`, restrictive CSP. HSTS waits for HTTPS at D0.

## Failure modes

- Empty `CORS_ORIGINS`: fine for curl/agents. The C4 dashboard is same-origin via a `/v1` proxy on `:3000` (D0 Caddy will be `/` + `/v1`). Do not set `*`.
- Burst across two pods is up to 2× the numbers above.
- `hey` against create will 429; that is the demo, not a bug.

## Verify or demo

Compose must be up (`docker compose up --build -d`). `hey` is [github.com/rakyll/hey](https://github.com/rakyll/hey) (`go install github.com/rakyll/hey@latest`).

```bash
# Claim-per-IP (B2): 6th wrong claim is 429.
for i in 1 2 3 4 5 6; do
  curl -sS -o /tmp/claim.body -w "$i %{http_code}\n" \
    http://127.0.0.1:8080/v1/pairings/claim \
    -H 'Content-Type: application/json' \
    -d '{"code":"ZZZZ-ZZZZ","type":"phone"}'
done
# 1–5: 404; 6: 429

# Pairing-create token bucket (B6): some of these 20 will be 429.
hey -n 20 -c 10 -m POST \
  -H 'Content-Type: application/json' \
  -d '{}' \
  http://127.0.0.1:8080/v1/pairings
# Status code distribution includes 429

# Disallowed Origin has no ACAO (even on 200).
curl -sS -D - -o /dev/null http://127.0.0.1:8080/healthz \
  -H 'Origin: https://evil.example'
# 200, no Access-Control-Allow-Origin, X-Content-Type-Options: nosniff

# Allowed Origin (set CORS_ORIGINS then rebuild) echoes ACAO.
# CORS_ORIGINS=https://deskdrop.test docker compose up --build -d
curl -sS -D - -o /dev/null http://127.0.0.1:8080/healthz \
  -H 'Origin: https://deskdrop.test'
# Access-Control-Allow-Origin: https://deskdrop.test

go test ./internal/httpserver/ -run 'TestCORS_|TestWriteLimit_|TestTokenBucket_|TestSecurityHeaders'
go test ./internal/config/ -run TestSplitCORSOrigins
```
