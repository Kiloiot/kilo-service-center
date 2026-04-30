-- Revert migration 091: Remove completed_with_warnings state support
--
-- WARNING: This downgrade converts completed_with_warnings → completed, losing
-- the warning semantics. Any operations that completed with partial failures
-- will appear as fully successful after rollback. This is acceptable for
-- development rollbacks but should be audited in production before applying.

-- Step 1: Convert rows (must happen first to avoid length violation)
UPDATE scaci_operation_log
SET state = 'completed'
WHERE state = 'completed_with_warnings';

-- Step 2: Drop the expanded CHECK constraint
ALTER TABLE scaci_operation_log
DROP CONSTRAINT scaci_operation_log_state_check;

-- Step 3: Shrink column back to VARCHAR(20) (now safe - no 22-char values)
ALTER TABLE scaci_operation_log
ALTER COLUMN state TYPE VARCHAR(20);

-- Step 4: Restore original four-state CHECK constraint
ALTER TABLE scaci_operation_log
ADD CONSTRAINT scaci_operation_log_state_check
CHECK (state IN ('pending', 'acknowledged', 'completed', 'failed'));
