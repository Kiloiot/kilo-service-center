-- Drop columns no repository reads or writes, with their indexes and
-- constraints, and rebuild the one hot-path index that led on a column that
-- was never written.
--
-- downlink_queue (009): response_to_message_id, retry_interval,
--   transmission_status, tx_power_dbm, transmitted_by_basestation_id,
--   correlation_id, retry_count. Dispatch tracks delivery through attempts,
--   status, result and que_id; the dropped columns were never assigned.
-- basestations (000034): hardware_name. The BSSCI connect handler keeps the
--   hardware-reported name on the session; the column was never written.
-- endpoints (015): nwk_addr, ep_eui_alt. MIOTY addressing uses sh_addr and
--   ep_eui; neither column is populated.
-- basestation_sessions (007/015): messages_received, messages_sent,
--   bytes_received, bytes_sent. The session repository never updates them;
--   traffic counters are derived from messages.
--
-- Each column is guarded by its own assertion for non-default data rather
-- than a blanket NOT NULL check, because retry_interval and retry_count carry
-- defaults and the session counters default to zero. Any populated value
-- aborts the migration for operator review.
--
-- idx_downlink_queue_mioty_prio (000032) led on
-- transmitted_by_basestation_id, so every pending row fell outside its
-- partial predicate; it is rebuilt on (status, prio DESC, created_at) over
-- the pending/scheduled rows the dispatcher scans.

DO $$
DECLARE
    populated bigint;
BEGIN
    SELECT count(*) INTO populated FROM downlink_queue WHERE response_to_message_id IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.response_to_message_id is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue
    WHERE retry_interval IS NOT NULL AND retry_interval <> INTERVAL '1 minute';
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.retry_interval carries a non-default value on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE transmission_status IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.transmission_status is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE tx_power_dbm IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.tx_power_dbm is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE transmitted_by_basestation_id IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.transmitted_by_basestation_id is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE correlation_id IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.correlation_id is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE retry_count IS NOT NULL AND retry_count <> 0;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.retry_count carries a non-default value on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM basestations WHERE hardware_name IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'basestations.hardware_name is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM endpoints WHERE nwk_addr IS NOT NULL OR ep_eui_alt IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'endpoints.nwk_addr or ep_eui_alt is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM basestation_sessions
    WHERE COALESCE(messages_received, 0) <> 0 OR COALESCE(messages_sent, 0) <> 0
       OR COALESCE(bytes_received, 0) <> 0 OR COALESCE(bytes_sent, 0) <> 0;
    IF populated > 0 THEN RAISE EXCEPTION 'basestation_sessions traffic counters are populated on % row(s)', populated; END IF;
END $$;

DROP INDEX IF EXISTS idx_downlink_queue_correlation;
DROP INDEX IF EXISTS idx_downlink_queue_response;
DROP INDEX IF EXISTS idx_downlink_queue_retry;
DROP INDEX IF EXISTS idx_downlink_queue_mioty_prio;
DROP INDEX IF EXISTS idx_basestations_hardware_name;

ALTER TABLE downlink_queue
    DROP CONSTRAINT IF EXISTS downlink_queue_retry_limit;

ALTER TABLE downlink_queue
    DROP COLUMN IF EXISTS response_to_message_id,
    DROP COLUMN IF EXISTS retry_interval,
    DROP COLUMN IF EXISTS transmission_status,
    DROP COLUMN IF EXISTS tx_power_dbm,
    DROP COLUMN IF EXISTS transmitted_by_basestation_id,
    DROP COLUMN IF EXISTS correlation_id,
    DROP COLUMN IF EXISTS retry_count;

ALTER TABLE basestations
    DROP COLUMN IF EXISTS hardware_name;

ALTER TABLE endpoints
    DROP COLUMN IF EXISTS nwk_addr,
    DROP COLUMN IF EXISTS ep_eui_alt;

ALTER TABLE basestation_sessions
    DROP COLUMN IF EXISTS messages_received,
    DROP COLUMN IF EXISTS messages_sent,
    DROP COLUMN IF EXISTS bytes_received,
    DROP COLUMN IF EXISTS bytes_sent;

CREATE INDEX idx_downlink_queue_mioty_prio ON downlink_queue(status, prio DESC, created_at)
    WHERE status IN ('pending', 'scheduled');
