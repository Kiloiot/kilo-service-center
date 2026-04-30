-- DEFER-007 rollback: Remove 'reserved' and 'queued' from downlink_queue status CHECK constraint
-- Keeps all other statuses that existed prior to DEFER-007 to prevent rollback failures
-- when rows exist with revoked/acked/delivered status

-- Drop the expanded constraint
ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_status_check;

-- Restore constraint excluding only reserved/queued (DEFER-007 additions)
-- NOTE: Rollback will fail if rows exist with status='reserved' or status='queued'
-- In production, first transition any reserved/queued to pending before downgrading
ALTER TABLE downlink_queue ADD CONSTRAINT downlink_queue_status_check CHECK (status IN (
    'pending',     -- Queued awaiting scheduler processing
    'scheduled',   -- Scheduler selected for transmission
    'transmitted', -- BS transmitted downlink
    'confirmed',   -- Endpoint confirmed receipt
    'failed',      -- Transmission failed
    'expired',     -- Validity period elapsed
    'cancelled',   -- Cancelled by user/system
    'revoked',     -- Revoked via dlDataRev
    'acked',       -- Endpoint acknowledgment received
    'delivered'    -- Endpoint acknowledged receipt
));
