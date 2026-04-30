-- BSSCI §5.15.1: packet_cnt is uint32 (0 to 4,294,967,295)
-- PostgreSQL INTEGER maxes at 2,147,483,647, so we keep BIGINT with CHECK constraint

-- Add CHECK constraint for uint32 range
ALTER TABLE dl_rx_status
    ADD CONSTRAINT packet_cnt_uint32_range
    CHECK (packet_cnt >= 0 AND packet_cnt <= 4294967295);

-- Update comment to clarify type
COMMENT ON COLUMN dl_rx_status.packet_cnt IS
    'End Point packet counter (uint32: 0-4294967295) per BSSCI §5.15.1. Stored as BIGINT with CHECK constraint for full uint32 range.';
