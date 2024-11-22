CREATE TABLE IF NOT EXISTS speed_tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_name TEXT NOT NULL,
    server_id TEXT NOT NULL,
    download_speed REAL NOT NULL,
    upload_speed REAL NOT NULL,
    latency TEXT NOT NULL,
    packet_loss REAL,
    jitter REAL,
    cv REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_ids TEXT NOT NULL,
    interval TEXT NOT NULL,
    last_run TIMESTAMP,
    next_run TIMESTAMP NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    options TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);