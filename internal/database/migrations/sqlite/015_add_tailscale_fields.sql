-- Add Tailscale support fields to monitor agents

ALTER TABLE monitor_agents ADD COLUMN is_tailscale BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE monitor_agents ADD COLUMN tailscale_hostname TEXT;
ALTER TABLE monitor_agents ADD COLUMN discovered_at TIMESTAMP;