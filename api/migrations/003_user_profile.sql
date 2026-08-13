-- Profile fields: bio, gamertags, avatar URL (R2 public URL).

ALTER TABLE users
    ADD COLUMN bio TEXT NOT NULL DEFAULT '',
    ADD COLUMN steam_gamertag TEXT NOT NULL DEFAULT '',
    ADD COLUMN discord_tag TEXT NOT NULL DEFAULT '',
    ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';
