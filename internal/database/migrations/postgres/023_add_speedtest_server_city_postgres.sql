ALTER TABLE speed_tests ADD COLUMN server_city VARCHAR(255);

CREATE TABLE speedtest_servers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    country TEXT NOT NULL,
    sponsor TEXT NOT NULL,
    url TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE speedtest_server_sources (
    source_key TEXT PRIMARY KEY,
    updated_at TIMESTAMPTZ NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    CHECK ((latitude IS NULL) = (longitude IS NULL))
);
