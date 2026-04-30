-- Rollback migration 092: Remove error columns from scaci_operation_log

DROP INDEX IF EXISTS idx_scaci_op_log_error_token;
DROP INDEX IF EXISTS idx_scaci_op_log_error_code;
ALTER TABLE scaci_operation_log DROP COLUMN IF EXISTS error_token;
ALTER TABLE scaci_operation_log DROP COLUMN IF EXISTS error_code;
