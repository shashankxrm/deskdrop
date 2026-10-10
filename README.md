# deskdrop

Share a link from Android and have it open in your laptop browser the next time that laptop is on — even if the machine was asleep or off when you shared.

v2 is a rebuild: phone and laptop are devices under one pairing chain, links sit in Postgres until the laptop pulls and acks them, and nothing is deleted on the way.

**Not production yet.** There is no public HTTPS URL, no APK, and no laptop agent download. Pairing, delivery, and hosting land in later steps on this `v2` branch.

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
- [v1 retrospective](docs/v1-retrospective.md) (why the first version never worked end to end)
- [Database schema](docs/components/database-schema.md)
- [Oracle / Terraform (A0 pending)](docs/components/terraform.md)
- [Testing](docs/components/testing.md) / [CI](docs/components/ci-cd.md)

Local API: `/healthz`, `/readyz` (Postgres ping), `/metrics` (latency histogram). Pairing is not built yet. See [testing.md](docs/components/testing.md).
