# Follow one link

Updated C4 (dashboard). HTTPS D0.

Handlers live in `internal/httpserver`. Tables are migration `000001_init`. Every JSON response (and `/healthz`) carries `X-Request-Id`; JSON logs use field `request_id`. Counters: `deskdrop_pairings_total`, `deskdrop_shares_total`, `deskdrop_pulls_total`, `deskdrop_acks_total`. Latency histogram: `deskdrop_http_request_duration_seconds{method,code,route}`.

Automated trace: `tests/e2e/compose_api.sh` (HTTP + agent binary). Pull vs push: [ADR 0007](adr/0007-pull-vs-push.md).

0. **Pair.** `POST /v1/pairings` (`createPairing`) inserts `users` (bootstrap) + `pairing_codes`. Phone (C1), dashboard web claim (C4, httpOnly cookie; proxy strips `token` from JSON), or curl `POST /v1/pairings/claim` (`claimPairing`) writes `devices.token_hash` (HMAC). Extra devices: authenticated create + QR on the dashboard, or `deskdrop-agent pair`. [pairing.md](components/pairing.md), [android.md](components/android.md), [agent.md](components/agent.md), [dashboard.md](components/dashboard.md).
1. **Share.** Android `ACTION_SEND` (C2) or curl `POST /v1/links` (`shareLink`) with `Idempotency-Key` (new UUID per tap; HTTP retries reuse it). Body `userId` ignored. Non-http(s) rejected in the app. `links` + `link_deliveries` in one transaction. New share → 201 and `deskdrop_shares_total`. Replay → 200, same `id`. [delivery.md](components/delivery.md), [android.md](components/android.md).
2. **Queue.** Rows in Postgres, not Redis. Fan-out only to laptops already paired ([ADR 0004](adr/0004-at-least-once-ack.md)).
3. **Pull.** Laptop `GET /v1/links/pending` (`pendingLinks`) uses `link_deliveries_pending_by_device_idx`. Increments `deskdrop_pulls_total`. Phone/web 403. Pull does **not** ack — a second pull still returns the same row (sleep-mid-ack). Agent poll: 10s.
4. **Open.** `deskdrop-agent` (`internal/agent` runner) opens the default browser (`open` on macOS). Same-session: one open per id; ack is retried without a second open. Poll every 10s (and on process start / wake).
5. **Ack.** `POST /v1/links/{id}/ack` (`ackLink`) sets `link_deliveries.acked_at` once. Repeat is 204. Increments `deskdrop_acks_total` including repeats. Still one delivery row.
6. **Cleanup.** Hourly `purgeDelivered` deletes acked deliveries and fully delivered links older than 30 days ([ADR 0005](adr/0005-cursor-history-ttl.md)). History: `GET /v1/links` (dashboard list). Wipe: `DELETE /v1/account` (dashboard confirm DELETE). Unpair laptop in the dashboard → agent poll 401.

Sleep-mid-ack, duplicate share, and API restart: [what-breaks-if.md](what-breaks-if.md).

## Curl-only trace

```bash
docker compose up --build -d
# or: tests/e2e/compose_api.sh

curl -sS -D - http://127.0.0.1:8080/v1/pairings -H 'Content-Type: application/json' -d '{}'
# 201, X-Request-Id, JSON code. slog request_id on the API log line.

curl -sS http://127.0.0.1:8080/v1/pairings/claim -H 'Content-Type: application/json' \
  -d '{"code":"ABCD-EFGH","type":"phone"}'
# 201 token. Repeat claim → 409. deskdrop_pairings_total += 1

# Mint laptop on the same user, then:
curl -sS -D - http://127.0.0.1:8080/v1/links \
  -H "Authorization: Bearer $PHONE" -H "Idempotency-Key: demo-1" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/from-phone","title":"demo","userId":"ignored"}'
# 201. Replay same key → 200, same id. deskdrop_shares_total stays 1.

curl -sS http://127.0.0.1:8080/v1/links/pending -H "Authorization: Bearer $LAPTOP"
# {"links":[{id,url,...}]} twice if you do not ack (sleep-mid-ack).

curl -sS -D - -X POST http://127.0.0.1:8080/v1/links/$LINK_ID/ack \
  -H "Authorization: Bearer $LAPTOP"
# 204; repeat 204; pending then []. One link_deliveries row.

docker compose restart api
# Pending unacked rows are still in Postgres after listen.

curl -sS http://127.0.0.1:8080/metrics | grep -E 'deskdrop_(pairings|shares|pulls|acks)_total'
```
