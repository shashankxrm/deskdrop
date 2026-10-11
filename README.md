# deskdrop

Share a link from Android and have it open in your laptop browser the next time that laptop is on — even if the machine was asleep or off when you shared.

v2 is a rebuild: phone and laptop are devices under one pairing chain, links sit in Postgres until the laptop pulls and acks them, and nothing is deleted on the way.

Closed beta is D0: Compose + Caddy on the Always Free Ampere VM. Set the DuckDNS hostname in `deploy/prod/.env`. Local Compose remains for development.

## Features (when v2 is live)

- Pair phone, laptop, and a web dashboard with a short-lived QR / pairing code. No password.
- Android share target sends a URL (optional title) to the API.
- Laptop agent starts at login, pulls pending links, opens the default browser, then acks.
- Dashboard: devices, link history, account delete.
- Delivered links purged after 30 days.

## Stack

Go API, PostgreSQL, Next.js dashboard, Kotlin Android client, Go macOS tray agent.

## Connect phone and laptop

[connect.md](docs/connect.md) — or https://shashank-dropdesk.duckdns.org/connect

## Docs

- [Connect phone + laptop](docs/connect.md)
- [Architecture](docs/architecture.md)
- [Follow one link](docs/follow-one-link.md) / [What breaks if](docs/what-breaks-if.md)
- [v1 retrospective](docs/v1-retrospective.md) (why the first version never worked end to end)
- [Database schema](docs/components/database-schema.md)
- [Oracle / Terraform (A0 Ampere)](docs/components/terraform.md)
- [Testing](docs/components/testing.md) / [CI](docs/components/ci-cd.md)
- [Pairing](docs/components/pairing.md) / [Auth](docs/components/auth.md)
- [Delivery, history, retention](docs/components/delivery.md)
- [CORS and rate limits](docs/components/rate-limiter.md)
- [Android pairing and share](docs/components/android.md)
- [macOS agent](docs/components/agent.md)
- [Dashboard](docs/components/dashboard.md) / [Privacy](docs/privacy.md)
- [Prod deploy](docs/runbooks/deploy-compose-caddy.md) / [Scaling](docs/scaling.md) / [Grafana](docs/components/grafana.md)

Local: `docker compose up --build -d`. Dashboard: `cd dashboard && npm run dev`. Prod: [deploy-compose-caddy.md](docs/runbooks/deploy-compose-caddy.md). Releases: `./scripts/release-checksums.sh`.
