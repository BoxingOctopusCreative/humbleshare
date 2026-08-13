-- Users, sessions, access requests/invites, OAuth identities, claim user FK.

CREATE TABLE users (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                TEXT NOT NULL,
    password_hash        TEXT NOT NULL,
    display_name         TEXT NOT NULL DEFAULT '',
    role                 TEXT NOT NULL DEFAULT 'member'
                             CHECK (role IN ('member', 'admin', 'superadmin')),
    two_factor_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_secret    TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email_lower ON users (lower(email));

CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE access_requests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT NOT NULL,
    note         TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'approved', 'denied')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ,
    resolved_by  UUID REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX idx_access_requests_pending_email
    ON access_requests (lower(email))
    WHERE status = 'pending';

CREATE TABLE access_invites (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    role         TEXT NOT NULL DEFAULT 'member'
                     CHECK (role IN ('member', 'admin', 'superadmin')),
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ,
    created_by   UUID REFERENCES users (id) ON DELETE SET NULL,
    request_id   UUID REFERENCES access_requests (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_invites_email_open
    ON access_invites (lower(email))
    WHERE consumed_at IS NULL;

CREATE TABLE oauth_identities (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider          TEXT NOT NULL CHECK (provider IN ('google', 'discord', 'steam')),
    provider_user_id  TEXT NOT NULL,
    provider_email    TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_oauth_identities_user_id ON oauth_identities (user_id);

ALTER TABLE codes
    ADD COLUMN claimed_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL;

CREATE INDEX idx_codes_claimed_by_user_id ON codes (claimed_by_user_id)
    WHERE claimed_by_user_id IS NOT NULL;
