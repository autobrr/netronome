-- Add DNS monitoring tables

-- Table for storing DNS monitor configurations
CREATE TABLE IF NOT EXISTS dns_monitors (
    id SERIAL PRIMARY KEY,
    host TEXT NOT NULL,                          -- resolver host, with an optional :port
    name TEXT,
    protocol TEXT NOT NULL DEFAULT 'udp',        -- udp, tcp, or dot
    query TEXT NOT NULL DEFAULT 'google.com',    -- hostname to ask for
    record_type TEXT NOT NULL DEFAULT 'A',
    interval TEXT NOT NULL DEFAULT '60s',
    enabled BOOLEAN DEFAULT true,
    last_run TIMESTAMP WITH TIME ZONE,
    next_run TIMESTAMP WITH TIME ZONE,
    last_state TEXT DEFAULT 'unknown',
    last_state_change TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing DNS check results
CREATE TABLE IF NOT EXISTS dns_results (
    id SERIAL PRIMARY KEY,
    monitor_id INTEGER NOT NULL,
    response_time_ms DOUBLE PRECISION NOT NULL,
    response_code TEXT NOT NULL DEFAULT '',      -- NOERROR, SERVFAIL, NXDOMAIN, and the like
    success BOOLEAN NOT NULL,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES dns_monitors(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_dns_results_monitor_created_at ON dns_results(monitor_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_dns_results_created_at ON dns_results(created_at);

-- DNS notification events
INSERT INTO notification_events (category, event_type, name, description, supports_threshold, threshold_unit) VALUES
('dns', 'monitor_down', 'DNS Monitor Down', 'DNS resolver did not answer or returned an error', false, NULL),
('dns', 'monitor_recovered', 'DNS Monitor Recovered', 'Previously failing DNS resolver answers again', false, NULL)
ON CONFLICT DO NOTHING;
