-- MIOTY Persistent Database Compliance Migration
-- Ensures full compliance with MIOTY BS-SC Interface v1.0.0 requirements

-- Ensure endpoints.ep_eui exists in snake_case (handle any previous states)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'eui') THEN
        ALTER TABLE endpoints RENAME COLUMN eui TO ep_eui;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_name = 'endpoints' AND column_name = 'epEui') THEN
        ALTER TABLE endpoints RENAME COLUMN "epEui" TO ep_eui;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_name = 'endpoints' AND column_name = 'epeui') THEN
        ALTER TABLE endpoints RENAME COLUMN epeui TO ep_eui;
    END IF;
END $$;

-- 1. Add lastPacketCnt to endpoints table (BSSCI Section 3.8.1)
-- This field is CRITICAL for session resumption and attach propagate
ALTER TABLE endpoints 
ADD COLUMN IF NOT EXISTS last_packet_cnt BIGINT DEFAULT 0;

COMMENT ON COLUMN endpoints.last_packet_cnt IS 'Last known End Point packet counter per MIOTY BSSCI 3.8.1 - CRITICAL for session resumption';

-- Add index for efficient packet counter lookups
CREATE INDEX IF NOT EXISTS idx_endpoints_last_packet_cnt
ON endpoints(ep_eui, last_packet_cnt DESC);

-- 2. Add proper queId field to downlink_queue (BSSCI Section 3.12.1)
-- MIOTY requires specific queue ID for reference
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS que_id BIGINT;

-- Generate unique queue IDs for existing records if any
UPDATE downlink_queue 
SET que_id = id 
WHERE que_id IS NULL;

-- Make que_id NOT NULL after populating
ALTER TABLE downlink_queue 
ALTER COLUMN que_id SET NOT NULL;

-- Add unique constraint on que_id
ALTER TABLE downlink_queue 
ADD CONSTRAINT unique_queue_id UNIQUE(que_id);

COMMENT ON COLUMN downlink_queue.que_id IS 'Assigned queue ID for reference per MIOTY BSSCI 3.12.1 (64-bit)';

-- Add counter-dependent fields for downlink queue (BSSCI Section 3.12.1)
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS cnt_depend BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS packet_cnt_array BIGINT[];

COMMENT ON COLUMN downlink_queue.cnt_depend IS 'True if userData is counter dependent per MIOTY BSSCI 3.12.1';
COMMENT ON COLUMN downlink_queue.packet_cnt_array IS 'End Point packet counters for which userData is valid';

-- 3. Update base station sessions table for proper session management (BSSCI Section 3.3)
-- Add MIOTY-compliant session tracking columns to existing table
ALTER TABLE basestation_sessions
ADD COLUMN IF NOT EXISTS sn_bs_uuid BYTEA CHECK (length(sn_bs_uuid) = 16),
ADD COLUMN IF NOT EXISTS sn_sc_uuid BYTEA CHECK (length(sn_sc_uuid) = 16),
ADD COLUMN IF NOT EXISTS sn_bs_op_id BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS sn_sc_op_id BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS can_resume BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Ensure only one active session per base station
CREATE UNIQUE INDEX IF NOT EXISTS idx_basestation_sessions_unique_active
ON basestation_sessions(basestation_id)
WHERE status = 'active';

-- Index for session UUID lookups
CREATE INDEX IF NOT EXISTS idx_basestation_sessions_bs_uuid
ON basestation_sessions(sn_bs_uuid);

CREATE INDEX IF NOT EXISTS idx_basestation_sessions_sc_uuid
ON basestation_sessions(sn_sc_uuid);

COMMENT ON TABLE basestation_sessions IS 'Base Station session management per MIOTY BSSCI 3.3 - CRITICAL for connection resumption';
COMMENT ON COLUMN basestation_sessions.sn_bs_uuid IS 'Base Station session UUID - must match with previous connect to resume session';
COMMENT ON COLUMN basestation_sessions.sn_sc_uuid IS 'Service Center session UUID - must match with previous connect to resume session';
COMMENT ON COLUMN basestation_sessions.sn_bs_op_id IS 'Minimum required known Base Station operation ID to resume session';
COMMENT ON COLUMN basestation_sessions.sn_sc_op_id IS 'Maximum known Service Center operation ID to resume session';

-- 4. Update endpoint_sessions to properly track packet counters
ALTER TABLE endpoint_sessions
ALTER COLUMN packet_cnt TYPE BIGINT;

-- Rename to match MIOTY spec naming
ALTER TABLE endpoint_sessions
RENAME COLUMN packet_cnt TO last_packet_cnt;

COMMENT ON COLUMN endpoint_sessions.last_packet_cnt IS 'Last known packet counter for this session per MIOTY BSSCI';

-- 5. Add fields to endpoints for complete MIOTY compliance
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS dual_chan BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS repetition BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS wide_carr_off BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS long_blk_dist BOOLEAN DEFAULT FALSE;

COMMENT ON COLUMN endpoints.dual_chan IS 'True if End Point uses dual channel mode per MIOTY BSSCI 3.8.1';
COMMENT ON COLUMN endpoints.repetition IS 'True if End Point uses DL repetition per MIOTY BSSCI 3.8.1';
COMMENT ON COLUMN endpoints.wide_carr_off IS 'Wide carrier offset flag per MIOTY BSSCI';
COMMENT ON COLUMN endpoints.long_blk_dist IS 'Long block distance flag per MIOTY BSSCI';

-- 6. Create function to update last_packet_cnt on message insertion
CREATE OR REPLACE FUNCTION update_endpoint_last_packet_cnt()
RETURNS TRIGGER AS $$
BEGIN
    -- Update the endpoint's last packet counter if this is newer
    UPDATE endpoints
    SET last_packet_cnt = GREATEST(last_packet_cnt, NEW.packet_cnt),
        last_seen_at = NEW.received_at,
        updated_at = NOW()
    WHERE ep_eui = NEW.ep_eui
    AND (last_packet_cnt IS NULL OR last_packet_cnt < NEW.packet_cnt);
    
    -- Also update active session if exists
    UPDATE endpoint_sessions es
    SET last_packet_cnt = GREATEST(last_packet_cnt, NEW.packet_cnt),
        updated_at = NOW()
    FROM endpoints e
    WHERE e.ep_eui = NEW.ep_eui
    AND es.endpoint_id = e.id
    AND es.status = 'active'
    AND (es.last_packet_cnt IS NULL OR es.last_packet_cnt < NEW.packet_cnt);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 7. Create trigger on messages table to maintain packet counter
DROP TRIGGER IF EXISTS trg_update_packet_counter ON messages;
CREATE TRIGGER trg_update_packet_counter
AFTER INSERT ON messages
FOR EACH ROW
WHEN (NEW.packet_counter IS NOT NULL)
EXECUTE FUNCTION update_endpoint_last_packet_cnt();

-- 8. Create table for tracking operation IDs per connection
CREATE TABLE IF NOT EXISTS bssci_operation_tracking (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES basestation_sessions(id) ON DELETE CASCADE,
    
    -- Operation tracking
    op_id BIGINT NOT NULL,
    command VARCHAR(50) NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    
    -- Operation state
    state VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'acknowledged', 'completed', 'failed')),
    
    -- Timing
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Response data
    response_data JSONB,
    error_message TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bssci_operation_session ON bssci_operation_tracking(session_id, op_id);
CREATE INDEX IF NOT EXISTS idx_bssci_operation_pending ON bssci_operation_tracking(session_id, state) WHERE state = 'pending';

COMMENT ON TABLE bssci_operation_tracking IS 'Tracks BSSCI operations for session resumption per MIOTY spec';

-- 9. Add downlink result tracking
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS result VARCHAR(20) CHECK (result IN ('sent', 'expired', 'invalid')),
ADD COLUMN IF NOT EXISTS tx_time BIGINT,  -- Unix UTC time of transmission
ADD COLUMN IF NOT EXISTS tx_bs_eui BYTEA CHECK (length(tx_bs_eui) = 8);

COMMENT ON COLUMN downlink_queue.result IS 'DL data result per MIOTY BSSCI 3.14.1';
COMMENT ON COLUMN downlink_queue.tx_time IS 'Unix UTC time of transmission per MIOTY BSSCI 3.14.1';
COMMENT ON COLUMN downlink_queue.tx_bs_eui IS 'EUI of transmitting Base Station per MIOTY BSSCI 3.14.1';

-- Create index for queue management
CREATE INDEX IF NOT EXISTS idx_downlink_queue_status_priority
ON downlink_queue(status, priority DESC, created_at)
WHERE status IN ('pending', 'scheduled');

-- 10. Schema comment removed — requires schema ownership which managed DB providers don't grant.