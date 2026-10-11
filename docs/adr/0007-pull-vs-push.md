# ADR 0007: Pull, not push

## Status

Accepted (B7)

## Context

v1 pushed over Socket.IO and treated “emitted or queued in Redis” as delivered. Laptops sleep. Always Free VMs sleep or reboot. The consumer is one tray agent per user, not a worker fleet. Postgres is already the queue ([ADR 0001](0001-postgres-as-queue.md)).

## Decision

The laptop **pulls** `GET /v1/links/pending` and **acks** `POST /v1/links/{id}/ack`. The API does not hold a socket, does not emit, and does not mark delivered on share. C3 polls on start, every 10s, and on wake (the ticker resumes).

## Alternatives

- WebSocket / SSE push: reconnect storms after sleep; another moving part; v1 already failed this way.
- FCM / APNs: paid or extra accounts; still need a durable row if the push is dropped.
- Long poll: more open connections on a 1–4 OCPU VM for no user-visible gain at 20 users.

## Consequences

- Latency is poll interval, not instant. Fine for “open this when I sit down.”
- Sleep-mid-ack is a pending row, not a lost link ([what-breaks-if.md](../what-breaks-if.md)).
- Two laptops each pull their own `link_deliveries` row.
