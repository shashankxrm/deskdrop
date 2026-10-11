# Testing

How to run each suite. This file grows with B/C steps. No testify. No mocked Postgres.

## Schema (A2)

```bash
docker compose up -d
docker compose exec -T postgres \
  psql -U deskdrop -d deskdrop -v ON_ERROR_STOP=1 \
  < tests/schema/fk_delete.sql
```

## Go (B1)

Standard library `testing`, table-driven config tests, `httptest` for handlers, testcontainers-go for `/readyz` against real Postgres 16.

```bash
go test ./...
```

Docker must be running for `TestReadyz_ok`.

CI job **go-test** (B1b) runs `go vet ./...` and `go test ./...` on every push/PR to `v2`. See [ci-cd.md](ci-cd.md).

`github.com/stretchr/testify` may appear as an **indirect** module of testcontainers-go. Our tests must not import it.

Later steps add pairing/delivery tests in the same `go test ./...` invocation.

## Pairing (B2)

`go test ./internal/httpserver/ ./internal/tokhash/` covers bootstrap vs extra device, HMAC-at-rest, 409/410, and 6th wrong claim → 429 (testcontainers). Same `go test ./...` as CI.

## Auth (B3)

`go test ./internal/httpserver/ -run TestAuth_` — Bearer vs cookie, rename ignores `userId`, revoke then 401 on list/share/pull. Laptop pending after B4 is 200 (empty list).

## Delivery (B4)

`go test ./internal/httpserver/ -run TestShare_` — table-driven URL/body/key validation, idempotent replay, phone cannot pull, double ack, late laptop misses old shares, `/metrics` counters (testcontainers). Same `go test ./...` as CI.

## History, account, retention (B5)

`go test ./internal/httpserver/ -run 'TestHistory_|TestAccount_|TestRetention_'` — cursor pages stay stable after a new share, `DELETE /v1/account` leaves zero rows for that user, stamped 31-day-old delivered rows are purged (unacked kept). Same `go test ./...` as CI.

## CORS and rate limits (B6)

`go test ./internal/httpserver/ -run 'TestCORS_|TestWriteLimit_|TestTokenBucket_|TestSecurityHeaders'` — disallowed Origin has no ACAO; 6th pairing create from one IP is 429; security headers on `/healthz`. `hey` demo is in [rate-limiter.md](rate-limiter.md) (not CI).

## Compose E2E (B7)

HTTP-only: pair phone + laptop, share, pull (twice still one row), ack, double ack still one `link_deliveries` row, replayed `Idempotency-Key` is 200. Not in GitHub Actions until D1.

```bash
docker compose up --build -d
tests/e2e/compose_api.sh
# e2e ok link_id=... deliveries=1
```

Needs Docker, `curl`, `python3`, and `psql` via `docker compose exec`. Trace and failure modes: [follow-one-link.md](../follow-one-link.md), [what-breaks-if.md](../what-breaks-if.md). After the HTTP asserts, the same script runs [compose_agent.sh](../../tests/e2e/compose_agent.sh) (C3).

## Android (C1–C2)

JUnit 4 on the JVM (no device required). Full-jitter backoff, pairing-code normalize, **table-driven URL parse**, share retry reuses `Idempotency-Key`. Gradle is not in GitHub Actions until D1.

```bash
echo "sdk.dir=$HOME/Library/Android/sdk" > android/local.properties
cd android && ./gradlew test
```

LAN pair + YouTube share: [android.md](android.md).

## macOS agent (C3)

Stdlib `testing` + `httptest` (no browser, no testify). Poll/ack/idempotency, 0600 config, 401 does not retry, 429 does.

```bash
go test ./internal/agent/
go build -o /tmp/deskdrop-agent ./cmd/agent

docker compose up --build -d
tests/e2e/compose_agent.sh
# e2e agent ok link_id=... opened=1
```

`compose_api.sh` also runs the agent script at the end. Sleep/wake and Gatekeeper: [agent.md](agent.md).
