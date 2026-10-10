# Database schema

Postgres 16 holds users, devices, pairing codes, links, and per-laptop deliveries. Pending deliveries *are* the queue (see [ADR 0001](../adr/0001-postgres-as-queue.md)). The API does not talk to these tables yet (B1+); A2 only creates them and checks `/healthz`.

Local Compose uses user/password/db `deskdrop` / `deskdrop` / `deskdrop` on `localhost:5432`. That password is **local only**. Production secrets come from env later, never from this file.

## Tables (migration `000001_init`)

- **users** — one row per pairing chain (`id`, `created_at`).
- **devices** — `type` in `phone` | `laptop` | `web`; `token_hash` unique (HMAC of the device token, written in B2); `revoked_at` nullable.
- **pairing_codes** — `code_hash` unique, `expires_at`, `used_at` nullable.
- **links** — `url`, optional `title`, `created_by_device_id` nullable, unique `(user_id, idempotency_key)`.
- **link_deliveries** — one row per laptop device per link; `acked_at` null means pending.

Indexes: `devices(user_id)`, `pairing_codes(user_id)`, `links(user_id, created_at, id)` for cursor history, partial `link_deliveries(device_id) WHERE acked_at IS NULL` for pull.

Deletes cascade from `users`. Unpairing a device sets `links.created_by_device_id` to NULL (`ON DELETE SET NULL`) so history remains; `link_deliveries.device_id` is `ON DELETE CASCADE` so that laptop’s pending/acked rows go away with the device.

## Verify or demo

From the repo root. Docker required.

```bash
# Start Postgres, run migrations, start the /healthz placeholder.
docker compose up --build -d

# API placeholder.
curl -sS http://127.0.0.1:8080/healthz
# Expect: ok

# Tables exist.
docker compose exec postgres \
  psql -U deskdrop -d deskdrop -c '\dt'
# Expect: users, devices, pairing_codes, links, link_deliveries

docker compose exec postgres \
  psql -U deskdrop -d deskdrop -c '\d link_deliveries'
# Expect: unique (link_id, device_id) and index link_deliveries_pending_by_device_idx

# Migrate down then up twice (schema returns).
docker compose run --rm migrate down 1
docker compose exec postgres \
  psql -U deskdrop -d deskdrop -c '\dt'
# Expect: no application tables (schema_migrations remains)

docker compose run --rm migrate up
docker compose run --rm migrate down 1
docker compose run --rm migrate up

docker compose exec postgres \
  psql -U deskdrop -d deskdrop -c '\dt'

# FK behaviour: delete device (link remains, created_by NULL, deliveries CASCADE),
# then delete user (zero leftover rows). Script rolls back.
docker compose exec -T postgres \
  psql -U deskdrop -d deskdrop -v ON_ERROR_STOP=1 \
  < tests/schema/fk_delete.sql
# Expect: COMMIT/ROLLBACK with no EXCEPTION (psql prints ROLLBACK)

docker compose exec postgres \
  psql -U deskdrop -d deskdrop -c '\d links'
# Expect: created_by_device_id nullable, ON DELETE SET NULL

docker compose exec postgres \
  psql -U deskdrop -d deskdrop -c '\d link_deliveries'
# Expect: device_id ON DELETE CASCADE

# Stop (keeps the named volume). Add -v to wipe data.
docker compose down
```
