-- Add application settings table for single-user persisted UI/server settings
CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER IF NOT EXISTS update_app_settings_timestamp
AFTER UPDATE ON app_settings
BEGIN
    UPDATE app_settings SET updated_at = CURRENT_TIMESTAMP WHERE key = NEW.key;
END;
