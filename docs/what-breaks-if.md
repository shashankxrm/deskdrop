# What breaks if

Through D6 (backups). Queue is Postgres ([ADR 0001](adr/0001-postgres-as-queue.md)); agents **pull** ([ADR 0007](adr/0007-pull-vs-push.md)).

| Scenario | What happens | How to see |
|---|---|---|
| Postgres down | `/healthz` stays 200. `/readyz` is 503 (`readyz` handler). Shares/pulls/acks 500. No silent drop: uncommitted inserts do not exist. | Stop Postgres; `curl /readyz` → 503. JSON log `readyz ping failed` with `request_id`. |
| Laptop sleeps mid-ack | `link_deliveries.acked_at` stays NULL. Next `GET /v1/links/pending` returns the same row. Same process: agent retries **ack only** (no second `open`). Process died before ack: next launch opens again (at-least-once). Repeat ack is 204. | `go test ./internal/agent/ -run TestRunner_openOnceIfAckFails`; or share, sleep Mac 2+ min, one tab on wake. |
| API process restarts | No in-memory queue. `links` / `link_deliveries` remain. In-memory claim + write buckets reset (brief extra burst, [ADR 0006](adr/0006-per-pod-rate-limit.md)). Two replicas: killing one pod does not drop pending rows. | Share, `kubectl -n deskdrop delete pod -l app=api`, pull — same pending id. |
| Duplicate share | Unique `(user_id, idempotency_key)` on `links`. Replay is 200, same `id`, no second `link_deliveries` fan-out. `deskdrop_shares_total` does not increment. | `POST /v1/links` twice with the same `Idempotency-Key`. |
| Device token leaks | `DELETE /v1/devices/{id}` (`revokeDevice`) sets `devices.revoked_at`. Next Bearer/cookie is 401 (`requireAuth`). DB dump has HMAC (`devices.token_hash`), not the raw token. Dashboard unpair of the laptop → agent poll 401. Pepper leak is worse: rotate `DEVICE_TOKEN_PEPPER` and re-pair every device ([secrets.md](runbooks/secrets.md)). | Dashboard **Unpair**; `./scripts/check-secrets.sh`; missing env → API exit 1. |
| VM dies | Restore gzip SQL from Object Storage `deskdrop-backups` ([restore-postgres.md](runbooks/restore-postgres.md)). Keep the instance Running. | `./scripts/pg-restore-drill.sh` (sidecar DB, not live overwrite) |
| TLS cert expires | cert-manager renews Let’s Encrypt via Traefik HTTP-01. If DuckDNS A record or port 80 breaks, HTTPS fails. | `curl -vI https://HOST/healthz`; `kubectl -n deskdrop describe certificate deskdrop-tls` |
| 6th wrong claim / IP / min | `claimPairing` + in-memory window → 429 | [rate-limiter.md](components/rate-limiter.md) |
| 6th pairing create / IP burst | `withWriteLimit` token bucket → 429 | same |
| Disallowed browser Origin | Request still runs; **no** `Access-Control-Allow-Origin` | `curl -H 'Origin: https://evil.example' /healthz` |

`X-Request-Id` is on every response (sanitized `[A-Za-z0-9-_]{1,64}`). Probe paths log at DEBUG; others at INFO with `request_id`, `route` (ServeMux pattern, never a raw id in the metric label).

## Verify or demo

```bash
docker compose up --build -d
tests/e2e/compose_api.sh
# prints e2e ok then e2e agent ok; HTTP single delivery + agent binary one open

# API restart does not drop an unacked share:
# (run the pair+share+pull steps, then)
docker compose restart api
curl -sS http://127.0.0.1:8080/v1/links/pending -H "Authorization: Bearer $LAPTOP"
# same link id still pending until ack
```
