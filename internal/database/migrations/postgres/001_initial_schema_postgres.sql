CREATE TABLE IF NOT EXISTS speed_tests (
    id SERIAL PRIMARY KEY,
    server_name VARCHAR(255) NOT NULL,
    server_id VARCHAR(255) NOT NULL,
    download_speed REAL NOT NULL,
    upload_speed REAL NOT NULL,
    latency VARCHAR(50) NOT NULL,
    packet_loss REAL,
    jitter REAL,
    cv REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS schedules (
    id SERIAL PRIMARY KEY,
    server_ids TEXT NOT NULL,
    interval VARCHAR(50) NOT NULL,
    last_run TIMESTAMP,
    next_run TIMESTAMP NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    options TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
