-- Recreate the legacy basestation_connection_events table as migration 019
-- defined it. The archived system_events rows stay in place and the table
-- comes back empty.

CREATE TABLE IF NOT EXISTS basestation_connection_events (
    id BIGSERIAL PRIMARY KEY,
    basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB DEFAULT '{}',
    remote_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_basestation_connection_events_bs ON basestation_connection_events(basestation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_basestation_connection_events_type ON basestation_connection_events(event_type, created_at DESC);

COMMENT ON TABLE basestation_connection_events IS 'Audit log of Base Station connection events';
