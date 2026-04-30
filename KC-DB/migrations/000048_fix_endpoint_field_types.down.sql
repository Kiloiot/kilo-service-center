-- Migration 048 Down: Revert endpoint field type changes

-- Remove constraints
ALTER TABLE endpoints 
DROP CONSTRAINT IF EXISTS valid_format_id,
DROP CONSTRAINT IF EXISTS valid_nonce_length,
DROP CONSTRAINT IF EXISTS valid_sign_length;

-- Revert format field type back to INTEGER
ALTER TABLE endpoints 
ALTER COLUMN last_format_id TYPE INTEGER USING last_format_id::INTEGER;

-- Make packet counter fields nullable again (reverting to previous state)
ALTER TABLE endpoints 
ALTER COLUMN packet_cnt DROP NOT NULL,
ALTER COLUMN packet_cnt DROP DEFAULT,
ALTER COLUMN last_packet_cnt DROP NOT NULL,
ALTER COLUMN last_packet_cnt DROP DEFAULT;

-- Remove updated comments (reverting to previous state)
COMMENT ON COLUMN endpoints.sh_addr IS NULL;
COMMENT ON COLUMN endpoints.packet_cnt IS NULL;
COMMENT ON COLUMN endpoints.last_packet_cnt IS NULL;
COMMENT ON COLUMN endpoints.last_rx_time IS NULL;
COMMENT ON COLUMN endpoints.last_rx_duration IS NULL;
COMMENT ON COLUMN endpoints.last_attach_rx_time IS NULL;
COMMENT ON COLUMN endpoints.last_attach_rx_duration IS NULL;
COMMENT ON COLUMN endpoints.last_format_id IS NULL;