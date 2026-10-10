# Follow one link

Updated B5 (history, account delete, 30-day retention).

0. Phone or laptop `POST /v1/pairings` then `POST /v1/pairings/claim` — HMAC in `pairing_codes` / `devices`, counter `deskdrop_pairings_total` ([pairing.md](components/pairing.md)). Later hops use Bearer or the web cookie ([auth.md](components/auth.md)).
1. Android share sheet → API `POST /v1/links` with `Idempotency-Key` — handler B4 ([delivery.md](components/delivery.md)); client C2
2. API inserts `links` + `link_deliveries` in Postgres — tables from migration `000001_init` (A2); fan-out only to laptops already paired ([ADR 0004](adr/0004-at-least-once-ack.md))
3. Laptop agent `GET /v1/links/pending` — partial index `link_deliveries_pending_by_device_idx` (A2); handler B4
4. Agent opens the default browser (C3)
5. Agent `POST /v1/links/{id}/ack` — column `link_deliveries.acked_at` (A2); handler B4, idempotent 204
6. Hourly `purgeDelivered` in the API process deletes acked deliveries and fully delivered links older than 30 days ([ADR 0005](adr/0005-cursor-history-ttl.md)). Dashboard history is `GET /v1/links`. `DELETE /v1/account` drops the pairing chain.

Live today: schema, health/ready/metrics, pairing, auth, share/pull/ack, history, account delete, retention. Demo: [delivery.md](components/delivery.md).
