CREATE TABLE users (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE devices (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name          text NOT NULL DEFAULT '',
    type          text NOT NULL,
    token_hash    bytea NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz,
    revoked_at    timestamptz,
    CONSTRAINT devices_type_check CHECK (type IN ('phone', 'laptop', 'web')),
    CONSTRAINT devices_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX devices_user_id_idx ON devices (user_id);

CREATE TABLE pairing_codes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash   bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pairing_codes_code_hash_unique UNIQUE (code_hash)
);

CREATE INDEX pairing_codes_user_id_idx ON pairing_codes (user_id);

CREATE TABLE links (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_by_device_id  uuid REFERENCES devices (id) ON DELETE SET NULL,
    url                   text NOT NULL,
    title                 text,
    idempotency_key       text NOT NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT links_user_idempotency_unique UNIQUE (user_id, idempotency_key)
);

CREATE INDEX links_user_created_id_idx ON links (user_id, created_at, id);

CREATE TABLE link_deliveries (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id     uuid NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    device_id   uuid NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    acked_at    timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT link_deliveries_link_device_unique UNIQUE (link_id, device_id)
);

CREATE INDEX link_deliveries_pending_by_device_idx
    ON link_deliveries (device_id)
    WHERE acked_at IS NULL;
