-- Add tables for storing vnstat agent data

-- Table for static/semi-static system information
CREATE TABLE vnstat_agent_system_info (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES vnstat_agents(id) ON DELETE CASCADE,
    hostname VARCHAR(255),
    kernel VARCHAR(255),
    vnstat_version VARCHAR(50),
    cpu_model VARCHAR(255),
    cpu_cores INTEGER,
    cpu_threads INTEGER,
    total_memory BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id)
);

-- Table for network interface configuration
CREATE TABLE vnstat_agent_interfaces (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES vnstat_agents(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    alias VARCHAR(255),
    ip_address VARCHAR(45),
    link_speed INTEGER, -- Mbps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, name)
);

-- Table for historical peak bandwidth
CREATE TABLE vnstat_peak_stats (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES vnstat_agents(id) ON DELETE CASCADE,
    peak_rx_bytes BIGINT,
    peak_tx_bytes BIGINT,
    peak_rx_timestamp TIMESTAMP,
    peak_tx_timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for hardware resource usage
CREATE TABLE vnstat_resource_stats (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES vnstat_agents(id) ON DELETE CASCADE,
    cpu_usage_percent REAL,
    memory_used_percent REAL,
    swap_used_percent REAL,
    disk_usage_json TEXT, -- JSON array of disk usage
    temperature_json TEXT, -- JSON array of temperature sensors
    uptime_seconds BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for vnstat native data snapshots
CREATE TABLE vnstat_historical_snapshots (
    id SERIAL PRIMARY KEY,
    agent_id INTEGER NOT NULL REFERENCES vnstat_agents(id) ON DELETE CASCADE,
    interface_name VARCHAR(50),
    period_type VARCHAR(10), -- 'hourly', 'daily', 'monthly'
    data_json TEXT, -- Compressed JSON of vnstat native data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_peak_stats_agent_time ON vnstat_peak_stats(agent_id, created_at DESC);
CREATE INDEX idx_resource_stats_agent_time ON vnstat_resource_stats(agent_id, created_at DESC);
CREATE INDEX idx_historical_snapshots ON vnstat_historical_snapshots(agent_id, period_type, created_at DESC);
CREATE INDEX idx_agent_system_info_agent ON vnstat_agent_system_info(agent_id);
CREATE INDEX idx_agent_interfaces_agent ON vnstat_agent_interfaces(agent_id);