-- Add missing MIOTY session management fields to basestation_sessions table
-- These fields are required per MIOTY BSSCI v1.0.0 Section 3.3

ALTER TABLE basestation_sessions 
ADD COLUMN IF NOT EXISTS sn_bs_uuid BYTEA,
ADD COLUMN IF NOT EXISTS sn_bs_op_id BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS sn_sc_op_id BIGINT DEFAULT 0;

-- Add comments for clarity
COMMENT ON COLUMN basestation_sessions.sn_bs_uuid IS 'Base Station session UUID (16 bytes) - from base station per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN basestation_sessions.sn_bs_op_id IS 'Last known Base Station operation ID counter per MIOTY BSSCI v1.0.0';  
COMMENT ON COLUMN basestation_sessions.sn_sc_op_id IS 'Service Center operation ID counter (negative, decrementing) per MIOTY BSSCI v1.0.0';