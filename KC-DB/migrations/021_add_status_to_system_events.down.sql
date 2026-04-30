-- Remove status column and index added for alert system
DROP INDEX IF EXISTS idx_system_events_status;
ALTER TABLE system_events DROP COLUMN IF EXISTS status;