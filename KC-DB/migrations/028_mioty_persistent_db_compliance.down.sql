-- Rollback MIOTY Persistent Database Compliance Migration

-- Revert endpoint column standardization (idempotent)
DO $$
BEGIN
    -- Only revert if column currently named ep_eui
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'ep_eui') THEN
        -- Check we're not in a state where other variants exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_name = 'endpoints'
                       AND column_name IN ('eui', 'epEui', 'epeui')) THEN
            ALTER TABLE endpoints RENAME COLUMN ep_eui TO eui;
        END IF;
    END IF;
END $$;

-- Drop trigger and function for packet counter updates
DROP TRIGGER IF EXISTS trg_update_packet_counter ON messages;
DROP FUNCTION IF EXISTS update_endpoint_last_packet_cnt();

-- Drop operation tracking table
DROP TABLE IF EXISTS bssci_operation_tracking;

-- Revert base station sessions changes - drop only columns added by migration 028
ALTER TABLE basestation_sessions
DROP COLUMN IF EXISTS sn_bs_uuid,
DROP COLUMN IF EXISTS sn_sc_uuid,
DROP COLUMN IF EXISTS sn_bs_op_id,
DROP COLUMN IF EXISTS sn_sc_op_id,
DROP COLUMN IF EXISTS can_resume,
DROP COLUMN IF EXISTS updated_at;

-- Drop indexes added by migration 028
-- NOTE: idx_basestation_sessions_unique_active existed before this migration (from 007)
-- so we should NOT drop it here - only drop indexes that migration 028 actually added
DROP INDEX IF EXISTS idx_basestation_sessions_bs_uuid;
DROP INDEX IF EXISTS idx_basestation_sessions_sc_uuid;

-- Revert downlink_queue changes
ALTER TABLE downlink_queue
DROP COLUMN IF EXISTS que_id,
DROP COLUMN IF EXISTS cnt_depend,
DROP COLUMN IF EXISTS packet_cnt_array,
DROP COLUMN IF EXISTS result,
DROP COLUMN IF EXISTS tx_time,
DROP COLUMN IF EXISTS tx_bs_eui;

DROP INDEX IF EXISTS idx_downlink_queue_status_priority;

-- Revert endpoint_sessions changes
ALTER TABLE endpoint_sessions
RENAME COLUMN last_packet_cnt TO packet_cnt;

ALTER TABLE endpoint_sessions
ALTER COLUMN packet_cnt TYPE INTEGER;

-- Revert endpoints changes  
ALTER TABLE endpoints
DROP COLUMN IF EXISTS last_packet_cnt,
DROP COLUMN IF EXISTS dual_chan,
DROP COLUMN IF EXISTS repetition,
DROP COLUMN IF EXISTS wide_carr_off,
DROP COLUMN IF EXISTS long_blk_dist;

DROP INDEX IF EXISTS idx_endpoints_last_packet_cnt;

-- Remove schema comment
COMMENT ON SCHEMA public IS NULL;