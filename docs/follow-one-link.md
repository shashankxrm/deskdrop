# Follow one link

Updated B2 (pairing exists; share/pull/ack do not).

0. Phone or laptop `POST /v1/pairings` then `POST /v1/pairings/claim` — HMAC in `pairing_codes` / `devices`, counter `deskdrop_pairings_total` ([pairing.md](components/pairing.md))
1. Android share sheet → API `POST /v1/links` (not built; B4)
2. API inserts `links` + `link_deliveries` in Postgres — tables from migration `000001_init` (A2)
3. Laptop agent `GET /v1/links/pending` — partial index `link_deliveries_pending_by_device_idx` (A2); handler B4
4. Agent opens the default browser (C3)
5. Agent `POST /v1/links/{id}/ack` — column `link_deliveries.acked_at` (A2); handler B4
6. Retention job deletes delivered rows after 30 days (B5)

Live today: schema, health/ready/metrics, pairing. Demo: [pairing.md](components/pairing.md).
