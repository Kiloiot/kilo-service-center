-- Enhance downlink_queue table with MIOTY-specific fields

-- Add MIOTY downlink control flags
ALTER TABLE downlink_queue
ADD COLUMN dl_open BOOLEAN DEFAULT false,
ADD COLUMN res_exp BOOLEAN DEFAULT false,
ADD COLUMN dl_ack BOOLEAN DEFAULT false;

-- Add transmission parameters
ALTER TABLE downlink_queue
ADD COLUMN repetition SMALLINT DEFAULT 1 CHECK (repetition >= 1 AND repetition <= 15),
ADD COLUMN tx_power_dbm REAL,
ADD COLUMN frequency BIGINT,
ADD COLUMN channel INTEGER;

-- Add scheduling window (latest_at only, earliest_at already exists)
ALTER TABLE downlink_queue
ADD COLUMN latest_at TIMESTAMPTZ;

-- Add retry configuration (max_retries already exists as max_attempts, add others)
ALTER TABLE downlink_queue
ADD COLUMN retry_count INTEGER DEFAULT 0,
ADD COLUMN retry_interval INTERVAL DEFAULT '1 minute';

-- Add correlation tracking
ALTER TABLE downlink_queue
ADD COLUMN correlation_id UUID,
ADD COLUMN response_to_message_id BIGINT; -- Reference to messages.id (no FK constraint due to partitioning)

-- Add transmission result
ALTER TABLE downlink_queue
ADD COLUMN transmitted_by_basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
ADD COLUMN transmission_status VARCHAR(20) CHECK (transmission_status IN (
    'pending', 'scheduled', 'transmitted', 'confirmed', 'failed', 'expired', 'cancelled'
));

-- Add constraint for scheduling window
ALTER TABLE downlink_queue
ADD CONSTRAINT downlink_queue_scheduling_window CHECK (
    (earliest_at IS NULL AND latest_at IS NULL) OR
    (earliest_at IS NOT NULL AND latest_at IS NOT NULL AND earliest_at < latest_at)
);

-- Update status default and check constraint
ALTER TABLE downlink_queue
ALTER COLUMN status SET DEFAULT 'pending',
DROP CONSTRAINT IF EXISTS downlink_queue_status_check;

ALTER TABLE downlink_queue
ADD CONSTRAINT downlink_queue_status_check CHECK (status IN (
    'pending', 'scheduled', 'transmitted', 'confirmed', 'failed', 'expired', 'cancelled'
));

-- Add indexes for new fields
CREATE INDEX idx_downlink_queue_scheduling ON downlink_queue(tenant_id, earliest_at, status) 
    WHERE status IN ('pending', 'scheduled');
CREATE INDEX idx_downlink_queue_correlation ON downlink_queue(correlation_id) 
    WHERE correlation_id IS NOT NULL;
CREATE INDEX idx_downlink_queue_response ON downlink_queue(response_to_message_id) 
    WHERE response_to_message_id IS NOT NULL;
CREATE INDEX idx_downlink_queue_retry ON downlink_queue(retry_count, max_attempts) 
    WHERE status = 'failed' AND retry_count < max_attempts;

-- Add comments
COMMENT ON COLUMN downlink_queue.dl_open IS 'MIOTY DL_OPEN flag - device ready to receive';
COMMENT ON COLUMN downlink_queue.res_exp IS 'MIOTY RES_EXP flag - response expected';
COMMENT ON COLUMN downlink_queue.dl_ack IS 'MIOTY DL_ACK flag - acknowledgment requested';
COMMENT ON COLUMN downlink_queue.repetition IS 'Number of transmission repetitions (1-15)';
COMMENT ON COLUMN downlink_queue.tx_power_dbm IS 'Transmission power in dBm';
COMMENT ON COLUMN downlink_queue.earliest_at IS 'Earliest time to transmit message';
COMMENT ON COLUMN downlink_queue.latest_at IS 'Latest time to transmit message';
COMMENT ON COLUMN downlink_queue.correlation_id IS 'Correlation ID for request/response tracking';
COMMENT ON COLUMN downlink_queue.response_to_message_id IS 'Reference to uplink message this responds to';
COMMENT ON COLUMN downlink_queue.transmitted_by_basestation_id IS 'Basestation that transmitted the message';
