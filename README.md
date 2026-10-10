# deskdrop

Share a link from Android and have it open in your laptop browser the next time that laptop is on — even if the machine was asleep or off when you shared.

v2 is a rebuild: phone and laptop are devices under one pairing chain, links sit in Postgres until the laptop pulls and acks them, and nothing is deleted on the way.

**Not production yet.** There is no public HTTPS URL, no APK, and no laptop agent download. The Go API on this `v2` branch does pairing through history, account delete, 30-day retention, CORS, and per-pod write limits, with a compose HTTP E2E (`tests/e2e/compose_api.sh`). Android, macOS agent, dashboard, and hosting are later steps.

## Features (when v2 is live)

- Pair phone, laptop, and a web dashboard with a short-lived QR / pairing code. No password.
- Android share target sends a URL (optional title) to the API.
- Laptop agent starts at login, pulls pending links, opens the default browser, then acks.
- Dashboard: devices, link history, account delete.
- Delivered links purged after 30 days.

## Stack

Go API, PostgreSQL, Next.js dashboard, Kotlin Android client, Go macOS tray agent.

## Docs

- [Architecture](docs/architecture.md)
- [Follow one link](docs/follow-one-link.md) / [What breaks if](docs/what-breaks-if.md)
- [v1 retrospective](docs/v1-retrospective.md) (why the first version never worked end to end)
- [Database schema](docs/components/database-schema.md)
- [Oracle / Terraform (A0 pending)](docs/components/terraform.md)
- [Testing](docs/components/testing.md) / [CI](docs/components/ci-cd.md)
- [Pairing](docs/components/pairing.md) / [Auth](docs/components/auth.md)
- [Delivery, history, retention](docs/components/delivery.md)
- [CORS and rate limits](docs/components/rate-limiter.md)

Local API: `docker compose up --build -d` then `/healthz`, `/readyz`, `/metrics`. HTTP E2E: `tests/e2e/compose_api.sh`. Demos: [pairing.md](docs/components/pairing.md), [delivery.md](docs/components/delivery.md), [rate-limiter.md](docs/components/rate-limiter.md). Suites: [testing.md](docs/components/testing.md).
