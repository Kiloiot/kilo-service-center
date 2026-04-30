-- Rollback: Remove MIOTY endpoint fields added in 000037

-- Remove the added columns
ALTER TABLE endpoints
DROP COLUMN IF EXISTS dual_chan,
DROP COLUMN IF EXISTS wide_carr_off,
DROP COLUMN IF EXISTS long_blk_dist,
DROP COLUMN IF EXISTS last_packet_cnt;

-- Remove the index
DROP INDEX IF EXISTS idx_endpoints_packet_cnt;