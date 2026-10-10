# Follow one link

Stub as of A1. Rewrite this whenever a component joins the path from share to ack.

Target hop list (names will become real handlers, tables, and metrics in Phase B–C):

1. Android share sheet → API `POST /v1/links`
2. API inserts `links` + `link_deliveries` in Postgres
3. Laptop agent `GET /v1/links/pending`
4. Agent opens the default browser
5. Agent `POST /v1/links/{id}/ack`
6. Retention job deletes delivered rows after 30 days

Until B4, you cannot demo this with curl. Do not invent command output here.
