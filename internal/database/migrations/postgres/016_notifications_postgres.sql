-- Create notification channels table (stores Shoutrrr URLs)
CREATE TABLE IF NOT EXISTS notification_channels (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create notification events table (defines what can trigger notifications)
CREATE TABLE IF NOT EXISTS notification_events (
    id SERIAL PRIMARY KEY,
    category TEXT NOT NULL, -- speedtest, packetloss, agent, etc.
    event_type TEXT NOT NULL, -- complete, threshold_exceeded, offline, etc.
    name TEXT NOT NULL,
    description TEXT,
    default_enabled BOOLEAN NOT NULL DEFAULT false,
    supports_threshold BOOLEAN NOT NULL DEFAULT false,
    threshold_unit TEXT, -- Mbps, ms, %, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create notification rules table (user's configuration)
CREATE TABLE IF NOT EXISTS notification_rules (
    id SERIAL PRIMARY KEY,
    channel_id INTEGER NOT NULL,
    event_id INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    threshold_value DOUBLE PRECISION,
    threshold_operator TEXT, -- 'gt', 'lt', 'eq', 'gte', 'lte'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES notification_events(id) ON DELETE CASCADE,
    UNIQUE(channel_id, event_id)
);

-- Create notification history table (track sent notifications)
CREATE TABLE IF NOT EXISTS notification_history (
    id SERIAL PRIMARY KEY,
    channel_id INTEGER NOT NULL,
    event_id INTEGER NOT NULL,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    payload TEXT, -- JSON payload that was sent
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES notification_events(id) ON DELETE CASCADE
);

-- Insert default notification events
INSERT INTO notification_events (category, event_type, name, description, supports_threshold, threshold_unit) VALUES
-- Speedtest events
('speedtest', 'complete', 'Speed Test Complete', 'Notification when any speed test completes', false, NULL),
('speedtest', 'ping_high', 'High Ping', 'Ping exceeds threshold', true, 'ms'),
('speedtest', 'download_low', 'Low Download Speed', 'Download speed below threshold', true, 'Mbps'),
('speedtest', 'upload_low', 'Low Upload Speed', 'Upload speed below threshold', true, 'Mbps'),
('speedtest', 'failed', 'Speed Test Failed', 'Notification when a speed test fails', false, NULL),

-- Packet loss events
('packetloss', 'threshold_exceeded', 'High Packet Loss', 'Packet loss exceeds threshold', true, '%'),
('packetloss', 'monitor_down', 'Monitor Unreachable', 'Packet loss monitor target is unreachable', false, NULL),
('packetloss', 'monitor_recovered', 'Monitor Recovered', 'Previously unreachable monitor is back online', false, NULL),

-- Agent events
('agent', 'offline', 'Agent Offline', 'Agent has gone offline', false, NULL),
('agent', 'online', 'Agent Online', 'Agent has come back online', false, NULL),
('agent', 'high_bandwidth', 'High Bandwidth Usage', 'Bandwidth usage exceeds threshold', true, 'Mbps'),
('agent', 'disk_space_low', 'Low Disk Space', 'Available disk space below threshold', true, '%'),
('agent', 'cpu_high', 'High CPU Usage', 'CPU usage exceeds threshold', true, '%'),
('agent', 'memory_high', 'High Memory Usage', 'Memory usage exceeds threshold', true, '%')
ON CONFLICT DO NOTHING;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_notification_rules_channel ON notification_rules(channel_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_event ON notification_rules(event_id);
CREATE INDEX IF NOT EXISTS idx_notification_history_created ON notification_history(created_at);

-- Create function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to update timestamps
DROP TRIGGER IF EXISTS update_notification_channels_updated_at ON notification_channels;
CREATE TRIGGER update_notification_channels_updated_at 
BEFORE UPDATE ON notification_channels 
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_notification_rules_updated_at ON notification_rules;
CREATE TRIGGER update_notification_rules_updated_at 
BEFORE UPDATE ON notification_rules 
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();