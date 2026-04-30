-- Fix MIOTY Endpoint Compliance per BSSCI v1.0.0 specification
-- This migration corrects field types and adds missing fields required by MIOTY spec

-- 1. Fix repetition field - MIOTY spec says it's Boolean "True if End Point uses DL repetition"
-- We'll add a new column and migrate data, then drop the old one
ALTER TABLE endpoints 
ADD COLUMN repetition_bool BOOLEAN DEFAULT false;

-- Copy data from old column (any value > 0 becomes true)
UPDATE endpoints SET repetition_bool = (repetition > 0);

-- Drop the old column
ALTER TABLE endpoints DROP COLUMN repetition;

-- Rename the new column
ALTER TABLE endpoints RENAME COLUMN repetition_bool TO repetition;

-- Update comment to match MIOTY spec exactly
COMMENT ON COLUMN endpoints.repetition IS 'True if End Point uses DL repetition per MIOTY BSSCI v1.0.0 Section 3.8.1';

-- 2. Add missing Attach operation fields per BSSCI v1.0.0 Section 3.6.1
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS nonce BYTEA, -- 4 Byte End Point nonce
ADD COLUMN IF NOT EXISTS sign BYTEA,  -- 4 Byte End Point signature
ADD COLUMN IF NOT EXISTS last_attach_rx_time BIGINT, -- Unix UTC time of last attach reception, ns resolution
ADD COLUMN IF NOT EXISTS last_attach_rx_duration BIGINT; -- Duration of last attach reception in ns

-- 3. Add missing radio metrics fields for endpoint per BSSCI v1.0.0
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS last_snr FLOAT, -- Last reception signal to noise ratio in dB
ADD COLUMN IF NOT EXISTS last_rssi FLOAT, -- Last reception signal strength in dBm
ADD COLUMN IF NOT EXISTS last_eq_snr FLOAT, -- Last AWGN equivalent reception SNR in dB
ADD COLUMN IF NOT EXISTS last_profile VARCHAR(10); -- MIOTY profile used for last reception (e.g., "eu1")

-- 4. Add comments for all new fields per MIOTY spec
COMMENT ON COLUMN endpoints.nonce IS '4 Byte End Point nonce from last attach per MIOTY BSSCI v1.0.0 Section 3.6.1';
COMMENT ON COLUMN endpoints.sign IS '4 Byte End Point signature from last attach per MIOTY BSSCI v1.0.0 Section 3.6.1';
COMMENT ON COLUMN endpoints.last_attach_rx_time IS 'Unix UTC time of last attach reception, 64 bit, ns resolution per MIOTY BSSCI v1.0.0 Section 3.6.1';
COMMENT ON COLUMN endpoints.last_attach_rx_duration IS 'Duration of last attach reception in ns per MIOTY BSSCI v1.0.0 Section 3.6.1';
COMMENT ON COLUMN endpoints.last_snr IS 'Last reception signal to noise ratio in dB per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_rssi IS 'Last reception signal strength in dBm per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_eq_snr IS 'Last AWGN equivalent reception SNR in dB per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_profile IS 'MIOTY profile used for last reception (e.g., "eu1") per MIOTY BSSCI v1.0.0';

-- 5. Fix attach_cnt field comment to match MIOTY spec
COMMENT ON COLUMN endpoints.attach_cnt IS 'End Point attachment counter per MIOTY BSSCI v1.0.0 Section 3.6.1';

-- 6. Add indexes for new fields that might be queried
CREATE INDEX IF NOT EXISTS idx_endpoints_last_attach_time 
ON endpoints(tenant_id, last_attach_rx_time DESC) 
WHERE last_attach_rx_time IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_endpoints_radio_quality 
ON endpoints(tenant_id, last_snr DESC, last_rssi DESC) 
WHERE last_snr IS NOT NULL OR last_rssi IS NOT NULL;

-- 7. Create/update mioty_messages table to include missing UL Data fields
-- Check if we need to add missing columns to mioty_messages
ALTER TABLE mioty_messages
ADD COLUMN IF NOT EXISTS rx_duration BIGINT, -- Duration of reception per BSSCI v1.0.0 Section 3.10.1
ADD COLUMN IF NOT EXISTS profile VARCHAR(10), -- MIOTY profile identifier per BSSCI v1.0.0 Section 3.10.1
ADD COLUMN IF NOT EXISTS subpackets JSONB; -- Subpackets object per BSSCI v1.0.0 Section 3.10.1

-- Add comments for message fields
COMMENT ON COLUMN mioty_messages.rx_duration IS 'Duration of reception, center of first to last subpacket in ns per MIOTY BSSCI v1.0.0 Section 3.10.1';
COMMENT ON COLUMN mioty_messages.profile IS 'Name of the MIOTY profile used for reception (e.g., "eu1") per MIOTY BSSCI v1.0.0 Section 3.10.1';
COMMENT ON COLUMN mioty_messages.subpackets IS 'Subpackets object with reception info per subpacket per MIOTY BSSCI v1.0.0 Section 3.10.1';