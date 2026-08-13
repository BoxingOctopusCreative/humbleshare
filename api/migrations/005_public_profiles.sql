-- Public profiles: slug URL, visibility toggle, hero banner.

ALTER TABLE users
    ADD COLUMN profile_slug TEXT NOT NULL DEFAULT '',
    ADD COLUMN profile_public BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN banner_url TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_users_profile_slug
    ON users (profile_slug) WHERE profile_slug <> '';
