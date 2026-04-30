-- Add missing UL Data message fields and downlink control fields per BSSCI v1.0.0 Section 5.10.1

-- Add message/payload related fields to mioty_messages table
ALTER TABLE mioty_messages
ADD COLUMN IF NOT EXISTS user_data BYTEA,              -- userData field: n Byte End Point user data (payload)
ADD COLUMN IF NOT EXISTS format_id INTEGER DEFAULT 0,  -- format field: User data format identifier (8 bit)
ADD COLUMN IF NOT EXISTS mode VARCHAR(20),             -- mode field: MIOTY mode and variant ("ulp", "ulp-rep", "ulp-ll")
ADD COLUMN IF NOT EXISTS dl_open BOOLEAN DEFAULT false,     -- dlOpen field: True if End Point downlink window is opened
ADD COLUMN IF NOT EXISTS response_exp BOOLEAN DEFAULT false, -- responseExp field: True if End Point expects a response
ADD COLUMN IF NOT EXISTS dl_ack BOOLEAN DEFAULT false;      -- dlAck field: True if End Point acknowledges DL transmission

-- Add comments per MIOTY specification
COMMENT ON COLUMN mioty_messages.user_data IS 'End Point user data payload per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN mioty_messages.format_id IS 'User data format identifier (8 bit) per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN mioty_messages.mode IS 'MIOTY mode and variant used for reception (ulp, ulp-rep, ulp-ll) per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN mioty_messages.dl_open IS 'True if End Point downlink window is opened per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN mioty_messages.response_exp IS 'True if End Point expects a response in the DL window per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN mioty_messages.dl_ack IS 'True if End Point acknowledges reception of DL transmission per MIOTY BSSCI v1.0.0 Section 5.10.1';

-- Add indexes for downlink-related queries
CREATE INDEX IF NOT EXISTS idx_mioty_messages_dl_open 
ON mioty_messages(ep_eui, dl_open, rx_time DESC) 
WHERE dl_open = true;

CREATE INDEX IF NOT EXISTS idx_mioty_messages_response_expected 
ON mioty_messages(ep_eui, response_exp, rx_time DESC) 
WHERE response_exp = true;

-- Add downlink control fields to endpoints table for current state tracking
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS last_dl_open BOOLEAN DEFAULT false,        -- Last known downlink window state
ADD COLUMN IF NOT EXISTS last_response_exp BOOLEAN DEFAULT false,   -- Last known response expectation state  
ADD COLUMN IF NOT EXISTS last_dl_ack BOOLEAN DEFAULT false,         -- Last known DL acknowledgment state
ADD COLUMN IF NOT EXISTS last_user_data BYTEA,                      -- Last received user data (payload)
ADD COLUMN IF NOT EXISTS last_format_id INTEGER DEFAULT 0,          -- Last received format identifier
ADD COLUMN IF NOT EXISTS last_mode VARCHAR(20);                     -- Last known MIOTY mode

-- Add comments for endpoint downlink fields
COMMENT ON COLUMN endpoints.last_dl_open IS 'Last known End Point downlink window state per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_response_exp IS 'Last known End Point response expectation state per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_dl_ack IS 'Last known End Point DL acknowledgment state per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_user_data IS 'Last received user data payload from End Point per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_format_id IS 'Last received user data format identifier per MIOTY BSSCI v1.0.0';
COMMENT ON COLUMN endpoints.last_mode IS 'Last known MIOTY mode used by End Point per MIOTY BSSCI v1.0.0';