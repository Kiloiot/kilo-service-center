-- Migration 000139: restore message archival columns and rebuild messages_archive.
-- Migration 000047 rebuilt the messages table without the archived/archived_at
-- columns while messages_archive kept the pre-000047 column layout, so the
-- archival copy from messages into messages_archive failed at runtime.

-- Restore archival tracking columns on the live table.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS archived BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP WITH TIME ZONE;

-- Partial index used by the archival sweep and the purge of archived rows.
CREATE INDEX IF NOT EXISTS idx_messages_archived ON messages(archived, received_at) WHERE archived = false;

-- Rebuild the archive table so its column layout matches the live table.
DROP TABLE IF EXISTS messages_archive;
CREATE TABLE messages_archive (
    LIKE messages INCLUDING DEFAULTS INCLUDING CONSTRAINTS
);

-- Primary key keeps the copy-if-absent archival insert deduplicated.
ALTER TABLE messages_archive ADD PRIMARY KEY (id);

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
