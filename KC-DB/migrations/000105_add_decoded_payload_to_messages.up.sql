-- Migration 000105: Add decoded payload columns to messages table
-- Part of Blueprint feature (MIOTY Application Layer payload decoding)
-- NOTE: Alters canonical 'messages' table, NOT deprecated 'mioty_messages'

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS decoded_payload JSONB,
    ADD COLUMN IF NOT EXISTS blueprint_type_eui BYTEA CHECK (blueprint_type_eui IS NULL OR length(blueprint_type_eui) = 8),
    ADD COLUMN IF NOT EXISTS blueprint_version_id UUID REFERENCES blueprints(id),
    ADD COLUMN IF NOT EXISTS decode_status VARCHAR(32) DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS decode_error_code VARCHAR(64);

-- GIN index for efficient JSONB queries on decoded payload
CREATE INDEX idx_messages_decoded_gin ON messages USING GIN (decoded_payload)
    WHERE decoded_payload IS NOT NULL;

-- Partial index for decode status queries (non-success cases for debugging/monitoring)
CREATE INDEX idx_messages_decode_status ON messages(decode_status)
    WHERE decode_status NOT IN ('success', 'skipped');

-- Index for blueprint lookups
CREATE INDEX idx_messages_blueprint_type_eui ON messages(blueprint_type_eui)
    WHERE blueprint_type_eui IS NOT NULL;

-- Comments for documentation
COMMENT ON COLUMN messages.decoded_payload IS 'Decoded user_data as JSON per blueprint spec';
COMMENT ON COLUMN messages.blueprint_type_eui IS '8-byte MIOTY Type EUI used for decoding';
COMMENT ON COLUMN messages.blueprint_version_id IS 'Reference to the blueprint version used for decoding';
COMMENT ON COLUMN messages.decode_status IS 'Decoding status: pending, success, failed, skipped (use centralized constants)';
COMMENT ON COLUMN messages.decode_error_code IS 'Error token if decode_status is failed';
