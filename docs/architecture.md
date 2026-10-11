# Architecture

Updated C4. Update this file whenever a component joins the flow.

## What it is

deskdrop queues a URL from an Android share until a paired laptop agent pulls it, opens the default browser, and acks. One user is one pairing chain. Production target is a single Always Free Oracle **Ampere** VM (A0 recorded in [terraform.md](components/terraform.md)).

Locally, Docker Compose runs Postgres 16, golang-migrate, and the Go API (`/healthz`, `/readyz`, `/metrics`). The dashboard is `npm run dev` in `dashboard/` on port 3000 (proxies `/v1`).

## Components

| Piece | Role | Status |
|---|---|---|
| Go API | pairing through retention, CORS, write limits; compose E2E | B1–B7 ([api.md](components/api.md), [pairing.md](components/pairing.md), [auth.md](components/auth.md), [delivery.md](components/delivery.md), [rate-limiter.md](components/rate-limiter.md), [follow-one-link.md](follow-one-link.md)) |
| PostgreSQL | users, devices, pairing_codes, links, link_deliveries | A2 — schema in [database-schema.md](components/database-schema.md) |
| Android client | pairing + `ACTION_SEND` share | C1–C2 ([android.md](components/android.md)) |
| macOS agent | pair, 10s poll, default browser, ack, LaunchAgent, 0600 config | C3 ([agent.md](components/agent.md)) |
| Next.js dashboard | web pair + QR, devices, history, delete account, /privacy | C4 ([dashboard.md](components/dashboard.md), [privacy.md](privacy.md)) |
| Oracle VM | Ampere A1 Flex 1 OCPU / 6 GB Running; public HTTPS at D0 | A0 Ampere ([terraform.md](components/terraform.md)) |
| CI | `go vet` + `go test ./...` on push/PR to `v2` | B1b ([ci-cd.md](components/ci-cd.md)) |

Queue choice: [ADR 0001 — Postgres as the delivery queue](adr/0001-postgres-as-queue.md). Pull vs push: [ADR 0007](adr/0007-pull-vs-push.md). Ack: [ADR 0004](adr/0004-at-least-once-ack.md). History/TTL: [ADR 0005](adr/0005-cursor-history-ttl.md). Limits: [ADR 0006](adr/0006-per-pod-rate-limit.md).

## Data flow (target)

```mermaid
sequenceDiagram
  participant Phone
  participant API
  participant PG as Postgres
  participant Agent as LaptopAgent
  participant Browser
  Phone->>API: POST /v1/links
  API->>PG: insert links and link_deliveries
  Agent->>API: GET /v1/links/pending
  API->>PG: select where acked_at is null
  API-->>Agent: links
  Agent->>Browser: open URL
  Agent->>API: POST /v1/links/{id}/ack
  API->>PG: set acked_at once
  Note over API,PG: hourly purge acked > 30d
```

## Verify

- Schema: [database-schema.md](components/database-schema.md)
- API: [api.md](components/api.md) (`go test ./...`, `curl /readyz`, `/metrics`)
- Delivery: [delivery.md](components/delivery.md) (share, pending, ack)
- CORS / limits: [rate-limiter.md](components/rate-limiter.md) (`hey` 429, claim 429)
- Follow one link / breaks: [follow-one-link.md](follow-one-link.md), [what-breaks-if.md](what-breaks-if.md)
- Compose E2E: [testing.md](components/testing.md) (`tests/e2e/compose_api.sh`)
- Suites: [testing.md](components/testing.md)
- Android: [android.md](components/android.md) (`./gradlew test`)
- Agent: [agent.md](components/agent.md) (`go test ./internal/agent/`, `tests/e2e/compose_agent.sh`)
- Dashboard: [dashboard.md](components/dashboard.md) (`cd dashboard && npm test` / `npm run test:e2e`)
- CI: [ci-cd.md](components/ci-cd.md) (job `go-test`)
