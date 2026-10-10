# What breaks if

API-only (B7). Clients (C1–C4) and public TLS (D0) add rows later. Queue is Postgres ([ADR 0001](adr/0001-postgres-as-queue.md)); agents **pull** ([ADR 0007](adr/0007-pull-vs-push.md)).

| Scenario | What happens | How to see |
|---|---|---|
| Postgres down | `/healthz` stays 200. `/readyz` is 503 (`readyz` handler). Shares/pulls/acks 500. No silent drop: uncommitted inserts do not exist. | Stop Postgres; `curl /readyz` → 503. JSON log `readyz ping failed` with `request_id`. |
| Laptop sleeps mid-ack | `link_deliveries.acked_at` stays NULL. Next `GET /v1/links/pending` (`pendingLinks`) returns the same `links` row. `POST .../ack` (`ackLink`) is 204 even if retried. At-least-once: the agent may open the URL twice (C3). | Share, pull, **do not ack**, pull again — still one element. Then ack twice — still one row with `acked_at` set. |
| API process restarts | No in-memory queue. `links` / `link_deliveries` remain. In-memory claim + write buckets reset (brief extra burst, [ADR 0006](adr/0006-per-pod-rate-limit.md)). | Share, `docker compose restart api`, pull — same pending id. slog `listening` then `request_id` on the next pull. |
| Duplicate share | Unique `(user_id, idempotency_key)` on `links`. Replay is 200, same `id`, no second `link_deliveries` fan-out. `deskdrop_shares_total` does not increment. | `POST /v1/links` twice with the same `Idempotency-Key`. |
| Device token leaks | `DELETE /v1/devices/{id}` (`revokeDevice`) sets `devices.revoked_at`. Next Bearer/cookie is 401 (`requireAuth`). DB dump has HMAC (`devices.token_hash`), not the raw token. | Revoke, then `GET /v1/links/pending` → 401. |
| VM dies | Restore from backup (D6). A0 still pending. | n/a until D6 |
| TLS cert expires | Caddy or cert-manager renew (D0 / D4). Local Compose is HTTP. | n/a until D0 |
| 6th wrong claim / IP / min | `claimPairing` + in-memory window → 429 | [rate-limiter.md](components/rate-limiter.md) |
| 6th pairing create / IP burst | `withWriteLimit` token bucket → 429 | same |
| Disallowed browser Origin | Request still runs; **no** `Access-Control-Allow-Origin` | `curl -H 'Origin: https://evil.example' /healthz` |

`X-Request-Id` is on every response (sanitized `[A-Za-z0-9-_]{1,64}`). Probe paths log at DEBUG; others at INFO with `request_id`, `route` (ServeMux pattern, never a raw id in the metric label).

## Verify or demo

```bash
docker compose up --build -d
tests/e2e/compose_api.sh
# prints e2e ok; asserts single delivery + double ack + replay 200

# API restart does not drop an unacked share:
# (run the pair+share+pull steps, then)
docker compose restart api
curl -sS http://127.0.0.1:8080/v1/links/pending -H "Authorization: Bearer $LAPTOP"
# same link id still pending until ack
```
