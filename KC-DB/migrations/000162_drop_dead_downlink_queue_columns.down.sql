-- Restore the dropped columns with their original types, defaults, comments
-- and index. The up migration only proceeds when every column holds its
-- default, so nothing is lost by recreating them empty.

ALTER TABLE downlink_queue
    ADD COLUMN IF NOT EXISTS prio REAL DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS valid_until TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS packet_cnt_array BIGINT[];

COMMENT ON COLUMN downlink_queue.prio IS 'MIOTY priority field - single precision floating point, higher values are prioritized (default 0.0)';
COMMENT ON COLUMN downlink_queue.packet_cnt_array IS 'End Point packet counters for which userData is valid';

CREATE INDEX IF NOT EXISTS idx_downlink_queue_mioty_prio ON downlink_queue(status, prio DESC, created_at)
    WHERE status IN ('pending', 'scheduled');
