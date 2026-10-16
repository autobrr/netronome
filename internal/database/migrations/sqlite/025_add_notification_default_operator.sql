ALTER TABLE notification_events ADD COLUMN default_operator TEXT;

UPDATE notification_events SET default_operator = 'lt'
WHERE category = 'speedtest' AND event_type IN ('download_low', 'upload_low');

UPDATE notification_events SET default_operator = 'gt'
WHERE supports_threshold AND default_operator IS NULL;

UPDATE notification_rules SET threshold_operator = 'lt'
WHERE threshold_operator = 'gt' AND event_id IN (
    SELECT id FROM notification_events
    WHERE category = 'speedtest' AND event_type IN ('download_low', 'upload_low')
);

UPDATE notification_rules SET threshold_operator = (
    SELECT default_operator FROM notification_events WHERE notification_events.id = notification_rules.event_id
)
WHERE threshold_operator IS NULL AND event_id IN (
    SELECT id FROM notification_events WHERE supports_threshold
);
