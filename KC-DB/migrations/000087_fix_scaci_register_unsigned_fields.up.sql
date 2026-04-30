-- Migration 087: Fix unsigned integer storage for SCACI §3.6 Register fields
-- Converts signed integer columns to larger types that can hold full unsigned range
-- with CHECK constraints to enforce non-negative values per MIOTY spec.
--
-- Affected tables: endpoints, endpoint_sessions
-- Not applicable: downlink_queue.packet_cnt (already BIGINT[])

BEGIN;

-- =============================================================================
-- PART A: endpoints table
-- =============================================================================

-- sh_addr: uint16 (0-65535) currently stored as SMALLINT
-- Fix potential negative values from wraparound (e.g., -1 should become 65535)
UPDATE endpoints SET sh_addr = sh_addr + 65536 WHERE sh_addr IS NOT NULL AND sh_addr < 0;
-- Alter column type to INTEGER (can hold all uint16 values)
ALTER TABLE endpoints ALTER COLUMN sh_addr TYPE INTEGER USING sh_addr::INTEGER;
-- Add CHECK constraint to enforce unsigned bounds
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_sh_addr_unsigned
  CHECK (sh_addr IS NULL OR (sh_addr >= 0 AND sh_addr <= 65535));
COMMENT ON COLUMN endpoints.sh_addr IS 'Short address (uint16, 0-65535) per SCACI §3.6.1';

-- attach_cnt: uint32 (0-4294967295) currently stored as INTEGER
-- Fix potential negative values from wraparound
UPDATE endpoints SET attach_cnt = attach_cnt::BIGINT + 4294967296 WHERE attach_cnt IS NOT NULL AND attach_cnt < 0;
-- Alter column type to BIGINT (can hold all uint32 values)
ALTER TABLE endpoints ALTER COLUMN attach_cnt TYPE BIGINT USING attach_cnt::BIGINT;
-- Add CHECK constraint
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_attach_cnt_unsigned
  CHECK (attach_cnt IS NULL OR (attach_cnt >= 0 AND attach_cnt <= 4294967295));
COMMENT ON COLUMN endpoints.attach_cnt IS 'Attachment counter (uint32, 0-4294967295) per SCACI §3.6.1';

-- packet_cnt: uint32 (0-4294967295) currently stored as INTEGER
UPDATE endpoints SET packet_cnt = packet_cnt::BIGINT + 4294967296 WHERE packet_cnt < 0;
ALTER TABLE endpoints ALTER COLUMN packet_cnt TYPE BIGINT USING packet_cnt::BIGINT;
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_packet_cnt_unsigned
  CHECK (packet_cnt >= 0 AND packet_cnt <= 4294967295);
COMMENT ON COLUMN endpoints.packet_cnt IS 'Packet counter (uint32, 0-4294967295) per SCACI §3.6.1';

-- last_packet_cnt: uint32 (0-4294967295) currently stored as INTEGER
UPDATE endpoints SET last_packet_cnt = last_packet_cnt::BIGINT + 4294967296 WHERE last_packet_cnt < 0;
ALTER TABLE endpoints ALTER COLUMN last_packet_cnt TYPE BIGINT USING last_packet_cnt::BIGINT;
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_last_packet_cnt_unsigned
  CHECK (last_packet_cnt >= 0 AND last_packet_cnt <= 4294967295);
COMMENT ON COLUMN endpoints.last_packet_cnt IS 'Last packet counter (uint32, 0-4294967295) per SCACI §3.6.1';

-- =============================================================================
-- PART B: endpoint_sessions table (dependent)
-- =============================================================================

-- sh_addr: SMALLINT → INTEGER with CHECK
UPDATE endpoint_sessions SET sh_addr = sh_addr + 65536 WHERE sh_addr IS NOT NULL AND sh_addr < 0;
ALTER TABLE endpoint_sessions ALTER COLUMN sh_addr TYPE INTEGER USING sh_addr::INTEGER;
ALTER TABLE endpoint_sessions ADD CONSTRAINT chk_endpoint_sessions_sh_addr_unsigned
  CHECK (sh_addr IS NULL OR (sh_addr >= 0 AND sh_addr <= 65535));

-- attach_cnt: INTEGER → BIGINT with CHECK
UPDATE endpoint_sessions SET attach_cnt = attach_cnt::BIGINT + 4294967296 WHERE attach_cnt < 0;
ALTER TABLE endpoint_sessions ALTER COLUMN attach_cnt TYPE BIGINT USING attach_cnt::BIGINT;
ALTER TABLE endpoint_sessions ADD CONSTRAINT chk_endpoint_sessions_attach_cnt_unsigned
  CHECK (attach_cnt >= 0 AND attach_cnt <= 4294967295);

-- last_packet_cnt: already BIGINT, add CHECK constraint only
ALTER TABLE endpoint_sessions ADD CONSTRAINT chk_endpoint_sessions_last_packet_cnt_unsigned
  CHECK (last_packet_cnt IS NULL OR (last_packet_cnt >= 0 AND last_packet_cnt <= 4294967295));

-- =============================================================================
-- PART C: downlink_queue table - NOT APPLICABLE
-- downlink_queue.packet_cnt is BIGINT[] (array of bigint), already wide enough.
-- No structural change needed. Values stored are uint32 from wire protocol.
-- =============================================================================

COMMIT;
