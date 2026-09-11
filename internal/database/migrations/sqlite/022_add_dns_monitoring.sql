-- Add DNS monitoring tables

-- Table for storing DNS monitor configurations
CREATE TABLE dns_monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host TEXT NOT NULL,                          -- resolver host, with an optional :port
    name TEXT,
    protocol TEXT NOT NULL DEFAULT 'udp',        -- udp, tcp, or dot
    query TEXT NOT NULL DEFAULT 'google.com',    -- hostname to ask for
    record_type TEXT NOT NULL DEFAULT 'A',
    interval TEXT NOT NULL DEFAULT '60s',
    enabled BOOLEAN DEFAULT 1,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    last_state TEXT DEFAULT 'unknown',
    last_state_change TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing DNS check results
CREATE TABLE dns_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    response_time_ms REAL NOT NULL,
    response_code TEXT NOT NULL DEFAULT '',      -- NOERROR, SERVFAIL, NXDOMAIN, and the like
    success BOOLEAN NOT NULL,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES dns_monitors(id) ON DELETE CASCADE
);

CREATE INDEX idx_dns_results_monitor_created_at ON dns_results(monitor_id, created_at DESC, id DESC);
CREATE INDEX idx_dns_results_created_at ON dns_results(created_at);

-- DNS notification events
INSERT INTO notification_events (category, event_type, name, description, supports_threshold, threshold_unit) VALUES
('dns', 'monitor_down', 'DNS Monitor Down', 'DNS resolver did not answer or returned an error', 0, NULL),
('dns', 'monitor_recovered', 'DNS Monitor Recovered', 'Previously failing DNS resolver answers again', 0, NULL);
