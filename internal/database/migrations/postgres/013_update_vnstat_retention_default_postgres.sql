-- Update default retention days from 30 to 365 for vnstat agents
-- This migration only updates the schema default, existing data remains unchanged

-- PostgreSQL allows altering column defaults directly
ALTER TABLE vnstat_agents ALTER COLUMN retention_days SET DEFAULT 365;