-- Remove MIOTY attach propagate fields
ALTER TABLE endpoints 
DROP COLUMN IF EXISTS dual_chan,
DROP COLUMN IF EXISTS repetition,
DROP COLUMN IF EXISTS wide_carr_off,
DROP COLUMN IF EXISTS long_blk_dist,
DROP COLUMN IF EXISTS last_packet_cnt;