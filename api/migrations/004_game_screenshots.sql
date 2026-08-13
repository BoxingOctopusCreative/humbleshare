-- Steam store screenshots (full-size URLs) for game detail carousel.
ALTER TABLE games
    ADD COLUMN screenshot_urls TEXT[] NOT NULL DEFAULT '{}';
