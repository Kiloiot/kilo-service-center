-- Migration 087 Down: Revert unsigned integer storage changes
-- WARNING: This may lose data if values exceed original signed type range

BEGIN;

-- Drop CHECK constraints from endpoints table
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_sh_addr_unsigned;
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_attach_cnt_unsigned;
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_packet_cnt_unsigned;
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_last_packet_cnt_unsigned;

-- Drop CHECK constraints from endpoint_sessions table
ALTER TABLE endpoint_sessions DROP CONSTRAINT IF EXISTS chk_endpoint_sessions_sh_addr_unsigned;
ALTER TABLE endpoint_sessions DROP CONSTRAINT IF EXISTS chk_endpoint_sessions_attach_cnt_unsigned;
ALTER TABLE endpoint_sessions DROP CONSTRAINT IF EXISTS chk_endpoint_sessions_last_packet_cnt_unsigned;

-- Revert column types in endpoints table (potential data loss for values exceeding signed range)
ALTER TABLE endpoints ALTER COLUMN sh_addr TYPE SMALLINT
  USING (sh_addr % 65536 - CASE WHEN sh_addr >= 32768 THEN 65536 ELSE 0 END)::SMALLINT;
ALTER TABLE endpoints ALTER COLUMN attach_cnt TYPE INTEGER
  USING (CASE WHEN attach_cnt > 2147483647 THEN attach_cnt - 4294967296 ELSE attach_cnt END)::INTEGER;
ALTER TABLE endpoints ALTER COLUMN packet_cnt TYPE INTEGER
  USING (CASE WHEN packet_cnt > 2147483647 THEN packet_cnt - 4294967296 ELSE packet_cnt END)::INTEGER;
ALTER TABLE endpoints ALTER COLUMN last_packet_cnt TYPE INTEGER
  USING (CASE WHEN last_packet_cnt > 2147483647 THEN last_packet_cnt - 4294967296 ELSE last_packet_cnt END)::INTEGER;

-- Revert column types in endpoint_sessions table
ALTER TABLE endpoint_sessions ALTER COLUMN sh_addr TYPE SMALLINT
  USING (sh_addr % 65536 - CASE WHEN sh_addr >= 32768 THEN 65536 ELSE 0 END)::SMALLINT;
ALTER TABLE endpoint_sessions ALTER COLUMN attach_cnt TYPE INTEGER
  USING (CASE WHEN attach_cnt > 2147483647 THEN attach_cnt - 4294967296 ELSE attach_cnt END)::INTEGER;
-- Note: last_packet_cnt stays BIGINT (was already BIGINT), just constraint removed

COMMIT;
