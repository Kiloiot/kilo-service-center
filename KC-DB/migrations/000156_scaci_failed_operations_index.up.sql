-- Failed SCACI operations are grouped per tenant and time range for the
-- errors view; the partial index keeps that aggregation off the full log.
CREATE INDEX IF NOT EXISTS idx_scaci_op_log_tenant_failed
    ON scaci_operation_log (tenant_id, initiated_at DESC)
    WHERE state = 'failed';
