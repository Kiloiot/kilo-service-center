-- Migration to fix MIOTY security schema alignment
-- Adds missing security fields and fixes identifier type mismatches

-- Add missing MIOTY security fields to endpoints table
ALTER TABLE endpoints 
ADD COLUMN IF NOT EXISTS nwk_addr INTEGER, -- Network address assigned during join
ADD COLUMN IF NOT EXISTS ep_eui_alt BYTEA CHECK (length(ep_eui_alt) = 8); -- Alternative Endpoint EUI (if different from main EUI)

-- Add session key to endpoint_sessions table
ALTER TABLE endpoint_sessions
ADD COLUMN IF NOT EXISTS session_key BYTEA CHECK (length(session_key) = 16); -- NWKSnKey for session security

-- Create a new endpoint_sessions table with correct foreign key types
-- First, rename the old table
ALTER TABLE endpoint_sessions RENAME TO endpoint_sessions_old;

-- Create new table with BIGINT foreign keys matching the main schema
CREATE TABLE endpoint_sessions (
    id BIGSERIAL PRIMARY KEY,
    endpoint_id BIGINT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Session identifiers
    session_id UUID NOT NULL,
    attach_cnt INTEGER NOT NULL,
    
    -- MIOTY session security
    session_key BYTEA CHECK (length(session_key) = 16), -- NWKSnKey
    
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
    
    -- Basestation association
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
DROP INDEX IF EXISTS idx_endpoint_sessions_unique_active;
CREATE UNIQUE INDEX IF NOT EXISTS idx_endpoint_sessions_unique_active 
ON endpoint_sessions(endpoint_id) 
WHERE status = 'active';

-- Create indexes for new table
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_endpoint ON endpoint_sessions(endpoint_id, status);
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_tenant ON endpoint_sessions(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_active ON endpoint_sessions(status, last_activity_at DESC) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_endpoint_sessions_basestation ON endpoint_sessions(primary_basestation_id) WHERE primary_basestation_id IS NOT NULL;

-- Drop old table
DROP TABLE endpoint_sessions_old;

-- Fix endpoint_keys table foreign key types
ALTER TABLE endpoint_keys RENAME TO endpoint_keys_old;

CREATE TABLE endpoint_keys (
    id BIGSERIAL PRIMARY KEY,
    endpoint_id BIGINT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Key type and version
    key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('network', 'application', 'join', 'session')),
    key_version INTEGER NOT NULL DEFAULT 1,
    
    -- Key data (encrypted at rest in production)
    key_value BYTEA NOT NULL,
    
    -- Key metadata
    key_id VARCHAR(64), -- External key identifier
    algorithm VARCHAR(20) DEFAULT 'AES-128' CHECK (algorithm IN ('AES-128', 'AES-256')),
    
    -- Validity period
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    
    -- Usage tracking
    usage_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    
    -- Key rotation
    rotated_from BIGINT REFERENCES endpoint_keys(id),
    rotation_reason VARCHAR(100),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255)
);

-- Create partial unique constraint as a separate index
DROP INDEX IF EXISTS idx_endpoint_keys_unique_active;
CREATE UNIQUE INDEX IF NOT EXISTS idx_endpoint_keys_unique_active 
ON endpoint_keys(endpoint_id, key_type) 
WHERE is_active = true;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_endpoint ON endpoint_keys(endpoint_id, key_type, is_active);
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_tenant ON endpoint_keys(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_validity ON endpoint_keys(valid_from, valid_until) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_endpoint_keys_rotation ON endpoint_keys(rotated_from) WHERE rotated_from IS NOT NULL;

-- Drop old table
DROP TABLE endpoint_keys_old;

-- Fix basestation_sessions table if it exists
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'basestation_sessions') THEN
        ALTER TABLE basestation_sessions RENAME TO basestation_sessions_old;
        
        CREATE TABLE basestation_sessions (
            id BIGSERIAL PRIMARY KEY,
            basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
            tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
            
            -- Session info
            session_id UUID NOT NULL,
            connection_id VARCHAR(255),
            
            -- Connection details
            remote_addr INET,
            protocol_version VARCHAR(20),
            
            -- Status
            status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disconnected', 'terminated')),
            
            -- Timing
            started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            last_ping_at TIMESTAMPTZ,
            ended_at TIMESTAMPTZ,
            
            -- Statistics
            messages_received BIGINT DEFAULT 0,
            messages_sent BIGINT DEFAULT 0,
            bytes_received BIGINT DEFAULT 0,
            bytes_sent BIGINT DEFAULT 0,
            
            -- Metadata
            metadata JSONB DEFAULT '{}',
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        
        -- Create partial unique constraint as a separate index
        DROP INDEX IF EXISTS idx_basestation_sessions_unique_active;
        CREATE UNIQUE INDEX IF NOT EXISTS idx_basestation_sessions_unique_active 
        ON basestation_sessions(basestation_id) 
        WHERE status = 'active';
        
        -- Create indexes
        CREATE INDEX IF NOT EXISTS idx_basestation_sessions_basestation ON basestation_sessions(basestation_id, status);
        CREATE INDEX IF NOT EXISTS idx_basestation_sessions_tenant ON basestation_sessions(tenant_id, created_at DESC);
        CREATE INDEX IF NOT EXISTS idx_basestation_sessions_active ON basestation_sessions(status, last_ping_at DESC) WHERE status = 'active';
        
        DROP TABLE basestation_sessions_old;
    END IF;
END $$;

-- Add comment explaining the security model
COMMENT ON COLUMN endpoints.nwk_key IS 'Root network key (NwkKey) used to derive session keys';
COMMENT ON COLUMN endpoints.app_key IS 'Application key (AppKey) for end-to-end encryption';
COMMENT ON COLUMN endpoints.nwk_addr IS 'Network address assigned during endpoint activation';
COMMENT ON COLUMN endpoints.ep_eui_alt IS 'Alternative Endpoint EUI if different from main EUI (usually NULL)';
COMMENT ON COLUMN endpoint_sessions.session_key IS 'Network session key (NWKSnKey) derived from network key for this session';
