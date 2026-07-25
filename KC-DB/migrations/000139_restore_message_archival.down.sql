-- Revert to the pre-000139 state: messages without archival columns and the
-- pre-000047 messages_archive layout inherited from migrations 001/014.
--
-- Non-destructive: a corrected archive holding rows blocks the downgrade with a
-- named diagnostic instead of silently destroying archived data. A legacy
-- archive preserved by the up migration (messages_archive_pre000139) is
-- restored by rename; otherwise the legacy layout is recreated empty.

DO $$
DECLARE
    archived_rows BIGINT;
BEGIN
    IF to_regclass('messages_archive') IS NOT NULL THEN
        EXECUTE 'SELECT COUNT(*) FROM messages_archive' INTO archived_rows;
        IF archived_rows > 0 THEN
            RAISE EXCEPTION 'KC-MIG-000139-DOWN: messages_archive holds % row(s); downgrade would destroy archived messages. Export or migrate them first.', archived_rows;
        END IF;
        EXECUTE 'DROP TRIGGER IF EXISTS set_messages_archive_timestamp ON messages_archive';
        EXECUTE 'DROP TABLE messages_archive';
    END IF;
END $$;

DROP INDEX IF EXISTS idx_messages_archived;
ALTER TABLE messages DROP COLUMN IF EXISTS archived;
ALTER TABLE messages DROP COLUMN IF EXISTS archived_at;

-- Restore the preserved legacy archive if the up migration kept one; otherwise
-- recreate the legacy layout (001-era messages columns plus the
-- archived/archived_at columns that migration 014 relied on).
DO $$
BEGIN
    IF to_regclass('messages_archive_pre000139') IS NOT NULL THEN
        EXECUTE 'ALTER TABLE messages_archive_pre000139 RENAME TO messages_archive';
        IF to_regclass('messages_archive_pre000139_pkey') IS NOT NULL THEN
            EXECUTE 'ALTER INDEX messages_archive_pre000139_pkey RENAME TO messages_archive_pkey';
        END IF;
        IF to_regclass('idx_messages_archive_pre000139_ep_eui') IS NOT NULL THEN
            EXECUTE 'ALTER INDEX idx_messages_archive_pre000139_ep_eui RENAME TO idx_messages_archive_ep_eui';
        END IF;
        IF to_regclass('idx_messages_archive_pre000139_tenant_id') IS NOT NULL THEN
            EXECUTE 'ALTER INDEX idx_messages_archive_pre000139_tenant_id RENAME TO idx_messages_archive_tenant_id';
        END IF;
        IF to_regclass('idx_messages_archive_pre000139_received_at') IS NOT NULL THEN
            EXECUTE 'ALTER INDEX idx_messages_archive_pre000139_received_at RENAME TO idx_messages_archive_received_at';
        END IF;
        IF to_regclass('idx_messages_archive_pre000139_archived_at') IS NOT NULL THEN
            EXECUTE 'ALTER INDEX idx_messages_archive_pre000139_archived_at RENAME TO idx_messages_archive_archived_at';
        END IF;
        RAISE NOTICE 'KC-MIG-000139-DOWN: restored preserved legacy archive';
        RETURN;
    END IF;

    CREATE TABLE messages_archive (
        id BIGINT NOT NULL,
        ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),
        bs_eui BYTEA NOT NULL CHECK (length(bs_eui) = 8),
        tenant_id BIGINT NOT NULL,
        payload BYTEA NOT NULL,
        frame_count INTEGER NOT NULL,
        rssi REAL NOT NULL,
        snr REAL NOT NULL,
        eq_snr REAL NOT NULL,
        frequency DOUBLE PRECISION NOT NULL,
        uplink_mode VARCHAR(10) DEFAULT 'standard',
        packet_counter INTEGER,
        dl_open BOOLEAN DEFAULT false,
        res_exp BOOLEAN DEFAULT false,
        dl_ack BOOLEAN DEFAULT false,
        is_roaming BOOLEAN DEFAULT false,
        home_network_id VARCHAR(50),
        roaming_agreement_id BIGINT,
        received_at TIMESTAMP WITH TIME ZONE NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        archived BOOLEAN DEFAULT false,
        archived_at TIMESTAMP WITH TIME ZONE,
        PRIMARY KEY (id, received_at)
    );
END $$;

CREATE INDEX IF NOT EXISTS idx_messages_archive_ep_eui ON messages_archive(ep_eui);
CREATE INDEX IF NOT EXISTS idx_messages_archive_tenant_id ON messages_archive(tenant_id);
CREATE INDEX IF NOT EXISTS idx_messages_archive_received_at ON messages_archive(received_at);
CREATE INDEX IF NOT EXISTS idx_messages_archive_archived_at ON messages_archive(archived_at);

CREATE TRIGGER set_messages_archive_timestamp
    BEFORE INSERT ON messages_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();
