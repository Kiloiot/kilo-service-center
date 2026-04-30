-- Rollback migration for MIOTY security schema fixes

-- Remove added columns from endpoints table
ALTER TABLE endpoints
DROP COLUMN IF EXISTS nwk_addr,
DROP COLUMN IF EXISTS ep_eui_alt;

-- Revert endpoint_sessions table to UUID foreign keys
ALTER TABLE endpoint_sessions RENAME TO endpoint_sessions_old;

-- Drop the old table (this also drops all its indexes)
DROP TABLE endpoint_sessions_old;

-- Recreate with UUID foreign keys (as it was in migration 005)
CREATE TABLE endpoint_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL, -- Note: This will fail if endpoints table doesn't have UUID id
    tenant_id UUID NOT NULL, -- Note: This will fail if tenants table doesn't have UUID id
    
    -- Session identifiers
    session_id UUID NOT NULL,
    attach_cnt INTEGER NOT NULL,
    
    -- Session state (without session_key)
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'terminated')),
    
    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    
    -- Session parameters
    sh_addr INTEGER,
    packet_cnt INTEGER DEFAULT 0,
    
    -- Uplink configuration
    uplink_mode VARCHAR(20) DEFAULT 'standard' CHECK (uplink_mode IN ('standard', 'low_latency', 'repeated')),
    
    -- Downlink configuration
    dl_open BOOLEAN DEFAULT false,
    res_exp BOOLEAN DEFAULT false,
    dl_ack BOOLEAN DEFAULT false,
    repetition SMALLINT DEFAULT 1,
    
    -- Basestation association
    primary_basestation_id UUID,
    
    -- Statistics
    uplink_count INTEGER DEFAULT 0,
    downlink_count INTEGER DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recreate indexes (including partial unique constraint as index)
CREATE UNIQUE INDEX idx_endpoint_sessions_unique_active ON endpoint_sessions(endpoint_id) WHERE status = 'active';
CREATE INDEX idx_endpoint_sessions_endpoint ON endpoint_sessions(endpoint_id, status);
CREATE INDEX idx_endpoint_sessions_tenant ON endpoint_sessions(tenant_id, created_at DESC);
CREATE INDEX idx_endpoint_sessions_active ON endpoint_sessions(status, last_activity_at DESC) WHERE status = 'active';
CREATE INDEX idx_endpoint_sessions_basestation ON endpoint_sessions(primary_basestation_id) WHERE primary_basestation_id IS NOT NULL;

-- Revert endpoint_keys table
ALTER TABLE endpoint_keys RENAME TO endpoint_keys_old;

-- Drop the old table (this also drops all its indexes)
DROP TABLE endpoint_keys_old;

CREATE TABLE endpoint_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Key type and version (remove 'session' type)
    key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('network', 'application', 'join')),
    key_version INTEGER NOT NULL DEFAULT 1,
    
    -- Key data
    key_value BYTEA NOT NULL,
    
    -- Key metadata
    key_id VARCHAR(64),
    algorithm VARCHAR(20) DEFAULT 'AES-128' CHECK (algorithm IN ('AES-128', 'AES-256')),
    
    -- Validity period
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    
    -- Usage tracking
    usage_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    
    -- Key rotation
    rotated_from UUID,
    rotation_reason VARCHAR(100),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255)
);

-- Recreate indexes (including partial unique constraint as index)
CREATE UNIQUE INDEX idx_endpoint_keys_unique_active ON endpoint_keys(endpoint_id, key_type) WHERE is_active = true;
CREATE INDEX idx_endpoint_keys_endpoint ON endpoint_keys(endpoint_id, key_type, is_active);
CREATE INDEX idx_endpoint_keys_tenant ON endpoint_keys(tenant_id, created_at DESC);
CREATE INDEX idx_endpoint_keys_validity ON endpoint_keys(valid_from, valid_until) WHERE is_active = true;
CREATE INDEX idx_endpoint_keys_rotation ON endpoint_keys(rotated_from) WHERE rotated_from IS NOT NULL;

-- Revert basestation_sessions if it was modified
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'basestation_sessions') THEN
        ALTER TABLE basestation_sessions RENAME TO basestation_sessions_old;

        -- Drop the old table (this also drops all its indexes)
        DROP TABLE basestation_sessions_old;

        CREATE TABLE basestation_sessions (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            basestation_id UUID NOT NULL,
            tenant_id UUID NOT NULL,
            
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

        -- Create indexes (including partial unique constraint as index)
        CREATE UNIQUE INDEX idx_basestation_sessions_unique_active ON basestation_sessions(basestation_id) WHERE status = 'active';
        CREATE INDEX idx_basestation_sessions_basestation ON basestation_sessions(basestation_id, status);
        CREATE INDEX idx_basestation_sessions_tenant ON basestation_sessions(tenant_id, created_at DESC);
        CREATE INDEX idx_basestation_sessions_active ON basestation_sessions(status, last_ping_at DESC) WHERE status = 'active';
    END IF;
END $$;
