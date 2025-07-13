-- Add vnstat hourly aggregation table

-- Table for storing hourly aggregated bandwidth data
CREATE TABLE vnstat_bandwidth_hourly (
    agent_id INTEGER NOT NULL,
    hour_start TIMESTAMP NOT NULL,
    total_rx_bytes BIGINT DEFAULT 0,
    total_tx_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, hour_start),
    FOREIGN KEY (agent_id) REFERENCES vnstat_agents(id) ON DELETE CASCADE
);

-- Create index for better query performance
CREATE INDEX idx_vnstat_bandwidth_hourly_agent_hour ON vnstat_bandwidth_hourly(agent_id, hour_start);
CREATE INDEX idx_vnstat_bandwidth_hourly_hour ON vnstat_bandwidth_hourly(hour_start);