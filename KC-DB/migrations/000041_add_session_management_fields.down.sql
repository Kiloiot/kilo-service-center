-- Remove MIOTY session management fields from basestation_sessions table

ALTER TABLE basestation_sessions 
DROP COLUMN IF EXISTS sn_bs_uuid,
DROP COLUMN IF EXISTS sn_bs_op_id,
DROP COLUMN IF EXISTS sn_sc_op_id;