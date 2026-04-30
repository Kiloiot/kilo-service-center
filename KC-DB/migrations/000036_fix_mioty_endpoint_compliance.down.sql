-- Rollback MIOTY Endpoint Compliance changes

-- 1. Revert repetition field back to smallint
-- Must DROP DEFAULT before type conversion to avoid cast errors
ALTER TABLE endpoints
ALTER COLUMN repetition DROP DEFAULT;

ALTER TABLE endpoints
ALTER COLUMN repetition TYPE SMALLINT USING (CASE WHEN repetition THEN 1 ELSE 0 END);

ALTER TABLE endpoints
ALTER COLUMN repetition SET DEFAULT 1;

COMMENT ON COLUMN endpoints.repetition IS 'Number of downlink repetitions (1-15) per MIOTY spec';

-- 2. Remove added Attach operation fields
ALTER TABLE endpoints
DROP COLUMN IF EXISTS nonce,
DROP COLUMN IF EXISTS sign,
DROP COLUMN IF EXISTS last_attach_rx_time,
DROP COLUMN IF EXISTS last_attach_rx_duration;

-- 3. Remove added radio metrics fields
ALTER TABLE endpoints
DROP COLUMN IF EXISTS last_snr,
DROP COLUMN IF EXISTS last_rssi,
DROP COLUMN IF EXISTS last_eq_snr,
DROP COLUMN IF EXISTS last_profile;

-- 4. Drop created indexes
DROP INDEX IF EXISTS idx_endpoints_last_attach_time;
DROP INDEX IF EXISTS idx_endpoints_radio_quality;

-- 5. Remove mioty_messages columns (conditionally - table may not exist after migration 047)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public'
        AND table_name = 'mioty_messages'
    ) THEN
        ALTER TABLE mioty_messages
        DROP COLUMN IF EXISTS rx_duration,
        DROP COLUMN IF EXISTS profile,
        DROP COLUMN IF EXISTS subpackets;
    END IF;
END $$;