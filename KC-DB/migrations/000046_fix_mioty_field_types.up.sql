-- Fix MIOTY field data types to match specification exactly
-- Per MIOTY BSSCI v1.0.0: shAddr is 16-bit, opId is 64-bit, repetition is boolean

-- 1. Fix short address (shAddr) to be 16-bit (SMALLINT in PostgreSQL)
-- Currently stored as INTEGER (32-bit) or BIGINT (64-bit)
ALTER TABLE endpoints 
ALTER COLUMN sh_addr TYPE SMALLINT USING sh_addr::SMALLINT;

-- Fix in mioty_messages table (currently BIGINT)
ALTER TABLE mioty_messages 
ALTER COLUMN sh_addr TYPE SMALLINT USING sh_addr::SMALLINT;

-- Fix in endpoint_sessions table  
ALTER TABLE endpoint_sessions
ALTER COLUMN sh_addr TYPE SMALLINT USING sh_addr::SMALLINT;

-- 2. Fix repetition field to be BOOLEAN in all tables
-- Fix in endpoint_sessions (currently SMALLINT)
ALTER TABLE endpoint_sessions
DROP CONSTRAINT IF EXISTS endpoint_sessions_repetition_check,
ALTER COLUMN repetition DROP DEFAULT,
ALTER COLUMN repetition TYPE BOOLEAN USING (repetition > 0),
ALTER COLUMN repetition SET DEFAULT false;

-- Fix in downlink_queue (currently SMALLINT)  
ALTER TABLE downlink_queue
DROP CONSTRAINT IF EXISTS downlink_queue_repetition_check,
ALTER COLUMN repetition DROP DEFAULT,
ALTER COLUMN repetition TYPE BOOLEAN USING (repetition > 0),
ALTER COLUMN repetition SET DEFAULT false;

-- 3. Fix repetition default value in endpoints table
-- Migration 030 incorrectly set DEFAULT true, should be false per MIOTY spec
ALTER TABLE endpoints
ALTER COLUMN repetition SET DEFAULT false;

-- 4. Add missing attach operation fields to endpoints table
-- These fields are provided during attach and should override user values
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS attach_cnt INTEGER,
ADD COLUMN IF NOT EXISTS nonce BYTEA CHECK (length(nonce) = 4 OR nonce IS NULL),
ADD COLUMN IF NOT EXISTS sign BYTEA CHECK (length(sign) = 4 OR sign IS NULL),
ADD COLUMN IF NOT EXISTS last_attach_rx_time BIGINT,     -- Unix UTC ns
ADD COLUMN IF NOT EXISTS last_attach_rx_duration BIGINT; -- Duration in ns

-- 5. Add radio metrics fields if missing
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS last_snr DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS last_rssi DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS last_eq_snr DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS last_profile VARCHAR(16);

-- 6. Add comments to clarify field purposes and sizes
COMMENT ON COLUMN endpoints.sh_addr IS 'Short address (16-bit) per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.repetition IS 'True if End Point uses DL repetition (Boolean per MIOTY BSSCI v1.0.0)';
COMMENT ON COLUMN endpoints.attach_cnt IS 'End Point attachment counter from last attach operation';
COMMENT ON COLUMN endpoints.nonce IS '4-byte End Point nonce from attach';
COMMENT ON COLUMN endpoints.sign IS '4-byte End Point signature from attach';
COMMENT ON COLUMN endpoints.last_attach_rx_time IS 'Unix UTC time of last attach reception (nanoseconds)';
COMMENT ON COLUMN endpoints.last_attach_rx_duration IS 'Duration of last attach reception (nanoseconds)';
COMMENT ON COLUMN endpoints.last_snr IS 'Last reception SNR in dB';
COMMENT ON COLUMN endpoints.last_rssi IS 'Last reception RSSI in dBm';
COMMENT ON COLUMN endpoints.last_eq_snr IS 'Last AWGN equivalent SNR in dB';
COMMENT ON COLUMN endpoints.last_profile IS 'MIOTY profile used for last reception (e.g., eu1)';

COMMENT ON COLUMN mioty_messages.sh_addr IS 'Short address (16-bit) per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoint_sessions.sh_addr IS 'Short address (16-bit) per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoint_sessions.repetition IS 'True if End Point uses DL repetition (Boolean)';
COMMENT ON COLUMN downlink_queue.repetition IS 'True if DL repetition requested (Boolean)';