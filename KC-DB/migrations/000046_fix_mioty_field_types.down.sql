-- Revert MIOTY field type fixes

-- 1. Revert short address back to original types
ALTER TABLE endpoints
ALTER COLUMN sh_addr TYPE INTEGER USING sh_addr::INTEGER;

-- Conditionally revert mioty_messages sh_addr if table exists
-- (table may have been dropped by migration 018 or replaced by migration 047)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public'
        AND table_name = 'mioty_messages'
    ) THEN
        ALTER TABLE mioty_messages
        ALTER COLUMN sh_addr TYPE BIGINT USING sh_addr::BIGINT;
    END IF;
END $$;

ALTER TABLE endpoint_sessions
ALTER COLUMN sh_addr TYPE INTEGER USING sh_addr::INTEGER;

-- 2. Revert repetition fields back to SMALLINT
-- Must DROP DEFAULT before type conversion to avoid cast errors
ALTER TABLE endpoint_sessions
ALTER COLUMN repetition DROP DEFAULT;

ALTER TABLE endpoint_sessions
ALTER COLUMN repetition TYPE SMALLINT USING (CASE WHEN repetition THEN 1 ELSE 0 END);

-- Restore original default from migration 003 (DEFAULT 1)
ALTER TABLE endpoint_sessions
ALTER COLUMN repetition SET DEFAULT 1;

-- Restore CHECK constraint removed by UP script (line 20)
ALTER TABLE endpoint_sessions
ADD CONSTRAINT endpoint_sessions_repetition_check CHECK (repetition >= 1 AND repetition <= 15);

ALTER TABLE downlink_queue
ALTER COLUMN repetition DROP DEFAULT;

ALTER TABLE downlink_queue
ALTER COLUMN repetition TYPE SMALLINT USING (CASE WHEN repetition THEN 1 ELSE 0 END);

-- Restore original default from migration 009 (DEFAULT 1)
ALTER TABLE downlink_queue
ALTER COLUMN repetition SET DEFAULT 1;

-- Restore CHECK constraint removed by UP script (line 27)
ALTER TABLE downlink_queue
ADD CONSTRAINT downlink_queue_repetition_check CHECK (repetition >= 1 AND repetition <= 15);

-- 3. Revert endpoints.repetition default
-- Migration 046 UP changed DEFAULT from true→false (line 35)
-- Migration 036 already converted repetition from SMALLINT→BOOLEAN
-- So we restore to BOOLEAN DEFAULT true (not integer 1)
ALTER TABLE endpoints
ALTER COLUMN repetition SET DEFAULT true;

-- 4. Remove added attach operation fields
ALTER TABLE endpoints
DROP COLUMN IF EXISTS attach_cnt,
DROP COLUMN IF EXISTS nonce,
DROP COLUMN IF EXISTS sign,
DROP COLUMN IF EXISTS last_attach_rx_time,
DROP COLUMN IF EXISTS last_attach_rx_duration;

-- 5. Remove radio metrics fields
ALTER TABLE endpoints
DROP COLUMN IF EXISTS last_snr,
DROP COLUMN IF EXISTS last_rssi,
DROP COLUMN IF EXISTS last_eq_snr,
DROP COLUMN IF EXISTS last_profile;

-- 6. Remove comments
COMMENT ON COLUMN endpoints.sh_addr IS NULL;
COMMENT ON COLUMN endpoints.repetition IS NULL;

-- Conditionally remove comment from mioty_messages if table exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public'
        AND table_name = 'mioty_messages'
    ) THEN
        COMMENT ON COLUMN mioty_messages.sh_addr IS NULL;
    END IF;
END $$;

COMMENT ON COLUMN endpoint_sessions.sh_addr IS NULL;
COMMENT ON COLUMN endpoint_sessions.repetition IS NULL;
COMMENT ON COLUMN downlink_queue.repetition IS NULL;