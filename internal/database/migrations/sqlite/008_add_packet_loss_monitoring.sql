-- Add packet loss monitoring tables

-- Table for storing packet loss monitor configurations
CREATE TABLE packet_loss_monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    interval INTEGER DEFAULT 60, -- interval in seconds
    packet_count INTEGER DEFAULT 10, -- number of packets per test
    enabled BOOLEAN DEFAULT 1,
    threshold REAL DEFAULT 5.0, -- packet loss threshold percentage for alerts
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing packet loss test results
CREATE TABLE packet_loss_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
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