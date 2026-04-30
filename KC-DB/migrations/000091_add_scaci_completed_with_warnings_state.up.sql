-- Migration: 000091_add_scaci_completed_with_warnings_state
-- Description: Add 'completed_with_warnings' to scaci_operation_log state CHECK constraint
-- Fixes: SCACI deregister cleanup and UL/DL handshake warnings use this state

-- Widen column from VARCHAR(20) to VARCHAR(25) to accommodate 'completed_with_warnings' (22 chars)
ALTER TABLE scaci_operation_log
ALTER COLUMN state TYPE VARCHAR(25);

ALTER TABLE scaci_operation_log
DROP CONSTRAINT scaci_operation_log_state_check;

ALTER TABLE scaci_operation_log
ADD CONSTRAINT scaci_operation_log_state_check
CHECK (state IN ('pending', 'acknowledged', 'completed', 'completed_with_warnings', 'failed'));
