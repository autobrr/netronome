-- Add MTR support to packet loss monitoring

-- Add columns to track MTR usage and data
ALTER TABLE packet_loss_results ADD COLUMN used_mtr BOOLEAN DEFAULT FALSE;
ALTER TABLE packet_loss_results ADD COLUMN hop_count INTEGER DEFAULT 0;
ALTER TABLE packet_loss_results ADD COLUMN mtr_data TEXT; -- JSON blob containing hop-by-hop statistics
ALTER TABLE packet_loss_results ADD COLUMN privileged_mode BOOLEAN DEFAULT FALSE; -- Track if test ran in privileged (ICMP) mode

-- Create index for MTR-enabled results
CREATE INDEX idx_packet_loss_results_used_mtr ON packet_loss_results(used_mtr);