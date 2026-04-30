-- DEFER-007: Add 'reserved' and 'queued' to downlink_queue status CHECK constraint
-- 'reserved' is a transient state during auto-dispatch transaction (BSSCI §5.10.2)
-- 'queued' indicates message sent to BS via dlDataQue, awaiting actual transmission

-- Drop existing constraint
ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_status_check;

-- Add updated constraint with new status values
-- Full lifecycle: pending → scheduled → reserved → queued → transmitted/failed
ALTER TABLE downlink_queue ADD CONSTRAINT downlink_queue_status_check CHECK (status IN (
    'pending',     -- Queued awaiting scheduler processing
    'scheduled',   -- Scheduler selected for transmission
    'reserved',    -- DEFER-007: Reserved for auto-dispatch (transient)
    'queued',      -- Sent to BS via dlDataQue, awaiting transmission
    'transmitted', -- BS transmitted downlink (dlDataQueCmp success)
    'confirmed',   -- Endpoint confirmed receipt
    'failed',      -- Transmission failed (dlDataQueCmp error)
    'expired',     -- Validity period elapsed before transmission
    'cancelled',   -- Cancelled by user/system
    'revoked',     -- Revoked via dlDataRev
    'acked',       -- Endpoint acknowledgment received (dlDataRes)
    'delivered'    -- Endpoint acknowledged receipt (if ack requested)
));

-- Add comment documenting the lifecycle
COMMENT ON COLUMN downlink_queue.status IS 'Lifecycle: pending → reserved → queued → transmitted/failed (DEFER-007 §5.10.2)';
