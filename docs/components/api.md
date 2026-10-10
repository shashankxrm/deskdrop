# API

Go `net/http` process (`go 1.26` in go.mod so testcontainers’ transitive modules compile). B1 is the skeleton: config from env, JSON `slog`, request IDs, graceful shutdown, liveness/readiness, Prometheus latency histogram. Pairing and delivery are later steps.

## Why this way

- Missing `DEVICE_TOKEN_PEPPER` or `DATABASE_URL` is a hard exit. No compiled-in fallbacks (v1’s default bearer).
- `/healthz` does not touch Postgres (kube/compose liveness). `/readyz` pings Postgres (readiness).
- Each request gets `X-Request-Id` on the response and in the JSON log field `request_id`. Incoming values are kept only if they match `[A-Za-z0-9-_]{1,64}`; otherwise a new id is generated.
- `/healthz` and `/readyz` log at DEBUG so Compose healthchecks do not flood INFO.
- Latency is `deskdrop_http_request_duration_seconds{method,code,route}` on `/metrics`. `route` is the ServeMux pattern (e.g. `/v1/links/{id}/ack`), never the raw URL path; unmatched requests use `route="unmatched"`.

## Alternatives

- Frameworks (chi, echo, gin): extra API surface to defend in interviews. stdlib is enough.
- OpenTelemetry traces: Post-v2. Request IDs + histogram are v2.

## Failure modes

- Empty pepper or DB URL: process never binds a port (exit 1).
- Postgres down: `/healthz` stays 200, `/readyz` is 503, logs `readyz ping failed` with `request_id`.
- SIGINT/SIGTERM: 10s `Shutdown` then exit.

## Verify or demo

```bash
# Unit + httptest + testcontainers (Docker required for /readyz).
go test ./...

# Compose (local pepper is injected by compose.yaml, not by Go defaults).
docker compose up --build -d

curl -sS http://127.0.0.1:8080/healthz
# ok

curl -sS -D - http://127.0.0.1:8080/readyz -o /tmp/readyz.body
# HTTP/1.1 200, body ok, header X-Request-Id

# Probe lines stay off INFO (healthcheck hits /healthz every 5s).
docker compose logs api --tail=20
# no GET /healthz at level INFO

# Invalid request id is replaced (response header is a generated hex id).
curl -sS -D - -H 'X-Request-Id: not valid!' http://127.0.0.1:8080/healthz -o /dev/null

# Unmatched path logs at INFO and records route="unmatched" (not the raw path).
curl -sS -H 'X-Request-Id: demo-1' http://127.0.0.1:8080/no-such
docker compose logs api --tail=5
# JSON line with "request_id":"demo-1" and "route":"unmatched"

curl -sS http://127.0.0.1:8080/metrics | grep deskdrop_http_request_duration_seconds
# series include route="/healthz" and route="unmatched", never a raw path with ids
```
