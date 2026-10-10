# Follow one link

Updated A2 (tables exist; handlers do not). Rewrite this whenever a component joins the path from share to ack.

1. Android share sheet → API `POST /v1/links` (not built; B4)
2. API inserts `links` + `link_deliveries` in Postgres — tables from migration `000001_init` (A2)
3. Laptop agent `GET /v1/links/pending` — partial index `link_deliveries_pending_by_device_idx` (A2); handler B4
4. Agent opens the default browser (C3)
5. Agent `POST /v1/links/{id}/ack` — column `link_deliveries.acked_at` (A2); handler B4
6. Retention job deletes delivered rows after 30 days (B5)

Until B4, the only live hop is schema + `/healthz`. Demo commands: [database-schema.md](components/database-schema.md).
