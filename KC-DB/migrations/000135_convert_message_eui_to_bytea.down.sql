-- Revert 000135: EUI columns BYTEA(8) -> BIGINT.
-- Rows with the high bit set cannot be represented in signed BIGINT: the
-- bit(64)::bigint cast would silently reinterpret them as negative values
-- (corruption) and the ep_eui/bs_eui CHECK would then fail opaquely. The
-- precondition below fails the downgrade with a named diagnostic BEFORE any
-- column is altered.

DO $$
DECLARE
    high_bit_rows BIGINT := 0;
    archive_rows BIGINT := 0;
BEGIN
    SELECT COUNT(*) INTO high_bit_rows
    FROM messages
    WHERE get_byte(ep_eui, 0) >= 128 OR get_byte(bs_eui, 0) >= 128;

    SELECT COUNT(*) + high_bit_rows INTO high_bit_rows
    FROM endpoints
    WHERE last_attached_bs_eui IS NOT NULL
      AND get_byte(last_attached_bs_eui, 0) >= 128;

    IF to_regclass('messages_archive') IS NOT NULL THEN
        BEGIN
            EXECUTE 'SELECT COUNT(*) FROM messages_archive
                     WHERE get_byte(ep_eui, 0) >= 128 OR get_byte(bs_eui, 0) >= 128'
                INTO archive_rows;
        EXCEPTION WHEN undefined_column THEN
            -- Legacy archive layouts without BYTEA EUI columns have nothing to guard.
            archive_rows := 0;
        END;
        high_bit_rows := high_bit_rows + archive_rows;
    END IF;

    IF high_bit_rows > 0 THEN
        RAISE EXCEPTION 'KC-MIG-000135-DOWN: % row(s) carry high-bit EUI64 values that cannot fit signed BIGINT; downgrade would corrupt them. Remove or export these rows first.', high_bit_rows;
    END IF;
END $$;

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
