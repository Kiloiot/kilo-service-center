-- Migration: Allow unbounded priority values per MIOTY BSSCI §3.12.1
-- The spec states priority is "single precision floating point" with "higher values prioritized"
-- and no upper limit mentioned. We keep the lower bound at 0 as negative priorities aren't justified.

-- Remove the existing 0.0-1.0 range constraint
ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS priority_range;

-- Add new constraint allowing any non-negative float value
ALTER TABLE downlink_queue ADD CONSTRAINT priority_non_negative
  CHECK (priority >= 0.0);

-- Update comment to reflect the new constraint
COMMENT ON COLUMN downlink_queue.priority IS
  'MIOTY BSSCI §3.12.1: Priority, single precision float (>= 0.0), higher values prioritized';