-- Drop notification_limit table
DROP TABLE IF EXISTS notification_limits;

-- Remove setting column from users table (SQLite doesn't support DROP COLUMN directly)
-- Manual intervention required for SQLite databases
