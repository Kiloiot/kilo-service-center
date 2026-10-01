-- The traffic views filter uplinks by reception flags and radio profile and
-- the queue view by state, priority and queue id; these indexes back those
-- predicates on the per-tenant listings.
CREATE INDEX IF NOT EXISTS idx_messages_tenant_duplicate
    ON messages (tenant_id, rx_time DESC)
    WHERE duplicate = true;
CREATE INDEX IF NOT EXISTS idx_messages_tenant_dl_open
    ON messages (tenant_id, rx_time DESC)
    WHERE dl_open = true;
CREATE INDEX IF NOT EXISTS idx_messages_tenant_profile_mode
    ON messages (tenant_id, profile, mode, rx_time DESC);
CREATE INDEX IF NOT EXISTS idx_downlink_queue_tenant_status_priority
    ON downlink_queue (tenant_id, status, priority DESC, created_at);
