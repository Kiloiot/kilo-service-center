-- Rollback: Remove payload array support

-- Drop the user_data column
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS user_data;