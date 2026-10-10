# ADR 0005: Cursor history and 30-day TTL

## Status

Accepted (B5)

## Context

The dashboard needs a share history that does not jump or duplicate when new shares arrive mid-scroll. Delivered rows cannot grow forever on an Always Free VM. OFFSET pagination is unstable under inserts.

## Decision

- `GET /v1/links` lists the user’s links newest-first, keyed by `(created_at DESC, id DESC)` (index from A2).
- The page cursor is an opaque token of that pair. The next page is `(created_at, id) < cursor`. New shares are newer than the cursor, so they appear only on a fresh first page.
- Default `limit` 20, max 100.
- Retention (hourly in the API process): delete `link_deliveries` with `acked_at` older than 30 days; delete `links` older than 30 days that have **no unacked** delivery. Pending rows stay.
- `DELETE /v1/account` deletes the `users` row; FKs cascade. The web cookie is cleared.

## Alternatives

- OFFSET/LIMIT: a share during scroll skips or duplicates a row.
- Purge unacked after 30 days: a sleeping laptop would lose the URL. Rejected.
- Separate worker process: extra Compose service for a DELETE. The API already holds the pool.

## Consequences

- History after 30 days is gone even if the dashboard never opened it. Documented for C4.
- Two API replicas both run the hourly DELETE; it is idempotent.
