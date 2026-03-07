-- Add setting column to users table
ALTER TABLE users ADD COLUMN setting TEXT;

-- Create notification_limit table
CREATE TABLE IF NOT EXISTS notification_limits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    notify_type VARCHAR(50) NOT NULL,
    last_notify_time INTEGER NOT NULL,
    count INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_user_notify ON notification_limits(user_id, notify_type);
