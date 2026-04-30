-- Rollback MIOTY-specific fields from endpoints table

-- Drop index
DROP INDEX IF EXISTS idx_endpoints_status;

-- Remove columns
ALTER TABLE endpoints 
DROP COLUMN IF EXISTS bidi,
DROP COLUMN IF EXISTS pre_attach,
DROP COLUMN IF EXISTS sh_addr,
DROP COLUMN IF EXISTS attach_cnt,
DROP COLUMN IF EXISTS packet_cnt,
DROP COLUMN IF EXISTS dual_chan,
DROP COLUMN IF EXISTS repetition,
DROP COLUMN IF EXISTS wide_carr_off,
DROP COLUMN IF EXISTS long_blk_dist,
DROP COLUMN IF EXISTS ep_status;
