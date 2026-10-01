-- The errors center groups a tenant's warning-and-above events by time range;
-- this partial index serves that scan without touching informational rows.
CREATE INDEX IF NOT EXISTS idx_system_events_tenant_failures
    ON system_events (tenant_id, occurred_at DESC)
    WHERE severity IN ('warning', 'error', 'critical');
