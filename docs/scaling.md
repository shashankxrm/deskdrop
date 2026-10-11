# Scaling (D0)

Production is **one Always Free Ampere VM**: `VM.Standard.A1.Flex`, 1 OCPU, 6 GB, public IPv4 137.131.14.30. Stack: Caddy (TLS) → Next.js dashboard + Go API + Postgres 16. No Redis. Queue is Postgres ([ADR 0001](adr/0001-postgres-as-queue.md)).

## What this box can take

Closed beta: you + a few friends. One API process, in-memory token buckets ([ADR 0006](adr/0006-per-pod-rate-limit.md)), one Postgres. Poll interval 10s per laptop is cheap.

It will **not** take: many API replicas (that is D4 k3s), a global rate limiter (Post-v2), or in-cluster Grafana (RAM). Metrics go to **Grafana Cloud** via Alloy (`--profile metrics`).

## Bottlenecks, in order

1. **DuckDNS / Let’s Encrypt** — cert fail if the A record is wrong or port 80 is filtered (OCI iptables).
2. **Postgres on the same disk** — fine for beta; D6 adds `pg_dump` off-box.
3. **Fan-out at share** — one insert per currently paired laptop. Dozens of laptops per user is still small.
4. **1 OCPU** — JSON slog + `/metrics` histogram are cheap vs TLS and Next.

## What D4 changes

Same VM, k3s, two API replicas, cert-manager. Per-pod limiter until Post-v2 Postgres limiter. Do not add a second VM in v2 (Always Free A1 budget).

## Verify

```bash
curl -fsS https://YOUR.duckdns.org/healthz
curl -fsS https://YOUR.duckdns.org/readyz
# Grafana: shares / acks / ack rate / p95 — docs/components/grafana.md
```
