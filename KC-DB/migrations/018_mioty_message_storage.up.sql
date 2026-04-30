-- Migration: 018_mioty_message_storage
-- Description: Create tables for MIOTY protocol message storage supporting BSSCI and SCACI interfaces
-- Author: KiloCenter Development Team
-- Date: 2025-06-13

-- Create enum for message types
CREATE TYPE message_type AS ENUM (
    'ul_data',           -- Uplink data from End Point
    'dl_data',           -- Downlink data to End Point
    'attach',            -- End Point attachment
    'detach',            -- End Point detachment
    'attach_propagate',  -- Attachment propagation
    'detach_propagate',  -- Detachment propagation
    'status',            -- Base Station status
    'register',          -- SCACI register
    'deregister'         -- SCACI deregister
);

-- Create enum for message direction
CREATE TYPE message_direction AS ENUM (
    'uplink',
    'downlink'
);

-- Create enum for interface type
CREATE TYPE interface_type AS ENUM (
    'bssci',  -- Base Station Service Center Interface
    'scaci'   -- Service Center Application Center Interface
);

-- Create main MIOTY messages table (partitioned by month)
CREATE TABLE mioty_messages (
    id BIGSERIAL,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Core message identification
    message_type message_type NOT NULL,
    direction message_direction NOT NULL,
    interface_type interface_type NOT NULL,
    
    -- End Point identification
    ep_eui BYTEA CHECK (length(ep_eui) = 8),  -- End Point EUI64
    sh_addr BIGINT,                           -- Short Address
    type_eui BYTEA CHECK (length(type_eui) = 8), -- Type EUI for Blueprint
    
    -- Timing information
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rx_time BIGINT,                          -- Unix UTC nanoseconds from protocol
    rx_duration BIGINT,                      -- Reception duration in nanoseconds
    
    -- Message counters (for deduplication)
    packet_cnt BIGINT,                       -- End Point packet counter
    attach_cnt BIGINT,                       -- End Point attachment counter
    
    -- Signal quality metrics
    snr REAL,                               -- Signal to noise ratio in dB
    rssi REAL,                              -- Signal strength in dBm
    eq_snr REAL,                            -- AWGN equivalent SNR in dB
    
    -- Protocol information
    profile VARCHAR(32),                     -- MIOTY profile (e.g., "eu1")
    mode VARCHAR(32),                        -- MIOTY mode (e.g., "ulp", "ulp-rep")
    
    -- Payload data
    user_data BYTEA,                        -- End Point user data payload
    format_id SMALLINT DEFAULT 0,           -- Payload format identifier (8-bit)
    
    -- Authentication and security
    nonce BYTEA CHECK (length(nonce) = 4 OR nonce IS NULL),     -- 4-byte nonce
    signature BYTEA CHECK (length(signature) = 4 OR signature IS NULL), -- 4-byte signature
    cmac_verified BOOLEAN DEFAULT FALSE,     -- CMAC authentication status
    
    -- Downlink specific fields
    queue_id BIGINT,                        -- Queue ID for downlink messages
    priority REAL DEFAULT 0.0,              -- Message priority
    response_expected BOOLEAN DEFAULT FALSE, -- End Point response expected
    response_priority BOOLEAN DEFAULT FALSE, -- Priority response requested
    dl_window_requested BOOLEAN DEFAULT FALSE, -- DL window requested
    expiry_only BOOLEAN DEFAULT FALSE,       -- Send only if response expected
    
    -- Base Station information
    basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
    basestation_eui BYTEA CHECK (length(basestation_eui) = 8 OR basestation_eui IS NULL),
    
    -- Geographic location (from Base Station)
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    altitude REAL,
    
    -- Protocol operation tracking
    operation_id BIGINT,                    -- Protocol operation ID
    command VARCHAR(64),                    -- Protocol command
    
    -- End Point capabilities
    dual_channel BOOLEAN DEFAULT FALSE,     -- Dual channel mode
    repetition BOOLEAN DEFAULT FALSE,       -- DL repetition
    wide_carrier_offset BOOLEAN DEFAULT FALSE, -- Wide carrier offset
    long_block_distance BOOLEAN DEFAULT FALSE, -- Long DL interblock distance
    dl_open BOOLEAN DEFAULT FALSE,          -- Downlink window open
    dl_ack BOOLEAN DEFAULT FALSE,           -- DL acknowledgment
    
    -- Metadata and raw data
    raw_message JSONB,                      -- Complete raw protocol message
    metadata JSONB DEFAULT '{}',            -- Additional metadata
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT mioty_messages_pkey PRIMARY KEY (id, received_at),
    CONSTRAINT mioty_messages_tenant_isolation UNIQUE(id, tenant_id, received_at)
) PARTITION BY RANGE (received_at);

-- Create subpacket information table for detailed reception data
CREATE TABLE mioty_subpackets (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    message_received_at TIMESTAMPTZ NOT NULL,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Subpacket index
    subpacket_index INTEGER NOT NULL,
    
    -- Signal metrics per subpacket
    snr REAL,                               -- Subpacket SNR in dB
    rssi REAL,                              -- Subpacket RSSI in dBm
    frequency BIGINT,                       -- Subpacket frequency in Hz
    phase REAL,                             -- Subpacket phase in degrees ±180
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Foreign key to main message table
    FOREIGN KEY (message_id, message_received_at) REFERENCES mioty_messages(id, received_at) ON DELETE CASCADE,
    
    -- Constraints
    CONSTRAINT mioty_subpackets_unique UNIQUE(message_id, message_received_at, subpacket_index, tenant_id)
);

-- Create Base Station status messages table
CREATE TABLE mioty_basestation_status (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    basestation_id BIGINT REFERENCES basestations(id) ON DELETE CASCADE,
    basestation_eui BYTEA CHECK (length(basestation_eui) = 8),
    
    -- Status information
    status_code INTEGER NOT NULL,           -- POSIX error numbers, 0 for "ok"
    status_message TEXT,                    -- Status message
    system_time BIGINT,                     -- Unix UTC system time, ns resolution
    duty_cycle REAL,                        -- Fraction of TX time
    
    -- Geographic location
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    altitude REAL,
    
    -- System metrics
    uptime BIGINT,                          -- System uptime in seconds
    temperature REAL,                       -- System temperature in Celsius
    cpu_load REAL,                          -- CPU utilization (0.0-1.0)
    memory_load REAL,                       -- Memory utilization (0.0-1.0)
    
    -- Configuration
    config JSONB DEFAULT '{}',              -- Configuration object
    
    -- Protocol tracking
    operation_id BIGINT,                    -- Protocol operation ID
    
    -- Audit fields
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create message deduplication tracking table
CREATE TABLE mioty_message_deduplication (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Deduplication key components
    ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),
    packet_cnt BIGINT NOT NULL,
    message_hash BYTEA NOT NULL,            -- Hash of message content
    
    -- First reception information
    first_received_at TIMESTAMPTZ NOT NULL,
    first_message_id BIGINT NOT NULL,
    first_basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
    
    -- Duplicate tracking
    duplicate_count INTEGER DEFAULT 0,
    last_duplicate_at TIMESTAMPTZ,
    
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Unique constraint for deduplication
    CONSTRAINT mioty_dedup_unique UNIQUE(tenant_id, ep_eui, packet_cnt, message_hash)
);

-- Create indexes for efficient querying

-- Primary lookup indexes
CREATE INDEX idx_mioty_messages_ep_eui ON mioty_messages USING BTREE (ep_eui, received_at DESC);
CREATE INDEX idx_mioty_messages_sh_addr ON mioty_messages USING BTREE (sh_addr, received_at DESC);
CREATE INDEX idx_mioty_messages_tenant_time ON mioty_messages USING BTREE (tenant_id, received_at DESC);
CREATE INDEX idx_mioty_messages_type ON mioty_messages USING BTREE (message_type, received_at DESC);
CREATE INDEX idx_mioty_messages_basestation ON mioty_messages USING BTREE (basestation_id, received_at DESC);

-- Deduplication indexes
CREATE INDEX idx_mioty_messages_dedup ON mioty_messages USING BTREE (ep_eui, packet_cnt, received_at DESC) WHERE packet_cnt IS NOT NULL;
CREATE INDEX idx_mioty_dedup_lookup ON mioty_message_deduplication USING BTREE (ep_eui, packet_cnt);

-- Protocol operation tracking
CREATE INDEX idx_mioty_messages_operation ON mioty_messages USING BTREE (operation_id, received_at DESC) WHERE operation_id IS NOT NULL;

-- Payload format indexing
CREATE INDEX idx_mioty_messages_format ON mioty_messages USING BTREE (format_id, received_at DESC) WHERE format_id IS NOT NULL;

-- Type EUI indexing for Blueprint system
CREATE INDEX idx_mioty_messages_type_eui ON mioty_messages USING BTREE (type_eui, received_at DESC) WHERE type_eui IS NOT NULL;

-- Subpacket indexes
CREATE INDEX idx_mioty_subpackets_message ON mioty_subpackets USING BTREE (message_id, message_received_at);
CREATE INDEX idx_mioty_subpackets_tenant ON mioty_subpackets USING BTREE (tenant_id, created_at DESC);

-- Base Station status indexes
CREATE INDEX idx_mioty_bs_status_tenant ON mioty_basestation_status USING BTREE (tenant_id, received_at DESC);
CREATE INDEX idx_mioty_bs_status_basestation ON mioty_basestation_status USING BTREE (basestation_id, received_at DESC);
CREATE INDEX idx_mioty_bs_status_eui ON mioty_basestation_status USING BTREE (basestation_eui, received_at DESC);

-- Create initial monthly partition for current month
DO $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    -- Create partition for current month
    start_date := DATE_TRUNC('month', CURRENT_DATE);
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'mioty_messages_' || TO_CHAR(start_date, 'YYYY_MM');
    
    EXECUTE format('CREATE TABLE %I PARTITION OF mioty_messages FOR VALUES FROM (%L) TO (%L)', 
                   partition_name, start_date, end_date);
    
    -- Create partition for next month
    start_date := end_date;
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'mioty_messages_' || TO_CHAR(start_date, 'YYYY_MM');
    
    EXECUTE format('CREATE TABLE %I PARTITION OF mioty_messages FOR VALUES FROM (%L) TO (%L)', 
                   partition_name, start_date, end_date);
END $$;

-- Create function for automatic partition creation
CREATE OR REPLACE FUNCTION create_mioty_message_partition(partition_date DATE)
RETURNS TEXT AS $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
BEGIN
    start_date := DATE_TRUNC('month', partition_date);
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'mioty_messages_' || TO_CHAR(start_date, 'YYYY_MM');
    
    -- Check if partition already exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = partition_name) THEN
        RETURN 'Partition ' || partition_name || ' already exists';
    END IF;
    
    -- Create the partition
    EXECUTE format('CREATE TABLE %I PARTITION OF mioty_messages FOR VALUES FROM (%L) TO (%L)', 
                   partition_name, start_date, end_date);
    
    RETURN 'Created partition ' || partition_name;
END;
$$ LANGUAGE plpgsql;

-- Create trigger function for automatic updated_at timestamp
CREATE OR REPLACE FUNCTION update_mioty_message_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for timestamp updates
CREATE TRIGGER trigger_mioty_messages_updated_at
    BEFORE UPDATE ON mioty_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_mioty_message_timestamp();

CREATE TRIGGER trigger_mioty_dedup_updated_at
    BEFORE UPDATE ON mioty_message_deduplication
    FOR EACH ROW
    EXECUTE FUNCTION update_mioty_message_timestamp();

-- Add comments for documentation
COMMENT ON TABLE mioty_messages IS 'Main table for storing MIOTY protocol messages from BSSCI and SCACI interfaces';
COMMENT ON TABLE mioty_subpackets IS 'Detailed subpacket reception information for MIOTY messages';
COMMENT ON TABLE mioty_basestation_status IS 'Base Station status messages and system metrics';
COMMENT ON TABLE mioty_message_deduplication IS 'Message deduplication tracking as performed by Service Center';

COMMENT ON COLUMN mioty_messages.ep_eui IS 'End Point EUI64 identifier (8 bytes)';
COMMENT ON COLUMN mioty_messages.sh_addr IS 'Short Address assigned to End Point';
COMMENT ON COLUMN mioty_messages.type_eui IS 'Type EUI for Blueprint identification (8 bytes)';
COMMENT ON COLUMN mioty_messages.rx_time IS 'Reception timestamp from protocol (Unix UTC nanoseconds)';
COMMENT ON COLUMN mioty_messages.packet_cnt IS 'End Point packet counter for deduplication';
COMMENT ON COLUMN mioty_messages.user_data IS 'End Point payload data';
COMMENT ON COLUMN mioty_messages.format_id IS 'Payload format identifier (8-bit)';
COMMENT ON COLUMN mioty_messages.nonce IS 'Authentication nonce (4 bytes)';
COMMENT ON COLUMN mioty_messages.signature IS 'Authentication signature (4 bytes)';
COMMENT ON COLUMN mioty_messages.cmac_verified IS 'CMAC authentication verification status';
COMMENT ON COLUMN mioty_messages.raw_message IS 'Complete raw protocol message in JSON format';
