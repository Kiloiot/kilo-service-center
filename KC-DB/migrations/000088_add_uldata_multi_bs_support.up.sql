-- Migration 000088: Add multi-BS support for SCACI §3.8.1 UL Data
-- Enables multi-base-station reception capture and duplicate propagation

-- Add base_stations JSONB column for multi-BS reception array per SCACI §3.8.1
ALTER TABLE messages ADD COLUMN IF NOT EXISTS base_stations JSONB;

-- Add duplicate boolean for reused packet counter detection per SCACI §3.8.1
ALTER TABLE messages ADD COLUMN IF NOT EXISTS duplicate BOOLEAN NOT NULL DEFAULT false;

-- Create index for duplicate queries
CREATE INDEX IF NOT EXISTS idx_messages_duplicate ON messages(ep_eui, duplicate) WHERE duplicate = true;

-- Index for UPDATE lookup by deterministic key (tenant + ep_eui + packet_cnt + rx_time)
-- Time-window enforcement stays in Go (aligned with deduplicator window), not in SQL
CREATE INDEX IF NOT EXISTS idx_messages_update_lookup ON messages(tenant_id, ep_eui, packet_cnt, rx_time DESC);

-- Add comments per SCACI §3.8.1
COMMENT ON COLUMN messages.base_stations IS 'Array of BaseStationReception objects per SCACI §3.8.1 - multi-BS context';
COMMENT ON COLUMN messages.duplicate IS 'True if packet counter was reused (duplicate detection) per SCACI §3.8.1';
