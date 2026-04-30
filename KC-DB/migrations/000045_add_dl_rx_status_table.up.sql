-- Create DL RX Status table per MIOTY BSSCI v1.0.0 Section 5.15
-- This table stores downlink reception quality reports from endpoints

CREATE TABLE IF NOT EXISTS dl_rx_status (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),
    bs_eui BYTEA NOT NULL CHECK (length(bs_eui) = 8),
    rx_time BIGINT NOT NULL, -- Unix UTC time of reception, 64 bit, ns resolution
    packet_cnt BIGINT NOT NULL, -- End Point packet counter
    dl_rx_snr FLOAT, -- End Point DL reception signal to noise ratio in dB
    dl_rx_rssi FLOAT, -- End Point DL reception signal strength in dBm
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add comments for MIOTY compliance documentation
COMMENT ON TABLE dl_rx_status IS 'Stores DL RX status reports from endpoints per MIOTY BSSCI v1.0.0 Section 5.15';
COMMENT ON COLUMN dl_rx_status.tenant_id IS 'Multi-tenant support';
COMMENT ON COLUMN dl_rx_status.ep_eui IS 'End Point EUI64 (8 bytes)';
COMMENT ON COLUMN dl_rx_status.bs_eui IS 'Base Station EUI64 (8 bytes) that received the report';
COMMENT ON COLUMN dl_rx_status.rx_time IS 'Unix UTC time of reception, center of last subpacket, 64 bit, ns resolution per MIOTY BSSCI 5.15.1';
COMMENT ON COLUMN dl_rx_status.packet_cnt IS 'End Point packet counter per MIOTY BSSCI 5.15.1';
COMMENT ON COLUMN dl_rx_status.dl_rx_snr IS 'End Point DL reception signal to noise ratio in dB per MIOTY BSSCI 5.15.1';
COMMENT ON COLUMN dl_rx_status.dl_rx_rssi IS 'End Point DL reception signal strength in dBm per MIOTY BSSCI 5.15.1';

-- Add indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_dl_rx_status_endpoint 
ON dl_rx_status(tenant_id, ep_eui, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_dl_rx_status_basestation 
ON dl_rx_status(tenant_id, bs_eui, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_dl_rx_status_time 
ON dl_rx_status(tenant_id, rx_time DESC);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_dl_rx_status_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_dl_rx_status_updated_at
    BEFORE UPDATE ON dl_rx_status
    FOR EACH ROW
    EXECUTE FUNCTION update_dl_rx_status_updated_at();