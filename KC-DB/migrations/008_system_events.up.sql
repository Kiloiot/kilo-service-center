-- Create system_events table
-- Tracks system events, audit logs, and important state changes

CREATE TABLE system_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT REFERENCES tenants(id) ON DELETE CASCADE,

    -- Event classification
    event_type VARCHAR(50) NOT NULL,
    event_category VARCHAR(50) NOT NULL CHECK (event_category IN (
        'security', 'endpoint', 'basestation', 'message', 'system', 'roaming', 'error', 'audit'
    )),
    severity VARCHAR(20) NOT NULL DEFAULT 'info' CHECK (severity IN (
        'debug', 'info', 'warning', 'error', 'critical'
    )),

    -- Event source
    source_type VARCHAR(50),
    source_id UUID,
    source_name VARCHAR(255),

    -- Event details
    event_code VARCHAR(50),
    title VARCHAR(255) NOT NULL,
    description TEXT,

    -- Related entities
    endpoint_id BIGINT REFERENCES endpoints(id) ON DELETE SET NULL,
    basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
    message_id BIGINT, -- Reference to messages.id (no FK constraint due to partitioning)
    user_id VARCHAR(255),

    -- Event data
    data JSONB DEFAULT '{}',
    tags HSTORE,

    -- Tracking
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Indexing helper columns
    searchable_text TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(title, '') || ' ' ||
            coalesce(description, '') || ' ' ||
            coalesce(event_type, '') || ' ' ||
            coalesce(source_name, '')
        )
    ) STORED
);

-- Indexes
CREATE INDEX idx_system_events_tenant ON system_events(tenant_id, occurred_at DESC);
CREATE INDEX idx_system_events_category ON system_events(event_category, occurred_at DESC);
CREATE INDEX idx_system_events_severity ON system_events(severity, occurred_at DESC) WHERE severity IN ('error', 'critical');
CREATE INDEX idx_system_events_endpoint ON system_events(endpoint_id, occurred_at DESC) WHERE endpoint_id IS NOT NULL;
CREATE INDEX idx_system_events_basestation ON system_events(basestation_id, occurred_at DESC) WHERE basestation_id IS NOT NULL;
CREATE INDEX idx_system_events_search ON system_events USING gin(searchable_text);
CREATE INDEX idx_system_events_data ON system_events USING gin(data);
CREATE INDEX idx_system_events_tags ON system_events USING gin(tags);

-- Partitioning by month for scalability
-- Note: This is a template - actual partitioning would be done via maintenance scripts
COMMENT ON TABLE system_events IS 'System events and audit logs - consider partitioning by month for large deployments';

-- Function to auto-expire old events
CREATE OR REPLACE FUNCTION expire_old_system_events()
RETURNS void AS $$
BEGIN
    DELETE FROM system_events 
    WHERE occurred_at < NOW() - INTERVAL '180 days'
    AND severity NOT IN ('error', 'critical')
    AND event_category NOT IN ('security', 'audit');
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON COLUMN system_events.event_type IS 'Specific event type (e.g., endpoint.attached, basestation.connected)';
COMMENT ON COLUMN system_events.event_category IS 'High-level event category for filtering';
COMMENT ON COLUMN system_events.source_type IS 'Type of source that generated the event';
COMMENT ON COLUMN system_events.source_id IS 'UUID of the source entity';
COMMENT ON COLUMN system_events.event_code IS 'Machine-readable event code for automation';
COMMENT ON COLUMN system_events.data IS 'Additional event data in JSON format';
COMMENT ON COLUMN system_events.tags IS 'Key-value tags for flexible categorization';
COMMENT ON COLUMN system_events.searchable_text IS 'Full-text search index';
