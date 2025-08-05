-- Fix column type issues for SQLite users

-- NOTE: SQLite ignores VARCHAR length constraints, so VARCHAR(50) and VARCHAR(255) are functionally identical.
-- We don't need to change monitor_agent_interfaces.name or schedules.interval columns.

-- Note: Issue #94 (packet_loss_monitors.interval) was already fixed in migration 010

-- This migration intentionally left mostly empty as SQLite doesn't enforce VARCHAR lengths