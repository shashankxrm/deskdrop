# ADR 0006: Per-pod token buckets, not a global limiter

## Status

Accepted (B6)

## Context

Pairing create can mint users. Share can fill Postgres. Claim guessing is already capped per IP in process memory (B2). v2 will run two API replicas after D4. A Redis or Postgres limiter is Post-v2.

## Decision

- In-memory **token buckets** keyed by client IP on `POST /v1/pairings` (create) and `POST /v1/links` (share).
- Claim failures stay on the existing sliding window (5 fails / minute → 429).
- CORS allowlist from `CORS_ORIGINS`. No `*` with credentials. Empty env = no ACAO.
- Security headers on every response. HSTS only when D0 serves HTTPS.

## Alternatives

- Postgres-backed limiter: correct globally, extra writes on every share. Post-v2.
- Reverse-proxy limits only (Caddy): another config surface; the API would still be open on the pod network later.
- One global in-memory map on a single replica: breaks the day we add the second replica.

## Consequences

- Effective limit with two pods is up to 2×. Fine for a 20-user beta; document it.
- Process restart resets buckets (brief extra burst).
- Browser dashboard needs `CORS_ORIGINS` set to its Origin (C4 / D0 same-origin behind Caddy can leave it empty).
