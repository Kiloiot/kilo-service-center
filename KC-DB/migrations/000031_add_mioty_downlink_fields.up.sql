-- Add MIOTY-specific downlink queue fields per BSSCI v1.0.0 specification

-- Add MIOTY queue ID and counter dependency fields
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS que_id BIGINT UNIQUE,
ADD COLUMN IF NOT EXISTS cnt_depend BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS packet_cnt BIGINT[],
ADD COLUMN IF NOT EXISTS format INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS response_exp BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS response_prio BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS dl_wind_req BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS exp_only BOOLEAN DEFAULT false;

-- Add result tracking per MIOTY spec
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS result VARCHAR(20) CHECK (result IN ('sent', 'expired', 'invalid', NULL)),
ADD COLUMN IF NOT EXISTS tx_time BIGINT; -- Unix UTC time in nanoseconds

-- Add index for queue ID lookups
CREATE INDEX IF NOT EXISTS idx_downlink_queue_que_id ON downlink_queue(que_id) WHERE que_id IS NOT NULL;

-- Add index for pending messages per endpoint
CREATE INDEX IF NOT EXISTS idx_downlink_queue_pending_ep ON downlink_queue(ep_eui, status) 
    WHERE status = 'pending';

-- Add comments for MIOTY fields
COMMENT ON COLUMN downlink_queue.que_id IS 'MIOTY Queue ID for reference (64 bit)';
COMMENT ON COLUMN downlink_queue.cnt_depend IS 'True if userData is counter dependent';
COMMENT ON COLUMN downlink_queue.packet_cnt IS 'End Point packet counters for which userData is valid';
COMMENT ON COLUMN downlink_queue.format IS 'User data format identifier (8 bit)';
COMMENT ON COLUMN downlink_queue.response_exp IS 'True to request End Point response';
COMMENT ON COLUMN downlink_queue.response_prio IS 'True to request priority End Point response';
COMMENT ON COLUMN downlink_queue.dl_wind_req IS 'True to request further End Point DL window';
COMMENT ON COLUMN downlink_queue.exp_only IS 'True to send downlink only if End Point expects a response';
COMMENT ON COLUMN downlink_queue.result IS 'MIOTY transmission result: sent, expired, or invalid';
COMMENT ON COLUMN downlink_queue.tx_time IS 'Unix UTC time of transmission in nanoseconds';