-- Update packet loss monitors to support flexible scheduling like speed test schedules

-- Alter the interval column from INTEGER to TEXT
ALTER TABLE packet_loss_monitors 
ALTER COLUMN interval TYPE TEXT USING 
    CASE 
        WHEN interval < 60 THEN interval || 's'
        WHEN interval < 3600 THEN (interval / 60) || 'm'
        WHEN interval < 86400 THEN (interval / 3600) || 'h'
        ELSE (interval / 86400) || 'd'
    END;

-- Set default value for interval
ALTER TABLE packet_loss_monitors 
ALTER COLUMN interval SET DEFAULT '60s';

-- Add new columns for scheduling
ALTER TABLE packet_loss_monitors 
ADD COLUMN IF NOT EXISTS last_run TIMESTAMP;

ALTER TABLE packet_loss_monitors 
ADD COLUMN IF NOT EXISTS next_run TIMESTAMP;

-- Create index for next_run to optimize scheduler queries
CREATE INDEX IF NOT EXISTS idx_packet_loss_monitors_next_run ON packet_loss_monitors(next_run);