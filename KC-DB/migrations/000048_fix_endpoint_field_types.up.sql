-- Migration 048: Fix endpoint field types to match MIOTY BSSCI v1.0.0 specification

-- Fix format field type (should be 8-bit, not 32-bit)
ALTER TABLE endpoints 
ALTER COLUMN last_format_id TYPE SMALLINT USING last_format_id::SMALLINT;

-- Add constraint to ensure format is within 8-bit range
ALTER TABLE endpoints 
ADD CONSTRAINT valid_format_id CHECK (last_format_id IS NULL OR (last_format_id >= 0 AND last_format_id <= 255));

-- The sh_addr field was already fixed to SMALLINT in migration 046, verify it's correct
-- Just add a comment to document the correct type
COMMENT ON COLUMN endpoints.sh_addr IS 'End Point short address (16-bit unsigned) per MIOTY BSSCI v1.0.0';

-- Ensure packet_cnt is using correct type (should be 32-bit unsigned, using INTEGER)
COMMENT ON COLUMN endpoints.packet_cnt IS 'Current End Point packet counter (32-bit unsigned) per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_packet_cnt IS 'Last known End Point packet counter (32-bit unsigned) for deduplication per MIOTY BSSCI v1.0.0';

-- Fix any NULL packet counters to 0 (they should never be NULL)
UPDATE endpoints SET packet_cnt = 0 WHERE packet_cnt IS NULL;
UPDATE endpoints SET last_packet_cnt = 0 WHERE last_packet_cnt IS NULL;

-- Make packet counter fields NOT NULL with default
ALTER TABLE endpoints 
ALTER COLUMN packet_cnt SET NOT NULL,
ALTER COLUMN packet_cnt SET DEFAULT 0,
ALTER COLUMN last_packet_cnt SET NOT NULL,
ALTER COLUMN last_packet_cnt SET DEFAULT 0;

-- Ensure time fields are properly documented as nanoseconds
COMMENT ON COLUMN endpoints.last_rx_time IS 'Unix UTC time in nanoseconds (64-bit) of last UL Data reception per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_rx_duration IS 'Duration in nanoseconds (64-bit) of last UL Data reception per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_attach_rx_time IS 'Unix UTC time in nanoseconds (64-bit) of last attach reception per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_attach_rx_duration IS 'Duration in nanoseconds (64-bit) of last attach reception per MIOTY BSSCI v1.0.0';

-- Ensure nonce and sign are exactly 4 bytes as per spec
ALTER TABLE endpoints 
ADD CONSTRAINT valid_nonce_length CHECK (nonce IS NULL OR length(nonce) = 4),
ADD CONSTRAINT valid_sign_length CHECK (sign IS NULL OR length(sign) = 4);

-- Update format field comment
COMMENT ON COLUMN endpoints.last_format_id IS 'Last received user data format identifier (8-bit, 0-255) per MIOTY BSSCI v1.0.0';