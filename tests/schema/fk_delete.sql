-- Unpairing a device must keep its links (created_by_device_id SET NULL).
-- Account delete must then cascade everything. link_deliveries.device_id CASCADE.
-- Run: docker compose exec -T postgres psql -U deskdrop -d deskdrop -v ON_ERROR_STOP=1 < tests/schema/fk_delete.sql

BEGIN;

DO $$
DECLARE
  uid uuid;
  did uuid;
  lid uuid;
  deliv uuid;
BEGIN
  INSERT INTO users DEFAULT VALUES RETURNING id INTO uid;

  INSERT INTO devices (user_id, type, token_hash)
  VALUES (uid, 'laptop', uuid_send(gen_random_uuid()))
  RETURNING id INTO did;

  INSERT INTO links (user_id, created_by_device_id, url, idempotency_key)
  VALUES (uid, did, 'https://example.com/fk-test', 'fk-test-' || uid::text)
  RETURNING id INTO lid;

  INSERT INTO link_deliveries (link_id, device_id)
  VALUES (lid, did)
  RETURNING id INTO deliv;

  DELETE FROM devices WHERE id = did;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'device delete failed';
  END IF;

  IF EXISTS (SELECT 1 FROM devices WHERE id = did) THEN
    RAISE EXCEPTION 'device row still present after delete';
  END IF;

  IF EXISTS (SELECT 1 FROM link_deliveries WHERE id = deliv) THEN
    RAISE EXCEPTION 'delivery did not CASCADE when device was deleted';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM links
    WHERE id = lid AND created_by_device_id IS NULL AND user_id = uid
  ) THEN
    RAISE EXCEPTION 'link must remain with created_by_device_id NULL after device delete';
  END IF;

  DELETE FROM users WHERE id = uid;

  IF NOT FOUND THEN
    RAISE EXCEPTION 'user delete failed';
  END IF;

  IF EXISTS (SELECT 1 FROM users WHERE id = uid)
     OR EXISTS (SELECT 1 FROM devices WHERE user_id = uid)
     OR EXISTS (SELECT 1 FROM links WHERE id = lid)
     OR EXISTS (SELECT 1 FROM pairing_codes WHERE user_id = uid)
     OR EXISTS (SELECT 1 FROM link_deliveries WHERE link_id = lid)
  THEN
    RAISE EXCEPTION 'account delete left leftover rows';
  END IF;
END $$;

ROLLBACK;
