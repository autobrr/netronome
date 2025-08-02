-- Fix VARCHAR column limits for PostgreSQL users

-- Issue #91: Increase schedules interval column to support multiple exact times
ALTER TABLE schedules ALTER COLUMN interval TYPE TEXT;

-- Issue #92: Increase monitor_agent_interfaces name column for long interface names
ALTER TABLE monitor_agent_interfaces ALTER COLUMN name TYPE VARCHAR(255);

-- Note: Issue #94 (packet_loss_monitors.interval) was already fixed in migration 010