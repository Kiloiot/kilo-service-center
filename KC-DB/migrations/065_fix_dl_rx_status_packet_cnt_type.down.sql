-- Remove CHECK constraint
ALTER TABLE dl_rx_status
    DROP CONSTRAINT IF EXISTS packet_cnt_uint32_range;

-- Revert comment
COMMENT ON COLUMN dl_rx_status.packet_cnt IS
    'End Point packet counter';
