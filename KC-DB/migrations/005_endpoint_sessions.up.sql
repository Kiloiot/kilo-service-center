-- Create endpoint_sessions table
-- Tracks endpoint attachment sessions and state

CREATE TABLE endpoint_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id BIGINT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Session identifiers
    session_id UUID NOT NULL,
    attach_cnt INTEGER NOT NULL,
    
    -- Session state
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'terminated')),
    
    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    
    -- Session parameters
    sh_addr INTEGER, -- Short address assigned for this session
    packet_cnt INTEGER DEFAULT 0, -- Last known packet counter
    
    -- Uplink configuration
    uplink_mode VARCHAR(20) DEFAULT 'standard' CHECK (uplink_mode IN ('standard', 'low_latency', 'repeated')),
    
    -- Downlink configuration
    dl_open BOOLEAN DEFAULT false,
    res_exp BOOLEAN DEFAULT false,
    dl_ack BOOLEAN DEFAULT false,
    repetition SMALLINT DEFAULT 1,
    
    -- Base Station association
    primary_basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
    
    -- Statistics
    uplink_count INTEGER DEFAULT 0,
    downlink_count INTEGER DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create partial unique constraint as a separate index
CREATE UNIQUE INDEX idx_endpoint_sessions_unique_active 
ON endpoint_sessions(endpoint_id) 
WHERE status = 'active';

-- Indexes
CREATE INDEX idx_endpoint_sessions_endpoint ON endpoint_sessions(endpoint_id, started_at DESC);
CREATE INDEX idx_endpoint_sessions_tenant ON endpoint_sessions(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_endpoint_sessions_session ON endpoint_sessions(session_id);
CREATE INDEX idx_endpoint_sessions_activity ON endpoint_sessions(last_activity_at DESC) WHERE status = 'active';

-- Function to update last_activity_at
CREATE OR REPLACE FUNCTION update_endpoint_session_activity()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic timestamp update
CREATE TRIGGER endpoint_sessions_update_timestamp
    BEFORE UPDATE ON endpoint_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_endpoint_session_activity();

-- Comments
COMMENT ON TABLE endpoint_sessions IS 'Endpoint attachment sessions tracking';
COMMENT ON COLUMN endpoint_sessions.session_id IS 'Unique session identifier';
COMMENT ON COLUMN endpoint_sessions.attach_cnt IS 'Attachment counter at session start';
COMMENT ON COLUMN endpoint_sessions.sh_addr IS 'Short address assigned by base station';
COMMENT ON COLUMN endpoint_sessions.packet_cnt IS 'Last known packet counter in session';
COMMENT ON COLUMN endpoint_sessions.primary_basestation_id IS 'Primary base station handling this session';
