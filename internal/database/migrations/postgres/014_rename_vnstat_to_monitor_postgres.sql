-- Rename vnstat tables to monitor tables

-- Rename main agent table
ALTER TABLE vnstat_agents RENAME TO monitor_agents;

-- Rename related tables
ALTER TABLE vnstat_agent_system_info RENAME TO monitor_agent_system_info;
ALTER TABLE vnstat_agent_interfaces RENAME TO monitor_agent_interfaces;
ALTER TABLE vnstat_bandwidth_samples RENAME TO monitor_bandwidth_samples;
ALTER TABLE vnstat_peak_stats RENAME TO monitor_peak_stats;
ALTER TABLE vnstat_resource_stats RENAME TO monitor_resource_stats;
ALTER TABLE vnstat_historical_snapshots RENAME TO monitor_historical_snapshots;

-- PostgreSQL will automatically update indexes when tables are renamed
-- But we should rename indexes for clarity
ALTER INDEX IF EXISTS idx_bandwidth_samples_agent_time RENAME TO idx_monitor_bandwidth_samples_agent_time;
ALTER INDEX IF EXISTS idx_peak_stats_agent_time RENAME TO idx_monitor_peak_stats_agent_time;
ALTER INDEX IF EXISTS idx_resource_stats_agent_time RENAME TO idx_monitor_resource_stats_agent_time;
ALTER INDEX IF EXISTS idx_historical_snapshots RENAME TO idx_monitor_historical_snapshots;
ALTER INDEX IF EXISTS idx_agent_system_info_agent RENAME TO idx_monitor_agent_system_info_agent;
ALTER INDEX IF EXISTS idx_agent_interfaces_agent RENAME TO idx_monitor_agent_interfaces_agent;