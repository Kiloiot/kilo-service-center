-- Remove MIOTY prio field and restore old index

-- Drop the MIOTY prio index
DROP INDEX IF EXISTS idx_downlink_queue_mioty_prio;

-- Remove the MIOTY prio column
ALTER TABLE downlink_queue
DROP COLUMN IF EXISTS prio;

-- Restore the old index using priority field
CREATE INDEX idx_downlink_queue_basestation ON downlink_queue(transmitted_by_basestation_id, status, priority DESC, created_at) 
    WHERE transmitted_by_basestation_id IS NOT NULL 
    AND status IN ('pending', 'scheduled');