-- Update default retention days from 30 to 365 for vnstat agents
-- This migration only updates the schema default, existing data remains unchanged

-- Since SQLite doesn't support altering column defaults directly,
-- we need to create a new table with the correct default and copy data
-- IMPORTANT: We must preserve foreign key relationships with hourly data

-- Temporarily disable foreign key checks to prevent CASCADE deletion
PRAGMA foreign_keys = OFF;

-- Create new table with updated default
CREATE TABLE vnstat_agents_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL,
    enabled BOOLEAN DEFAULT 1,
    interface VARCHAR(50),
    retention_days INTEGER DEFAULT 365,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Copy existing data with EXACT same IDs to preserve foreign key relationships
INSERT INTO vnstat_agents_new (id, name, url, enabled, interface, retention_days, created_at, updated_at)
SELECT id, name, url, enabled, interface, retention_days, created_at, updated_at
FROM vnstat_agents;

-- Drop old table (this won't cascade delete because foreign keys are disabled)
DROP TABLE vnstat_agents;

-- Rename new table
ALTER TABLE vnstat_agents_new RENAME TO vnstat_agents;

-- Re-enable foreign key checks
PRAGMA foreign_keys = ON;