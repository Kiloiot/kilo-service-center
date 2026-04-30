-- Initial schema for KiloCenter MIOTY Server
-- PostgreSQL 16.3+

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "hstore";

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    max_endpoints INTEGER DEFAULT 1000,
    max_basestations INTEGER DEFAULT 100,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create endpoints table (MIOTY End Points)
CREATE TABLE IF NOT EXISTS endpoints (
    id BIGSERIAL PRIMARY KEY,
    eui BYTEA UNIQUE NOT NULL CHECK (length(eui) = 8),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- MIOTY specific fields
    nwk_key BYTEA CHECK (length(nwk_key) = 16),  -- Network Key
    app_key BYTEA CHECK (length(app_key) = 16),  -- Application Key
    crypto_mode SMALLINT DEFAULT 0,
    endpoint_class CHAR(1) CHECK (endpoint_class IN ('Z', 'A')) DEFAULT 'A',
    
    -- Status fields
    last_seen_at TIMESTAMP WITH TIME ZONE,
    frame_count INTEGER DEFAULT 0,
    battery_level REAL CHECK (battery_level >= 0 AND battery_level <= 100),
    
    -- Metadata
    tags HSTORE DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_endpoint_per_tenant UNIQUE(eui, tenant_id)
);

-- Create basestations table (MIOTY Base Stations)
CREATE TABLE IF NOT EXISTS basestations (
    id BIGSERIAL PRIMARY KEY,
    eui BYTEA UNIQUE NOT NULL CHECK (length(eui) = 8),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Connection info
    host VARCHAR(255),
    port INTEGER DEFAULT 8000,
    tls_enabled BOOLEAN DEFAULT true,
    
    -- Status fields
    last_seen_at TIMESTAMP WITH TIME ZONE,
    is_online BOOLEAN DEFAULT false,
    
    -- Location
    latitude DOUBLE PRECISION CHECK (latitude >= -90 AND latitude <= 90),
    longitude DOUBLE PRECISION CHECK (longitude >= -180 AND longitude <= 180),
    altitude DOUBLE PRECISION,
    
    -- Metadata
    tags HSTORE DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_basestation_per_tenant UNIQUE(eui, tenant_id)
);

-- Create messages table (partitioned by time)
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL,
    ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),  -- End Point EUI
    bs_eui BYTEA NOT NULL CHECK (length(bs_eui) = 8),  -- Base Station EUI
    tenant_id BIGINT NOT NULL,
    
    -- Message data
    payload BYTEA NOT NULL,
    frame_count INTEGER NOT NULL,
    
    -- Radio metrics
    rssi REAL NOT NULL,
    snr REAL NOT NULL,
    eq_snr REAL NOT NULL,
    frequency DOUBLE PRECISION NOT NULL,
    
    -- MIOTY specific
    uplink_mode VARCHAR(10) DEFAULT 'standard', -- standard, low_latency, repeated
    packet_counter INTEGER, -- for deduplication
    
    -- Downlink flags
    dl_open BOOLEAN DEFAULT false,
    res_exp BOOLEAN DEFAULT false,
    dl_ack BOOLEAN DEFAULT false,
    
    -- Roaming support
    is_roaming BOOLEAN DEFAULT false,
    home_network_id VARCHAR(50), -- For roaming endpoints
    roaming_agreement_id BIGINT, -- Reference to roaming agreements
    
    -- Timestamps
    received_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Archival status
    archived BOOLEAN DEFAULT false,
    archived_at TIMESTAMP WITH TIME ZONE,
    
    PRIMARY KEY (id, received_at)
) PARTITION BY RANGE (received_at);

-- Create initial partition for current month
CREATE TABLE messages_default PARTITION OF messages DEFAULT;

-- Create downlink queue table
CREATE TABLE IF NOT EXISTS downlink_queue (
    id BIGSERIAL PRIMARY KEY,
    ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Message content
    payload BYTEA NOT NULL,
    port INTEGER DEFAULT 1,
    confirmed BOOLEAN DEFAULT false,
    
    -- Scheduling
    priority INTEGER DEFAULT 5 CHECK (priority >= 0 AND priority <= 10),
    valid_until TIMESTAMP WITH TIME ZONE,
    earliest_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'scheduled', 'transmitted', 'confirmed', 'failed', 'expired')),
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- Transmission results
    transmitted_at TIMESTAMP WITH TIME ZONE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    bs_eui BYTEA,
    failure_reason TEXT,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create message archive table (for old data)
CREATE TABLE IF NOT EXISTS messages_archive (
    LIKE messages INCLUDING ALL
);

-- Create roaming agreements table
CREATE TABLE IF NOT EXISTS roaming_agreements (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    partner_network_id VARCHAR(50) NOT NULL,
    partner_name VARCHAR(255) NOT NULL,
    
    -- Agreement details
    agreement_type VARCHAR(50) DEFAULT 'bilateral', -- bilateral, unilateral_in, unilateral_out
    is_active BOOLEAN DEFAULT true,
    valid_from TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP WITH TIME ZONE,
    
    -- Technical details
    partner_endpoint TEXT,
    api_key_hash TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_roaming_agreement UNIQUE(tenant_id, partner_network_id)
);

-- Create indexes
CREATE INDEX idx_endpoints_tenant ON endpoints(tenant_id);
CREATE INDEX idx_endpoints_last_seen ON endpoints(last_seen_at DESC);
CREATE INDEX idx_endpoints_eui ON endpoints(eui);

CREATE INDEX idx_basestations_tenant ON basestations(tenant_id);
CREATE INDEX idx_basestations_online ON basestations(is_online, last_seen_at DESC);
CREATE INDEX idx_basestations_eui ON basestations(eui);

CREATE INDEX idx_messages_endpoint ON messages(ep_eui, received_at DESC);
CREATE INDEX idx_messages_basestation ON messages(bs_eui, received_at DESC);
CREATE INDEX idx_messages_tenant ON messages(tenant_id, received_at DESC);
CREATE INDEX idx_messages_roaming ON messages(is_roaming, home_network_id) WHERE is_roaming = true;
CREATE INDEX idx_messages_archived ON messages(archived, received_at) WHERE archived = false;

CREATE INDEX idx_downlink_queue_pending ON downlink_queue(ep_eui, status, priority DESC, created_at) 
    WHERE status IN ('pending', 'scheduled');
CREATE INDEX idx_downlink_queue_tenant ON downlink_queue(tenant_id, status);

CREATE INDEX idx_roaming_agreements_active ON roaming_agreements(tenant_id, is_active) WHERE is_active = true;

-- Create update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_endpoints_updated_at BEFORE UPDATE ON endpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_basestations_updated_at BEFORE UPDATE ON basestations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_downlink_queue_updated_at BEFORE UPDATE ON downlink_queue
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_roaming_agreements_updated_at BEFORE UPDATE ON roaming_agreements
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function for automatic partition management
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    partition_name text;
    start_date date;
    end_date date;
BEGIN
    start_date := date_trunc('month', CURRENT_DATE);
    end_date := start_date + interval '1 month';
    partition_name := 'messages_' || to_char(start_date, 'YYYY_MM');
    
    -- Create partition if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE tablename = partition_name
    ) THEN
        EXECUTE format('CREATE TABLE %I PARTITION OF messages 
                        FOR VALUES FROM (%L) TO (%L)', 
                        partition_name, start_date, end_date);
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create function to archive old messages
CREATE OR REPLACE FUNCTION archive_old_messages(months_to_keep integer DEFAULT 12)
RETURNS integer AS $$
DECLARE
    archived_count integer;
BEGIN
    -- Move messages older than X months to archive
    WITH moved AS (
        INSERT INTO messages_archive
        SELECT * FROM messages 
        WHERE received_at < CURRENT_TIMESTAMP - (months_to_keep || ' months')::interval
        AND archived = false
        RETURNING id
    )
    UPDATE messages SET 
        archived = true,
        archived_at = CURRENT_TIMESTAMP
    WHERE id IN (SELECT id FROM moved);
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Insert default tenant
INSERT INTO tenants (name, description) 
VALUES ('Default', 'Default tenant for development')
ON CONFLICT DO NOTHING;
