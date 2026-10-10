# What breaks if

Stub as of A1. Fill each scenario with real behavior when the code exists (B7, then clients and deploy).

| Scenario | v2 intent (not yet implemented) |
|---|---|
| Postgres down | `/readyz` fails; shares and pulls error; no silent drop of links |
| Laptop sleeps mid-ack | row stays pending; next pull retries; ack is idempotent |
| API process restarts | no in-memory queue; pending rows remain in Postgres |
| Duplicate share | same `Idempotency-Key` does not insert a second link |
| Device token leaks | revoke that device; HMAC at rest so DB dump is not the raw token |
| VM dies | restore from backup (D6); A0 still pending |
| TLS cert expires | Caddy or cert-manager renew (D0 / D4) |

## Verify

No commands until the API exists. After B7 this file must name handlers, tables, and log fields.
