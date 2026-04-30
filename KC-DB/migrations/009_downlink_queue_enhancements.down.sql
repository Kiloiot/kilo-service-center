-- Rollback downlink_queue enhancements

-- Drop indexes
DROP INDEX IF EXISTS idx_downlink_queue_scheduling;
DROP INDEX IF EXISTS idx_downlink_queue_correlation;
DROP INDEX IF EXISTS idx_downlink_queue_response;
DROP INDEX IF EXISTS idx_downlink_queue_retry;

-- Drop constraints
ALTER TABLE downlink_queue
DROP CONSTRAINT IF EXISTS downlink_queue_scheduling_window,
DROP CONSTRAINT IF EXISTS downlink_queue_status_check;

-- Remove columns
ALTER TABLE downlink_queue
DROP COLUMN IF EXISTS dl_open,
DROP COLUMN IF EXISTS res_exp,
DROP COLUMN IF EXISTS dl_ack,
DROP COLUMN IF EXISTS repetition,
DROP COLUMN IF EXISTS tx_power_dbm,
DROP COLUMN IF EXISTS frequency,
DROP COLUMN IF EXISTS channel,
DROP COLUMN IF EXISTS earliest_at,
DROP COLUMN IF EXISTS latest_at,
DROP COLUMN IF EXISTS max_retries,
DROP COLUMN IF EXISTS retry_count,
DROP COLUMN IF EXISTS retry_interval,
DROP COLUMN IF EXISTS correlation_id,
DROP COLUMN IF EXISTS response_to_message_id,
DROP COLUMN IF EXISTS transmitted_by_gateway_id,
DROP COLUMN IF EXISTS transmission_status;

-- Restore original status constraint
ALTER TABLE downlink_queue
ADD CONSTRAINT downlink_queue_status_check CHECK (status IN ('pending', 'sent', 'failed', 'expired'));
