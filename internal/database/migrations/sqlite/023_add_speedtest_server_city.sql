ALTER TABLE speed_tests ADD COLUMN server_city TEXT;

CREATE TABLE speedtest_servers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    country TEXT NOT NULL,
    sponsor TEXT NOT NULL,
    url TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    observed_at DATETIME NOT NULL
);

CREATE TABLE speedtest_server_sources (
    source_key TEXT PRIMARY KEY,
    updated_at DATETIME NOT NULL,
    latitude REAL,
    longitude REAL,
    CHECK ((latitude IS NULL) = (longitude IS NULL))
);
