-- Rollback MIOTY-specific fields from basestations table

-- Drop index
DROP INDEX IF EXISTS idx_basestations_session;

-- Remove columns
ALTER TABLE basestations 
DROP COLUMN IF EXISTS vendor,
DROP COLUMN IF EXISTS model,
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS session_uuid,
DROP COLUMN IF EXISTS session_started_at,
DROP COLUMN IF EXISTS bidi,
DROP COLUMN IF EXISTS uptime,
DROP COLUMN IF EXISTS ul_load,
DROP COLUMN IF EXISTS dl_load,
DROP COLUMN IF EXISTS status_code,
DROP COLUMN IF EXISTS status_message;
