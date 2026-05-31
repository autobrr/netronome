CREATE INDEX IF NOT EXISTS idx_packet_loss_results_monitor_created_at
ON packet_loss_results(monitor_id, created_at DESC, id DESC);
