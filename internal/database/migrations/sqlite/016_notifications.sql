-- Create notification channels table (stores Shoutrrr URLs)
CREATE TABLE IF NOT EXISTS notification_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create notification events table (defines what can trigger notifications)
CREATE TABLE IF NOT EXISTS notification_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL, -- speedtest, packetloss, agent, etc.
    event_type TEXT NOT NULL, -- complete, threshold_exceeded, offline, etc.
    name TEXT NOT NULL,
    description TEXT,
    default_enabled BOOLEAN NOT NULL DEFAULT 0,
    supports_threshold BOOLEAN NOT NULL DEFAULT 0,
    threshold_unit TEXT, -- Mbps, ms, %, etc.
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create notification rules table (user's configuration)
CREATE TABLE IF NOT EXISTS notification_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL,
    event_id INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    threshold_value REAL,
    threshold_operator TEXT, -- 'gt', 'lt', 'eq', 'gte', 'lte'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES notification_events(id) ON DELETE CASCADE,
    UNIQUE(channel_id, event_id)
);

-- Create notification history table (track sent notifications)
CREATE TABLE IF NOT EXISTS notification_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL,
    event_id INTEGER NOT NULL,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    payload TEXT, -- JSON payload that was sent
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES notification_events(id) ON DELETE CASCADE
);

-- Insert default notification events
INSERT INTO notification_events (category, event_type, name, description, supports_threshold, threshold_unit) VALUES
-- Speedtest events
('speedtest', 'complete', 'Speed Test Complete', 'Notification when any speed test completes', 0, NULL),
('speedtest', 'ping_high', 'High Ping', 'Ping exceeds threshold', 1, 'ms'),
('speedtest', 'download_low', 'Low Download Speed', 'Download speed below threshold', 1, 'Mbps'),
('speedtest', 'upload_low', 'Low Upload Speed', 'Upload speed below threshold', 1, 'Mbps'),
('speedtest', 'failed', 'Speed Test Failed', 'Notification when a speed test fails', 0, NULL),

-- Packet loss events
('packetloss', 'threshold_exceeded', 'High Packet Loss', 'Packet loss exceeds threshold', 1, '%'),
('packetloss', 'monitor_down', 'Monitor Unreachable', 'Packet loss monitor target is unreachable', 0, NULL),
('packetloss', 'monitor_recovered', 'Monitor Recovered', 'Previously unreachable monitor is back online', 0, NULL),

-- Agent events
('agent', 'offline', 'Agent Offline', 'Agent has gone offline', 0, NULL),
('agent', 'online', 'Agent Online', 'Agent has come back online', 0, NULL),
('agent', 'high_bandwidth', 'High Bandwidth Usage', 'Bandwidth usage exceeds threshold', 1, 'Mbps'),
('agent', 'disk_space_low', 'Low Disk Space', 'Available disk space below threshold', 1, '%'),
('agent', 'cpu_high', 'High CPU Usage', 'CPU usage exceeds threshold', 1, '%'),
('agent', 'memory_high', 'High Memory Usage', 'Memory usage exceeds threshold', 1, '%');

-- Create indexes for better performance
CREATE INDEX idx_notification_rules_channel ON notification_rules(channel_id);
CREATE INDEX idx_notification_rules_event ON notification_rules(event_id);
CREATE INDEX idx_notification_history_created ON notification_history(created_at);

-- Create triggers to update timestamps
CREATE TRIGGER update_notification_channels_timestamp 
AFTER UPDATE ON notification_channels
BEGIN
    UPDATE notification_channels SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_notification_rules_timestamp 
AFTER UPDATE ON notification_rules
BEGIN
    UPDATE notification_rules SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Add state tracking to packet loss monitors
ALTER TABLE packet_loss_monitors ADD COLUMN last_state TEXT DEFAULT 'unknown';
ALTER TABLE packet_loss_monitors ADD COLUMN last_state_change TIMESTAMP;

-- Update existing monitors to have an initial state
UPDATE packet_loss_monitors SET last_state = 'unknown' WHERE last_state IS NULL;

-- Remove packet_loss from speed_tests table
-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- Create new table without packet_loss column
CREATE TABLE speed_tests_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_name TEXT NOT NULL,
    server_id TEXT NOT NULL,
    download_speed REAL NOT NULL,
    upload_speed REAL NOT NULL,
    latency TEXT NOT NULL,
    jitter REAL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    test_type TEXT,
    is_scheduled BOOLEAN DEFAULT 0,
    server_host TEXT
);

-- Copy data from old table (excluding packet_loss and cv)
-- Note: cv column may or may not exist depending on SQLite version (DROP COLUMN was added in 3.35.0)
-- This query handles both cases by explicitly selecting only the columns we want
INSERT INTO speed_tests_new (id, server_name, server_id, download_speed, upload_speed, latency, jitter, created_at, test_type, is_scheduled, server_host)
SELECT id, server_name, server_id, download_speed, upload_speed, latency, jitter, created_at, test_type, is_scheduled, server_host
FROM speed_tests;

-- Drop old table
DROP TABLE speed_tests;

-- Rename new table
ALTER TABLE speed_tests_new RENAME TO speed_tests;