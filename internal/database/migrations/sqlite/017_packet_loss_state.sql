-- Add state tracking to packet loss monitors
ALTER TABLE packet_loss_monitors ADD COLUMN last_state TEXT DEFAULT 'unknown';
ALTER TABLE packet_loss_monitors ADD COLUMN last_state_change TIMESTAMP;

-- Update existing monitors to have an initial state
UPDATE packet_loss_monitors SET last_state = 'unknown' WHERE last_state IS NULL;