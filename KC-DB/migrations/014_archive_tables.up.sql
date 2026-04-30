-- Create archive tables for data retention and historical queries

-- Messages archive table (already exists from initial schema, but adding if not exists)
CREATE TABLE IF NOT EXISTS messages_archive (
    LIKE messages INCLUDING ALL
);

-- Add archived_at column if it doesn't exist
ALTER TABLE messages_archive ADD COLUMN IF NOT EXISTS archived_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Basestation receptions archive table
CREATE TABLE IF NOT EXISTS basestation_receptions_archive (
    LIKE basestation_receptions INCLUDING ALL
);

-- Add archived_at column
ALTER TABLE basestation_receptions_archive ADD COLUMN archived_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Endpoint sessions archive table
CREATE TABLE IF NOT EXISTS endpoint_sessions_archive (
    LIKE endpoint_sessions INCLUDING ALL
);

-- Add archived_at column
ALTER TABLE endpoint_sessions_archive ADD COLUMN archived_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Endpoint keys archive table
CREATE TABLE IF NOT EXISTS endpoint_keys_archive (
    LIKE endpoint_keys INCLUDING ALL
);

-- Add archived_at column
ALTER TABLE endpoint_keys_archive ADD COLUMN archived_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Create indexes for archive tables
CREATE INDEX IF NOT EXISTS idx_messages_archive_ep_eui ON messages_archive(ep_eui);
CREATE INDEX IF NOT EXISTS idx_messages_archive_tenant_id ON messages_archive(tenant_id);
CREATE INDEX IF NOT EXISTS idx_messages_archive_received_at ON messages_archive(received_at);
CREATE INDEX IF NOT EXISTS idx_messages_archive_archived_at ON messages_archive(archived_at);

CREATE INDEX IF NOT EXISTS idx_basestation_receptions_archive_message_id ON basestation_receptions_archive(message_id);
CREATE INDEX IF NOT EXISTS idx_basestation_receptions_archive_basestation_id ON basestation_receptions_archive(basestation_id);
CREATE INDEX IF NOT EXISTS idx_basestation_receptions_archive_created_at ON basestation_receptions_archive(created_at);
CREATE INDEX IF NOT EXISTS idx_basestation_receptions_archive_archived_at ON basestation_receptions_archive(archived_at);

CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_archive_endpoint_id ON endpoint_sessions_archive(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_archive_tenant_id ON endpoint_sessions_archive(tenant_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_archive_status ON endpoint_sessions_archive(status);
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_archive_ended_at ON endpoint_sessions_archive(ended_at);
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_archive_archived_at ON endpoint_sessions_archive(archived_at);

CREATE INDEX IF NOT EXISTS idx_endpoint_keys_archive_endpoint_id ON endpoint_keys_archive(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_archive_tenant_id ON endpoint_keys_archive(tenant_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_archive_key_type ON endpoint_keys_archive(key_type);
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_archive_archived_at ON endpoint_keys_archive(archived_at);

-- Create function to automatically set archived_at timestamp
CREATE OR REPLACE FUNCTION set_archived_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.archived_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for archive tables
DROP TRIGGER IF EXISTS set_messages_archive_timestamp ON messages_archive;
CREATE TRIGGER set_messages_archive_timestamp
    BEFORE INSERT ON messages_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();

DROP TRIGGER IF EXISTS set_basestation_receptions_archive_timestamp ON basestation_receptions_archive;
CREATE TRIGGER set_basestation_receptions_archive_timestamp
    BEFORE INSERT ON basestation_receptions_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();

DROP TRIGGER IF EXISTS set_endpoint_sessions_archive_timestamp ON endpoint_sessions_archive;
CREATE TRIGGER set_endpoint_sessions_archive_timestamp
    BEFORE INSERT ON endpoint_sessions_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();

DROP TRIGGER IF EXISTS set_endpoint_keys_archive_timestamp ON endpoint_keys_archive;
CREATE TRIGGER set_endpoint_keys_archive_timestamp
    BEFORE INSERT ON endpoint_keys_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();

-- Add comments
COMMENT ON TABLE messages_archive IS 'Archive table for old messages';
COMMENT ON TABLE basestation_receptions_archive IS 'Archive table for old basestation reception records';
COMMENT ON TABLE endpoint_sessions_archive IS 'Archive table for terminated/expired endpoint sessions';
COMMENT ON TABLE endpoint_keys_archive IS 'Archive table for inactive/expired endpoint keys';

COMMENT ON COLUMN messages_archive.archived_at IS 'Timestamp when the record was archived';
COMMENT ON COLUMN basestation_receptions_archive.archived_at IS 'Timestamp when the record was archived';
COMMENT ON COLUMN endpoint_sessions_archive.archived_at IS 'Timestamp when the record was archived';
COMMENT ON COLUMN endpoint_keys_archive.archived_at IS 'Timestamp when the record was archived';
