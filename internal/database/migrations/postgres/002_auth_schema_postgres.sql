CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add user_id foreign key to schedules
ALTER TABLE schedules ADD COLUMN user_id INTEGER REFERENCES users(id);

-- Add user_id foreign key to speed_tests
ALTER TABLE speed_tests ADD COLUMN user_id INTEGER REFERENCES users(id);

-- Table to track if registration is allowed
CREATE TABLE IF NOT EXISTS registration_status (
    is_registration_enabled BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT registration_status_singleton CHECK (is_registration_enabled IS NOT NULL),
    CONSTRAINT registration_status_single_row UNIQUE (is_registration_enabled)
);

-- Initialize registration status to enabled
INSERT INTO registration_status (is_registration_enabled) VALUES (true);
