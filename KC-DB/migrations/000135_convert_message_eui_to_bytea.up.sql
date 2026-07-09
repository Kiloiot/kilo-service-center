-- Migration 000135: EUI columns BIGINT -> BYTEA(8).
-- BIGINT is signed int64 but EUI-64 is uint64 — EUIs with the high bit set overflowed.

-- Drop the numeric check first: "ep_eui > 0" is invalid on BYTEA.
ALTER TABLE messages DROP CONSTRAINT IF EXISTS valid_euis;

-- Convert existing values to 8-byte big-endian.
ALTER TABLE messages
    ALTER COLUMN ep_eui TYPE BYTEA USING decode(lpad(to_hex(ep_eui), 16, '0'), 'hex'),
    ALTER COLUMN bs_eui TYPE BYTEA USING decode(lpad(to_hex(bs_eui), 16, '0'), 'hex');

-- Length check replaces the numeric one.
ALTER TABLE messages ADD CONSTRAINT valid_euis
    CHECK (length(ep_eui) = 8 AND length(bs_eui) = 8);

-- endpoints.last_attached_bs_eui: BIGINT -> BYTEA (nullable).
ALTER TABLE endpoints
    ALTER COLUMN last_attached_bs_eui TYPE BYTEA
        USING CASE WHEN last_attached_bs_eui IS NULL THEN NULL
                   ELSE decode(lpad(to_hex(last_attached_bs_eui), 16, '0'), 'hex') END;

COMMENT ON COLUMN messages.ep_eui IS 'End Point EUI64 (8-byte big-endian BYTEA)';
COMMENT ON COLUMN messages.bs_eui IS 'Base Station EUI64 that received this (8-byte big-endian BYTEA)';
COMMENT ON COLUMN endpoints.last_attached_bs_eui IS 'Last BS EUI for attach/detach propagate (8-byte big-endian BYTEA)';
