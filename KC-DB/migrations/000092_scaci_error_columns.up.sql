-- Migration 092: Add structured error columns to scaci_operation_log
-- Per SCACI v1.0.0 §3.14 - Error handling compliance
--
-- Adds:
--   error_code  - POSIX error code sent on wire (e.g., 22 for EINVAL)
--   error_token - Internal catalog token for traceability (not on wire)

ALTER TABLE scaci_operation_log
ADD COLUMN error_code INTEGER,
ADD COLUMN error_token VARCHAR(128);

-- Index for error analysis queries (filter by POSIX code)
CREATE INDEX idx_scaci_op_log_error_code
    ON scaci_operation_log(error_code)
    WHERE error_code IS NOT NULL;

-- Index for catalog token lookups
CREATE INDEX idx_scaci_op_log_error_token
    ON scaci_operation_log(error_token)
    WHERE error_token IS NOT NULL;

COMMENT ON COLUMN scaci_operation_log.error_code IS 'POSIX error code per SCACI §3.14.1';
COMMENT ON COLUMN scaci_operation_log.error_token IS 'Internal catalog token (not on wire)';
