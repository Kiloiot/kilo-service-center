-- Rollback: Restore original 0.0-1.0 range constraint as migration 000054 left it

-- Drop the non-negative constraint
ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS priority_non_negative;

-- Restore the original range constraint
ALTER TABLE downlink_queue ADD CONSTRAINT priority_range
  CHECK (priority >= 0.0 AND priority <= 1.0);

-- Restore original comment
COMMENT ON COLUMN downlink_queue.priority IS
  'MIOTY BSSCI §3.12.1: Priority (0.0-1.0), higher values prioritized';