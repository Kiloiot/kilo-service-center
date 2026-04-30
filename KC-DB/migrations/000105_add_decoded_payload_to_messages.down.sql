-- Migration 000105 rollback: Remove decoded payload columns from messages table

DROP INDEX IF EXISTS idx_messages_blueprint_type_eui;
DROP INDEX IF EXISTS idx_messages_decode_status;
DROP INDEX IF EXISTS idx_messages_decoded_gin;

ALTER TABLE messages
    DROP COLUMN IF EXISTS decode_error_code,
    DROP COLUMN IF EXISTS decode_status,
    DROP COLUMN IF EXISTS blueprint_version_id,
    DROP COLUMN IF EXISTS blueprint_type_eui,
    DROP COLUMN IF EXISTS decoded_payload;
