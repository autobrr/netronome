-- Add API key authentication to vnstat agents

-- Add api_key column to vnstat_agents table
ALTER TABLE vnstat_agents ADD COLUMN api_key VARCHAR(255);