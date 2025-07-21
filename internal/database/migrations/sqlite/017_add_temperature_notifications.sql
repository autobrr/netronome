-- Add temperature high notification event
INSERT INTO notification_events (category, event_type, name, description, supports_threshold, threshold_unit) VALUES
('agent', 'temperature_high', 'High Temperature', 'System temperature exceeds threshold', 1, '°C');