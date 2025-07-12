-- Update packet loss monitors to support flexible scheduling like speed test schedules

-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table
-- First, create a new table with the updated schema
CREATE TABLE packet_loss_monitors_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    interval TEXT NOT NULL DEFAULT '60s', -- Changed from INTEGER to TEXT, default 60 seconds
    packet_count INTEGER DEFAULT 10,
    enabled BOOLEAN DEFAULT 1,
    threshold REAL DEFAULT 5.0,
    last_run TIMESTAMP,          -- New column to track last execution
    next_run TIMESTAMP,          -- New column to track next scheduled run
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Copy existing data, converting interval from seconds to duration string
INSERT INTO packet_loss_monitors_new (id, host, name, interval, packet_count, enabled, threshold, created_at, updated_at)
SELECT 
    id, 
    host, 
    name,
    CASE 
        WHEN interval < 60 THEN interval || 's'
        WHEN interval < 3600 THEN (interval / 60) || 'm'
        WHEN interval < 86400 THEN (interval / 3600) || 'h'
        ELSE (interval / 86400) || 'd'
    END as interval,
    packet_count,
    enabled,
    threshold,
    created_at,
    updated_at
FROM packet_loss_monitors;

-- Drop the old table
DROP TABLE packet_loss_monitors;

-- Rename the new table
ALTER TABLE packet_loss_monitors_new RENAME TO packet_loss_monitors;

-- Recreate indexes
CREATE INDEX idx_packet_loss_monitors_host ON packet_loss_monitors(host);
CREATE INDEX idx_packet_loss_monitors_enabled ON packet_loss_monitors(enabled);
CREATE INDEX idx_packet_loss_monitors_next_run ON packet_loss_monitors(next_run);