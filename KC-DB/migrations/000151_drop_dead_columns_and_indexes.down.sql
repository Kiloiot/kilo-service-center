-- Restore the dropped columns with their original types, defaults,
-- constraints and indexes. The up migration only proceeds when every column
-- holds its default, so nothing is lost by recreating them empty.

DROP INDEX IF EXISTS idx_downlink_queue_mioty_prio;

ALTER TABLE downlink_queue
    ADD COLUMN IF NOT EXISTS tx_power_dbm REAL,
    ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS retry_interval INTERVAL DEFAULT '1 minute',
    ADD COLUMN IF NOT EXISTS correlation_id UUID,
    ADD COLUMN IF NOT EXISTS response_to_message_id BIGINT,
    ADD COLUMN IF NOT EXISTS transmitted_by_basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS transmission_status VARCHAR(20) CHECK (transmission_status IN (
        'pending', 'scheduled', 'transmitted', 'confirmed', 'failed', 'expired', 'cancelled'
    ));

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_retry_limit CHECK (retry_count >= 0 AND retry_count <= 10);

ALTER TABLE basestations
    ADD COLUMN IF NOT EXISTS hardware_name VARCHAR(255);

ALTER TABLE endpoints
    ADD COLUMN IF NOT EXISTS nwk_addr INTEGER,
    ADD COLUMN IF NOT EXISTS ep_eui_alt BYTEA CHECK (length(ep_eui_alt) = 8);

ALTER TABLE basestation_sessions
    ADD COLUMN IF NOT EXISTS messages_received BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS messages_sent BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bytes_received BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS bytes_sent BIGINT DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_downlink_queue_correlation ON downlink_queue(correlation_id)
    WHERE correlation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_downlink_queue_response ON downlink_queue(response_to_message_id)
    WHERE response_to_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_downlink_queue_retry ON downlink_queue(retry_count, max_attempts)
    WHERE status = 'failed' AND retry_count < max_attempts;
CREATE INDEX IF NOT EXISTS idx_downlink_queue_mioty_prio ON downlink_queue(transmitted_by_basestation_id, status, prio DESC, created_at)
    WHERE transmitted_by_basestation_id IS NOT NULL
    AND status IN ('pending', 'scheduled');
CREATE INDEX IF NOT EXISTS idx_basestations_hardware_name ON basestations(hardware_name)
    WHERE hardware_name IS NOT NULL;
