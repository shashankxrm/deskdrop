# ADR 0008: Migrate live Compose to single-node k3s

## Status

Accepted (D4)

## Context

D0 put API + Postgres + dashboard + Caddy on the Always Free Ampere VM. A0 recorded Ampere, so k3s is in-scope for 1 Jan 2027. The VM is 1 OCPU / 6 GB. Pairing rows must survive the move. DuckDNS already points at `137.131.14.30`.

## Decision

- One k3s node on the existing VM. Traefik (k3s default) + cert-manager HTTP-01 replace Caddy.
- Postgres in-cluster, ClusterIP only, NetworkPolicy: only API and migrate pods to `:5432`.
- Two API replicas on GHCR `api:v2`. Dashboard one replica.
- Cutover is dump/restore (`pg_dump --clean`) while Compose API is stopped, then `compose down`, then Traefik takes 80/443. Same hostname.
- Grafana Cloud stays; Alloy scrapes the in-cluster API Service. No in-cluster Grafana.
- CI deploy uses `kubectl apply` when k3s is active, otherwise the old Compose path.

## Alternatives

- Keep Compose in production: misses the k3s-on-Oracle line with Ampere available.
- Attach the Compose Docker volume as a hostPath: couples k3s to Docker’s volume layout; dump/restore is explicit and checkable (`count(*) FROM users`).
- Two VMs / managed k8s: not Always Free Ampere budget.

## Consequences

- A few minutes of HTTPS downtime at cutover. Pull+ack means in-flight shares stay in Postgres ([ADR 0007](0007-pull-vs-push.md)).
- In-memory rate limits are per pod; two replicas ≈ 2× ([ADR 0006](0006-per-pod-rate-limit.md)).
- RAM is tight: resource requests stay small; Grafana stays in Cloud.
