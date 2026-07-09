-- Revert 000135: EUI columns BYTEA(8) -> BIGINT.
-- Rows with the high bit set can't fit signed BIGINT and will fail this revert (expected).

ALTER TABLE messages DROP CONSTRAINT IF EXISTS valid_euis;

ALTER TABLE messages
    ALTER COLUMN ep_eui TYPE BIGINT USING ('x' || encode(ep_eui, 'hex'))::bit(64)::bigint,
    ALTER COLUMN bs_eui TYPE BIGINT USING ('x' || encode(bs_eui, 'hex'))::bit(64)::bigint;

ALTER TABLE messages ADD CONSTRAINT valid_euis CHECK (ep_eui > 0 AND bs_eui > 0);

ALTER TABLE endpoints
    ALTER COLUMN last_attached_bs_eui TYPE BIGINT
        USING CASE WHEN last_attached_bs_eui IS NULL THEN NULL
                   ELSE ('x' || encode(last_attached_bs_eui, 'hex'))::bit(64)::bigint END;

COMMENT ON COLUMN messages.ep_eui IS 'End Point EUI64 (stored as BIGINT for efficiency)';
COMMENT ON COLUMN messages.bs_eui IS 'Base Station EUI64 that received this';
COMMENT ON COLUMN endpoints.last_attached_bs_eui IS 'Last BS EUI that handled attach/detach propagate (BSSCI §5.6/5.7)';
