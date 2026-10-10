# ADR 0001: Postgres as the delivery queue

## Status

Accepted (A2)

## Context

v1 queued undelivered links in Redis and pushed over Socket.IO. The free host slept, sockets dropped, and the server treated “queued or emitted” as delivered. v2 needs a queue that survives API restarts and laptop sleep, on one Always Free VM, with no Redis.

## Decision

Pending links are rows in PostgreSQL (`links` + `link_deliveries`). The laptop agent pulls and acks. Postgres is the system of record and the queue.

## Alternatives

- **Redis lists** — fast, but another process to back up and another failure mode. v1 already lost the plot here. Rejected for v2.
- **SQS / NATS / a cloud queue** — not ₹0, and the consumer is a sleeping laptop, not a worker fleet.
- **In-process memory** — dies on restart; cannot run two API replicas later.

## Consequences

- At-least-once delivery is `SELECT` pending rows + idempotent `UPDATE` on ack, not a broker feature.
- Backups (D6) cover the queue; a Redis dump would have been a second restore drill.
- Under huge write load Postgres would become the bottleneck first (`docs/scaling.md` at D0). For 10–20 users this is the right tradeoff.
