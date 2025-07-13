-- Add vnstat monitoring tables

-- Table for storing vnstat agent configurations
CREATE TABLE vnstat_agents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    interface VARCHAR(50),
    retention_days INTEGER DEFAULT 30,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing bandwidth history
CREATE TABLE vnstat_bandwidth (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id INTEGER NOT NULL,
    rx_bytes_per_second INTEGER,
    tx_bytes_per_second INTEGER,
    rx_packets_per_second INTEGER,
    tx_packets_per_second INTEGER,
    rx_rate_string VARCHAR(50),
    tx_rate_string VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES vnstat_agents(id) ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX idx_vnstat_agents_enabled ON vnstat_agents(enabled);
CREATE INDEX idx_vnstat_bandwidth_agent_id ON vnstat_bandwidth(agent_id);
CREATE INDEX idx_vnstat_bandwidth_created_at ON vnstat_bandwidth(created_at);
CREATE INDEX idx_vnstat_bandwidth_agent_time ON vnstat_bandwidth(agent_id, created_at);