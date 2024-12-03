-- Add test_type column to speed_tests table
ALTER TABLE speed_tests ADD COLUMN test_type VARCHAR(50) NOT NULL DEFAULT 'speedtest';

-- Add index for faster queries when filtering by test type
CREATE INDEX idx_speed_tests_test_type ON speed_tests(test_type);
