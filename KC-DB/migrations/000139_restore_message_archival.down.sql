-- Revert to the pre-000139 state: messages without archival columns and the
-- pre-000047 messages_archive layout inherited from migrations 001/014.

DROP TRIGGER IF EXISTS set_messages_archive_timestamp ON messages_archive;
DROP TABLE IF EXISTS messages_archive;

DROP INDEX IF EXISTS idx_messages_archived;
ALTER TABLE messages DROP COLUMN IF EXISTS archived;
ALTER TABLE messages DROP COLUMN IF EXISTS archived_at;

-- Recreate the legacy archive layout (001-era messages columns plus the
-- archived/archived_at columns that migration 014 relied on).
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

CREATE INDEX IF NOT EXISTS idx_messages_archive_ep_eui ON messages_archive(ep_eui);
CREATE INDEX IF NOT EXISTS idx_messages_archive_tenant_id ON messages_archive(tenant_id);
CREATE INDEX IF NOT EXISTS idx_messages_archive_received_at ON messages_archive(received_at);
CREATE INDEX IF NOT EXISTS idx_messages_archive_archived_at ON messages_archive(archived_at);

CREATE TRIGGER set_messages_archive_timestamp
    BEFORE INSERT ON messages_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();
