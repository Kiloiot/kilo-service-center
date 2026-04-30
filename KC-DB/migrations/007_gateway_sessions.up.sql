-- Create basestation_sessions table
-- Tracks Base Station connection sessions and health

CREATE TABLE basestation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Session identification
    session_id UUID NOT NULL,
    connection_id VARCHAR(255), -- External connection identifier
    
    -- Connection details
    remote_addr INET NOT NULL,
    protocol VARCHAR(20) DEFAULT 'tcp' CHECK (protocol IN ('tcp', 'udp', 'websocket')),
    tls_version VARCHAR(10),
    
    -- Session state
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disconnected', 'error')),
    error_message TEXT,
    
    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_ping_at TIMESTAMPTZ,
    last_message_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    
    -- Statistics
    messages_received BIGINT DEFAULT 0,
    messages_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    bytes_sent BIGINT DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    
    -- Health metrics
    latency_ms INTEGER,
    packet_loss_percent REAL,
    
    -- Metadata
    user_agent VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_basestation_sessions_bs ON basestation_sessions(basestation_id, started_at DESC);
CREATE INDEX idx_basestation_sessions_tenant ON basestation_sessions(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_basestation_sessions_session ON basestation_sessions(session_id);
CREATE INDEX idx_basestation_sessions_activity ON basestation_sessions(last_message_at DESC) WHERE status = 'active';
CREATE INDEX idx_basestation_sessions_errors ON basestation_sessions(basestation_id, errors_count) WHERE errors_count > 0;

-- Create partial unique constraint as a separate index
CREATE UNIQUE INDEX idx_basestation_sessions_unique_active 
ON basestation_sessions(basestation_id) 
WHERE status = 'active';

-- Comments
COMMENT ON TABLE basestation_sessions IS 'Base Station connection sessions tracking';
COMMENT ON COLUMN basestation_sessions.session_id IS 'Unique session identifier';
COMMENT ON COLUMN basestation_sessions.connection_id IS 'External connection ID from Base Station';
COMMENT ON COLUMN basestation_sessions.remote_addr IS 'Base Station IP address';
COMMENT ON COLUMN basestation_sessions.tls_version IS 'TLS version used for secure connections';
COMMENT ON COLUMN basestation_sessions.latency_ms IS 'Average round-trip latency in milliseconds';
COMMENT ON COLUMN basestation_sessions.packet_loss_percent IS 'Packet loss percentage for session';
