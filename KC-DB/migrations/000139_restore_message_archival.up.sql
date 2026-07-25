-- Migration 000139: restore message archival columns and rebuild messages_archive.
-- Migration 000047 rebuilt the messages table without the archived/archived_at
-- columns while messages_archive kept the pre-000047 (001-era) column layout, so
-- the archival copy from messages into messages_archive failed at runtime.
--
-- Non-destructive rebuild: the new archive is created under a temporary name with
-- an explicit column list mirroring the live messages table, then swapped in.
-- A legacy archive that still holds rows is PRESERVED under
-- messages_archive_pre000139 -- its 001-era layout (BIGINT id, payload/frame_count)
-- cannot be losslessly projected into the 000047-era shape, so rows are kept
-- queryable in place rather than re-identified. An empty legacy archive is dropped
-- only after its row count is proven zero.
--
-- Ownership contract after this migration:
--   * The canonical messages_archive receives ALL new archival writes; the
--     legacy table is never inserted into again.
--   * The legacy table preserves its original schema losslessly.
--   * Identity maintenance (EUI renames) and archival statistics cover BOTH
--     tables (repository helper updateLegacyArchiveEUI and GetArchivalStats
--     resolve the legacy table via to_regclass).
--   * The partition listing excludes both archive tables.

-- Restore archival tracking columns on the live table.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP WITH TIME ZONE;

-- Partial index used by the archival sweep and the purge of archived rows.
CREATE INDEX IF NOT EXISTS idx_messages_archived ON messages(archived, received_at) WHERE archived = false;

-- Build the corrected archive under a temporary name with an explicit column
-- list (kept in sync with the messages layout as of this migration).
CREATE TABLE messages_archive_new_000139 (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL,
    command_type VARCHAR(20) NOT NULL DEFAULT 'ulData',
    op_id BIGINT NOT NULL,
    ep_eui BYTEA NOT NULL,
    bs_eui BYTEA NOT NULL,
    rx_time BIGINT NOT NULL,
    packet_cnt INTEGER NOT NULL,
    snr DOUBLE PRECISION NOT NULL,
    rssi DOUBLE PRECISION NOT NULL,
    user_data BYTEA,
    dl_open BOOLEAN NOT NULL DEFAULT false,
    response_exp BOOLEAN NOT NULL DEFAULT false,
    dl_ack BOOLEAN NOT NULL DEFAULT false,
    rx_duration BIGINT,
    eq_snr DOUBLE PRECISION,
    profile VARCHAR(20),
    mode VARCHAR(20),
    format SMALLINT DEFAULT 0,
    subpackets JSONB,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    org_uuid UUID,
    owner_tenant_id BIGINT,
    base_stations JSONB,
    duplicate BOOLEAN NOT NULL DEFAULT false,
    decoded_payload JSONB,
    blueprint_type_eui BYTEA,
    blueprint_version_id UUID,
    decode_status VARCHAR(32) DEFAULT 'pending',
    decode_error_code VARCHAR(64),
    nwk_sn_key BYTEA,
    archived BOOLEAN NOT NULL DEFAULT false,
    archived_at TIMESTAMPTZ,
    CONSTRAINT messages_archive_new_000139_pkey PRIMARY KEY (id),
    CONSTRAINT messages_archive_valid_euis CHECK (length(ep_eui) = 8 AND length(bs_eui) = 8),
    CONSTRAINT messages_archive_valid_tenant_id CHECK (tenant_id > 0),
    CONSTRAINT messages_archive_valid_owner_tenant_id CHECK (owner_tenant_id IS NULL OR owner_tenant_id > 0),
    CONSTRAINT messages_archive_valid_packet_cnt CHECK (packet_cnt >= 0),
    CONSTRAINT messages_archive_valid_format CHECK (format >= 0 AND format <= 255),
    CONSTRAINT messages_archive_blueprint_type_eui_check CHECK (blueprint_type_eui IS NULL OR length(blueprint_type_eui) = 8)
);

-- Preserve or retire the legacy archive, then swap the corrected table in.
DO $$
DECLARE
    legacy_rows BIGINT;
BEGIN
    IF to_regclass('messages_archive') IS NOT NULL THEN
        EXECUTE 'SELECT COUNT(*) FROM messages_archive' INTO legacy_rows;
        IF legacy_rows > 0 THEN
            -- 001-era rows cannot be losslessly projected into the 000047-era
            -- shape; keep them queryable under a preserved name. Free the
            -- canonical index name so the corrected table can claim it.
            EXECUTE 'DROP TRIGGER IF EXISTS set_messages_archive_timestamp ON messages_archive';
            EXECUTE 'ALTER TABLE messages_archive RENAME TO messages_archive_pre000139';
            IF to_regclass('messages_archive_pkey') IS NOT NULL THEN
                EXECUTE 'ALTER INDEX messages_archive_pkey RENAME TO messages_archive_pre000139_pkey';
            END IF;
            -- The query indexes stay attached to the renamed table; free
            -- their canonical names for the corrected table
            IF to_regclass('idx_messages_archive_ep_eui') IS NOT NULL THEN
                EXECUTE 'ALTER INDEX idx_messages_archive_ep_eui RENAME TO idx_messages_archive_pre000139_ep_eui';
            END IF;
            IF to_regclass('idx_messages_archive_tenant_id') IS NOT NULL THEN
                EXECUTE 'ALTER INDEX idx_messages_archive_tenant_id RENAME TO idx_messages_archive_pre000139_tenant_id';
            END IF;
            IF to_regclass('idx_messages_archive_received_at') IS NOT NULL THEN
                EXECUTE 'ALTER INDEX idx_messages_archive_received_at RENAME TO idx_messages_archive_pre000139_received_at';
            END IF;
            IF to_regclass('idx_messages_archive_archived_at') IS NOT NULL THEN
                EXECUTE 'ALTER INDEX idx_messages_archive_archived_at RENAME TO idx_messages_archive_pre000139_archived_at';
            END IF;
            RAISE NOTICE 'KC-MIG-000139: preserved % legacy archive row(s) in messages_archive_pre000139', legacy_rows;
        ELSE
            -- Proven empty: nothing to preserve.
            EXECUTE 'DROP TABLE messages_archive';
            RAISE NOTICE 'KC-MIG-000139: legacy archive was empty; dropped';
        END IF;
    END IF;

    EXECUTE 'ALTER TABLE messages_archive_new_000139 RENAME TO messages_archive';
    EXECUTE 'ALTER INDEX messages_archive_new_000139_pkey RENAME TO messages_archive_pkey';
END $$;

-- Query indexes for the archive table (names follow migration 014).
CREATE INDEX idx_messages_archive_ep_eui ON messages_archive(ep_eui);
CREATE INDEX idx_messages_archive_tenant_id ON messages_archive(tenant_id);
CREATE INDEX idx_messages_archive_received_at ON messages_archive(received_at);
CREATE INDEX idx_messages_archive_archived_at ON messages_archive(archived_at);

-- Stamp archived_at on insert (set_archived_at() is created by migration 014).
CREATE TRIGGER set_messages_archive_timestamp
    BEFORE INSERT ON messages_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();

COMMENT ON TABLE messages_archive IS 'Archive table for old messages (layout mirrors messages)';
COMMENT ON COLUMN messages.archived IS 'True once the row has been copied to messages_archive';
COMMENT ON COLUMN messages.archived_at IS 'Timestamp when the row was copied to messages_archive';
