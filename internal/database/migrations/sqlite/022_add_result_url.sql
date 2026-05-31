-- Add result URL column for shareable speed test links (e.g. LibreSpeed share URLs)
ALTER TABLE speed_tests ADD COLUMN result_url TEXT;
