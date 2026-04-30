-- Rollback: Remove UL Data message fields and downlink control fields added in 000038

-- Remove indexes and columns from mioty_messages table (conditionally)
-- Migration 047 replaces mioty_messages table, so guard against missing table during rollback
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public'
        AND table_name = 'mioty_messages'
    ) THEN
        -- Remove indexes
        DROP INDEX IF EXISTS idx_mioty_messages_dl_open;
        DROP INDEX IF EXISTS idx_mioty_messages_response_expected;

        -- Remove fields from mioty_messages table
        ALTER TABLE mioty_messages
        DROP COLUMN IF EXISTS user_data,
        DROP COLUMN IF EXISTS format_id,
        DROP COLUMN IF EXISTS mode,
        DROP COLUMN IF EXISTS dl_open,
        DROP COLUMN IF EXISTS response_exp,
        DROP COLUMN IF EXISTS dl_ack;
    END IF;
END $$;

-- Remove fields from endpoints table
ALTER TABLE endpoints
DROP COLUMN IF EXISTS last_dl_open,
DROP COLUMN IF EXISTS last_response_exp,
DROP COLUMN IF EXISTS last_dl_ack,
DROP COLUMN IF EXISTS last_user_data,
DROP COLUMN IF EXISTS last_format_id,
DROP COLUMN IF EXISTS last_mode;