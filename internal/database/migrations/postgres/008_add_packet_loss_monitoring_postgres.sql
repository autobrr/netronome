-- Add packet loss monitoring tables

-- Table for storing packet loss monitor configurations
CREATE TABLE packet_loss_monitors (
    id SERIAL PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    interval INTEGER DEFAULT 60, -- interval in seconds
    packet_count INTEGER DEFAULT 10, -- number of packets per test
    enabled BOOLEAN DEFAULT true,
    threshold REAL DEFAULT 5.0, -- packet loss threshold percentage for alerts
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing packet loss test results
CREATE TABLE packet_loss_results (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL,
    packet_loss REAL NOT NULL,
    min_rtt REAL,
    max_rtt REAL,
    avg_rtt REAL,
    std_dev_rtt REAL,
    packets_sent INTEGER,
    packets_recv INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES packet_loss_monitors(id) ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX idx_packet_loss_monitors_host ON packet_loss_monitors(host);
CREATE INDEX idx_packet_loss_monitors_enabled ON packet_loss_monitors(enabled);
CREATE INDEX idx_packet_loss_results_monitor_id ON packet_loss_results(monitor_id);
CREATE INDEX idx_packet_loss_results_created_at ON packet_loss_results(created_at);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_packet_loss_monitors_updated_at BEFORE UPDATE
    ON packet_loss_monitors FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();