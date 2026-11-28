ALTER TABLE speedtest_server_sources ADD COLUMN failed_regions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE speedtest_server_sources ADD COLUMN total_regions INTEGER NOT NULL DEFAULT 0;
