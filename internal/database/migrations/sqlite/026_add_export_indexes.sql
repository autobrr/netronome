CREATE INDEX IF NOT EXISTS idx_speed_tests_created_at_julianday ON speed_tests(julianday(created_at));
CREATE INDEX IF NOT EXISTS idx_packet_loss_results_created_at_julianday ON packet_loss_results(julianday(created_at));
CREATE INDEX IF NOT EXISTS idx_dns_results_created_at_julianday ON dns_results(julianday(created_at));
