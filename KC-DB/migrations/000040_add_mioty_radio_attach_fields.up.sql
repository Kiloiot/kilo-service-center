-- Add missing MIOTY radio metrics and attach fields per BSSCI v1.0.0 specification
-- These fields complete the MIOTY endpoint implementation

-- Add radio metrics fields per BSSCI v1.0.0 Section 5.10.1 (UL Data)
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS last_snr DOUBLE PRECISION,     -- snr field: Reception signal to noise ratio (dB)
ADD COLUMN IF NOT EXISTS last_rssi DOUBLE PRECISION,    -- rssi field: Reception signal strength indicator (dBm)  
ADD COLUMN IF NOT EXISTS last_eq_snr DOUBLE PRECISION,  -- eqSnr field: AWGN equivalent reception SNR (dB)
ADD COLUMN IF NOT EXISTS last_profile VARCHAR(20);      -- profile field: MIOTY profile used for reception (e.g., "eu1")

-- Add attach operation fields per BSSCI v1.0.0 Section 5.6.1 (Attach)
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS attach_cnt INTEGER DEFAULT 0,           -- attachCnt field: End Point attachment counter
ADD COLUMN IF NOT EXISTS nonce BYTEA,                           -- nonce field: 4 Byte End Point nonce  
ADD COLUMN IF NOT EXISTS sign BYTEA,                            -- sign field: 4 Byte End Point signature
ADD COLUMN IF NOT EXISTS last_attach_rx_time BIGINT,            -- rxTime from attach operation (Unix UTC ns)
ADD COLUMN IF NOT EXISTS last_attach_rx_duration BIGINT;        -- rxDuration from attach operation (ns)

-- Add comments per MIOTY specification
COMMENT ON COLUMN endpoints.last_snr IS 'Reception signal to noise ratio in dB per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN endpoints.last_rssi IS 'Reception signal strength indicator in dBm per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN endpoints.last_eq_snr IS 'AWGN equivalent reception SNR in dB per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN endpoints.last_profile IS 'MIOTY profile used for reception (e.g., "eu1") per MIOTY BSSCI v1.0.0 Section 5.10.1';

COMMENT ON COLUMN endpoints.attach_cnt IS 'End Point attachment counter per MIOTY BSSCI v1.0.0 Section 5.6.1';
COMMENT ON COLUMN endpoints.nonce IS '4 Byte End Point nonce from attach operation per MIOTY BSSCI v1.0.0 Section 5.6.1';
COMMENT ON COLUMN endpoints.sign IS '4 Byte End Point signature from attach operation per MIOTY BSSCI v1.0.0 Section 5.6.1';
COMMENT ON COLUMN endpoints.last_attach_rx_time IS 'Unix UTC time of last attach reception in ns per MIOTY BSSCI v1.0.0 Section 5.6.1';
COMMENT ON COLUMN endpoints.last_attach_rx_duration IS 'Duration of last attach reception in ns per MIOTY BSSCI v1.0.0 Section 5.6.1';

-- Add indexes for radio metrics queries
CREATE INDEX IF NOT EXISTS idx_endpoints_last_snr 
ON endpoints(tenant_id, last_snr DESC) 
WHERE last_snr IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_endpoints_last_rssi 
ON endpoints(tenant_id, last_rssi DESC) 
WHERE last_rssi IS NOT NULL;

-- Add index for attach operations
CREATE INDEX IF NOT EXISTS idx_endpoints_attach_cnt 
ON endpoints(tenant_id, attach_cnt DESC) 
WHERE attach_cnt > 0;