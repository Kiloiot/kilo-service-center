-- Migration 047: Create proper messages table with exact MIOTY BSSCI v1.0.0 field names
-- This table stores complete UL Data messages per Section 5.10.1 specification

-- Drop old mioty_messages table if it exists (was using wrong field names)
DROP TABLE IF EXISTS mioty_messages CASCADE;

-- Drop the existing messages table (it has wrong column names)
DROP TABLE IF EXISTS messages CASCADE;

-- Create the canonical messages table with exact MIOTY field names
CREATE TABLE messages (
    -- Internal fields
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL,
    
    -- MIOTY BSSCI v1.0.0 Required Fields (Section 5.10.1)
    command_type VARCHAR(20) NOT NULL DEFAULT 'ulData',
    op_id BIGINT NOT NULL,                              -- Operation ID (64-bit per spec)
    ep_eui BIGINT NOT NULL,                             -- End Point EUI64 (stored as BIGINT for efficiency)
    bs_eui BIGINT NOT NULL,                             -- Base Station EUI64 that received this
    rx_time BIGINT NOT NULL,                            -- Unix UTC reception time in nanoseconds
    packet_cnt INTEGER NOT NULL,                        -- End Point packet counter (32-bit per spec)
    snr DOUBLE PRECISION NOT NULL,                      -- Reception signal-to-noise ratio in dB
    rssi DOUBLE PRECISION NOT NULL,                     -- Reception signal strength in dBm
    user_data BYTEA,                                    -- User data payload (may be empty)
    dl_open BOOLEAN NOT NULL DEFAULT false,             -- True if downlink window is open
    response_exp BOOLEAN NOT NULL DEFAULT false,        -- True if End Point expects DL response
    dl_ack BOOLEAN NOT NULL DEFAULT false,              -- True if End Point acknowledged DL

    -- MIOTY BSSCI v1.0.0 Optional Fields
    rx_duration BIGINT,                                 -- Reception duration in nanoseconds
    eq_snr DOUBLE PRECISION,                            -- AWGN equivalent SNR in dB
    profile VARCHAR(20),                                -- Mioty profile (e.g., "eu1")
    mode VARCHAR(20),                                   -- Mioty mode (e.g., "ulp", "ulp-rep", "ulp-ll")
    format SMALLINT DEFAULT 0,                          -- User data format ID (8-bit, using SMALLINT for compatibility)
    
    -- Subpackets data (stored as JSONB for flexibility)
    subpackets JSONB,                                   -- Contains snr[], rssi[], frequency[], phase[] arrays
    
    -- Metadata
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),     -- Server reception timestamp
    processed_at TIMESTAMPTZ,                           -- When message was processed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_format CHECK (format >= 0 AND format <= 255),  -- 8-bit format ID
    CONSTRAINT valid_packet_cnt CHECK (packet_cnt >= 0),
    CONSTRAINT valid_euis CHECK (ep_eui > 0 AND bs_eui > 0)
);

-- Create indexes for efficient querying
CREATE INDEX idx_messages_tenant_id ON messages(tenant_id);
CREATE INDEX idx_messages_ep_eui ON messages(ep_eui);
CREATE INDEX idx_messages_bs_eui ON messages(bs_eui);
CREATE INDEX idx_messages_rx_time ON messages(rx_time DESC);
CREATE INDEX idx_messages_packet_cnt ON messages(ep_eui, packet_cnt);
CREATE INDEX idx_messages_received_at ON messages(received_at DESC);

-- Index for deduplication (endpoint + packet counter within time window)
CREATE INDEX idx_messages_dedup ON messages(ep_eui, packet_cnt, rx_time);

-- Composite index for common queries
CREATE INDEX idx_messages_ep_time ON messages(ep_eui, rx_time DESC);
CREATE INDEX idx_messages_bs_time ON messages(bs_eui, rx_time DESC);

-- Add comments per MIOTY BSSCI v1.0.0 specification
COMMENT ON TABLE messages IS 'Stores UL Data messages per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN messages.op_id IS 'Operation ID - unique 64-bit incrementing value per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.ep_eui IS 'End Point EUI64 - 8-byte unique identifier per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.bs_eui IS 'Base Station EUI64 that received this message';
COMMENT ON COLUMN messages.rx_time IS 'Unix UTC reception time with nanosecond resolution (center of last subpacket) per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.rx_duration IS 'Reception duration from first to last subpacket center in nanoseconds per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.packet_cnt IS 'End Point packet counter (32-bit) per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.snr IS 'Reception signal-to-noise ratio in dB per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.rssi IS 'Reception signal strength in dBm per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.eq_snr IS 'AWGN equivalent SNR in dB (optional) per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.profile IS 'Mioty profile used (e.g., "eu1") per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.mode IS 'Mioty mode/variant (e.g., "ulp", "ulp-rep", "ulp-ll") per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.user_data IS 'User data payload from End Point (n bytes, may be empty) per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.format IS 'User data format identifier (8-bit, 0-255) per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.dl_open IS 'True if End Point downlink window is open per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.response_exp IS 'True if End Point expects a DL response (requires dl_open) per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.dl_ack IS 'True if End Point acknowledged DL transmission in last DL window per BSSCI Section 5.10.1';
COMMENT ON COLUMN messages.subpackets IS 'Subpacket info object with snr[], rssi[], frequency[], phase[] arrays per BSSCI Section 5.10.1';

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_messages_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_messages_updated_at();

-- Create partitioning for messages table by month (for scalability)
-- Note: This is commented out for initial deployment, can be enabled later
-- ALTER TABLE messages PARTITION BY RANGE (received_at);