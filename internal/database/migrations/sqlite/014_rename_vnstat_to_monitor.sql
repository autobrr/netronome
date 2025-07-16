-- Rename vnstat tables to monitor tables

-- Rename main agent table
ALTER TABLE vnstat_agents RENAME TO monitor_agents;

-- Rename related tables
ALTER TABLE vnstat_agent_system_info RENAME TO monitor_agent_system_info;
ALTER TABLE vnstat_agent_interfaces RENAME TO monitor_agent_interfaces;
ALTER TABLE vnstat_peak_stats RENAME TO monitor_peak_stats;
ALTER TABLE vnstat_resource_stats RENAME TO monitor_resource_stats;
ALTER TABLE vnstat_historical_snapshots RENAME TO monitor_historical_snapshots;

-- Update indexes to match new table names
DROP INDEX IF EXISTS idx_peak_stats_agent_time;
DROP INDEX IF EXISTS idx_resource_stats_agent_time;
DROP INDEX IF EXISTS idx_historical_snapshots;
DROP INDEX IF EXISTS idx_agent_system_info_agent;
DROP INDEX IF EXISTS idx_agent_interfaces_agent;

CREATE INDEX idx_peak_stats_agent_time ON monitor_peak_stats(agent_id, created_at DESC);
CREATE INDEX idx_resource_stats_agent_time ON monitor_resource_stats(agent_id, created_at DESC);
CREATE INDEX idx_historical_snapshots ON monitor_historical_snapshots(agent_id, period_type, created_at DESC);
CREATE INDEX idx_agent_system_info_agent ON monitor_agent_system_info(agent_id);
CREATE INDEX idx_agent_interfaces_agent ON monitor_agent_interfaces(agent_id);