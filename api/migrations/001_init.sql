-- Postgres 13+: gen_random_uuid() is in core (no pgcrypto). Managed DBs often
-- deny CREATE EXTENSION to the app role.

CREATE TABLE games (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    steam_appid          INTEGER UNIQUE,
    name                 TEXT NOT NULL,
    header_image_url     TEXT,
    capsule_image_url    TEXT,
    short_description    TEXT,
    store_url            TEXT,
    metadata_fetched_at  TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE codes (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id          UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    code_type        TEXT NOT NULL CHECK (code_type IN ('steam_key', 'gift_link')),
    code_value       TEXT NOT NULL,
    source           TEXT,
    source_order_id  TEXT,
    status           TEXT NOT NULL DEFAULT 'available'
                         CHECK (status IN ('available', 'claimed', 'dead')),
    claimed_by       TEXT,
    claimed_at       TIMESTAMPTZ,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_codes_code_value ON codes (code_value);
CREATE INDEX idx_codes_status ON codes (status);
CREATE INDEX idx_codes_game_id ON codes (game_id);
CREATE INDEX idx_codes_game_available ON codes (game_id, created_at ASC)
    WHERE status = 'available';
