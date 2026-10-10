# ADR 0004: At-least-once delivery and idempotent ack

## Status

Accepted (B4)

## Context

The laptop agent opens the default browser, then acks. Networks drop. Agents retry. Android share targets retry. We store the queue in Postgres ([ADR 0001](0001-postgres-as-queue.md)), not a broker.

## Decision

- **Share** requires `Idempotency-Key`. Unique `(user_id, idempotency_key)`: a replay returns the existing link (200) and does not insert or re-fan-out.
- **Fan-out at share time** only: one `link_deliveries` row per non-revoked `laptop` device of that user. A laptop paired later does not receive older links (history is B5).
- **Pull** is `SELECT` unacked deliveries for **this** device. Phone and web get 403.
- **Ack** sets `acked_at` once. Repeating ack is 204. We do not delete the row here (B5 retention).
- Delivery is **at-least-once**: if the agent opens the browser and then fails to ack, the next pull still returns the link. The agent (C3) must tolerate a second open.

## Alternatives

- At-most-once (ack before open): a crash after ack never opens the URL. Worse for a “send this to my laptop” tool.
- Re-fan-out on every pull for newly paired laptops: surprising duplicates for old shares; extra writes on the poll path.
- Exactly-once with a broker: out of v2 scope (no Redis/queues).

## Consequences

- Two laptops of the same user each get their own delivery row; acking one does not ack the other.
- Prometheus: shares count inserts, not replays; acks count every 204 including repeats.
