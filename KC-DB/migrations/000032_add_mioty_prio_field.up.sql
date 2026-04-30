-- Add MIOTY-compliant prio field per BSSCI v1.0.0 specification
-- The specification states: "Priority, higher values are prioritized, single precision floating point"

-- Add the MIOTY prio field if it doesn't exist
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS prio REAL DEFAULT 0.0;

-- Comment for documentation
COMMENT ON COLUMN downlink_queue.prio IS 'MIOTY priority field - single precision floating point, higher values are prioritized (default 0.0)';

-- Update index to use MIOTY prio field instead of the old priority field
DROP INDEX IF EXISTS idx_downlink_queue_basestation;

CREATE INDEX idx_downlink_queue_mioty_prio ON downlink_queue(transmitted_by_basestation_id, status, prio DESC, created_at) 
    WHERE transmitted_by_basestation_id IS NOT NULL 
    AND status IN ('pending', 'scheduled');