-- Add vnstat agent configuration table

-- Table for storing vnstat agent configurations
CREATE TABLE vnstat_agents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    interface VARCHAR(50),
    retention_days INTEGER DEFAULT 365,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for better query performance
CREATE INDEX idx_vnstat_agents_enabled ON vnstat_agents(enabled);