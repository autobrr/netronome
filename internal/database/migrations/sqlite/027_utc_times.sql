DROP INDEX IF EXISTS idx_speed_tests_created_at_julianday;
DROP INDEX IF EXISTS idx_packet_loss_results_created_at_julianday;
DROP INDEX IF EXISTS idx_dns_results_created_at_julianday;

UPDATE speed_tests SET created_at = COALESCE(strftime('%Y-%m-%d %H:%M:%f', CASE
    WHEN created_at LIKE '% +____ %' OR created_at LIKE '% -____ %' THEN
      replace(substr(created_at, 1, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') - 1), ' ', 'T')
      || substr(created_at, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') + 1, 3)
      || ':'
      || substr(created_at, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') + 4, 2)
    ELSE created_at END) || '+00:00', created_at);

UPDATE packet_loss_results SET created_at = COALESCE(strftime('%Y-%m-%d %H:%M:%f', CASE
    WHEN created_at LIKE '% +____ %' OR created_at LIKE '% -____ %' THEN
      replace(substr(created_at, 1, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') - 1), ' ', 'T')
      || substr(created_at, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') + 1, 3)
      || ':'
      || substr(created_at, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') + 4, 2)
    ELSE created_at END) || '+00:00', created_at);

UPDATE dns_results SET created_at = COALESCE(strftime('%Y-%m-%d %H:%M:%f', CASE
    WHEN created_at LIKE '% +____ %' OR created_at LIKE '% -____ %' THEN
      replace(substr(created_at, 1, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') - 1), ' ', 'T')
      || substr(created_at, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') + 1, 3)
      || ':'
      || substr(created_at, instr(created_at, ' ') + instr(substr(created_at, instr(created_at, ' ') + 1), ' ') + 4, 2)
    ELSE created_at END) || '+00:00', created_at);

CREATE INDEX IF NOT EXISTS idx_speed_tests_created_at ON speed_tests(created_at);
