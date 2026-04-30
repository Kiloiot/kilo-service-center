-- Create basestation_receptions table
-- Tracks individual message receptions from each basestation that heard the message

CREATE TABLE basestation_receptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id BIGINT NOT NULL, -- Reference to messages.id (no FK constraint due to partitioning)
    basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Reception metadata
    rssi REAL NOT NULL,
    snr REAL NOT NULL,
    eq_snr REAL,
    frequency BIGINT NOT NULL,
    channel INTEGER,
    
    -- Timing information
    received_at TIMESTAMPTZ NOT NULL,
    time_on_air_ms INTEGER,
    
    -- Base Station location at time of reception (for mobile base stations)
    basestation_latitude DOUBLE PRECISION,
    basestation_longitude DOUBLE PRECISION,
    basestation_altitude REAL,
    
    -- Additional metadata
    antenna_id INTEGER DEFAULT 0,
    fine_timestamp BIGINT, -- Fine timestamp in nanoseconds
    context BYTEA, -- Base Station-specific context data
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT basestation_receptions_unique_per_message UNIQUE(message_id, basestation_id)
);

-- Indexes for query performance
CREATE INDEX idx_basestation_receptions_message ON basestation_receptions(message_id);
CREATE INDEX idx_basestation_receptions_basestation ON basestation_receptions(basestation_id, received_at DESC);
CREATE INDEX idx_basestation_receptions_tenant ON basestation_receptions(tenant_id, received_at DESC);
CREATE INDEX idx_basestation_receptions_rssi ON basestation_receptions(tenant_id, rssi) WHERE rssi > -120;

-- Add comments
COMMENT ON TABLE basestation_receptions IS 'Individual base station reception reports for messages';
COMMENT ON COLUMN basestation_receptions.eq_snr IS 'Equivalent SNR for MIOTY modulation';
COMMENT ON COLUMN basestation_receptions.fine_timestamp IS 'High-precision timestamp in nanoseconds';
COMMENT ON COLUMN basestation_receptions.context IS 'Base Station-specific context data (e.g., hardware info)';
COMMENT ON COLUMN basestation_receptions.antenna_id IS 'Antenna identifier for multi-antenna base stations';
