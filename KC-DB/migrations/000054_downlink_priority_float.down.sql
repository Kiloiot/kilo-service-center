-- Rollback migration 050: Revert priority back to INTEGER

-- Step 1: Add INTEGER column
ALTER TABLE downlink_queue ADD COLUMN priority_int INTEGER;

-- Step 2: Convert float priorities back to integer (0.0-1.0 to 0-10 scale)
UPDATE downlink_queue
SET priority_int = ROUND(priority * 10)::INTEGER;

-- Step 3: Drop the float column and rename integer one
ALTER TABLE downlink_queue DROP COLUMN priority;
ALTER TABLE downlink_queue RENAME COLUMN priority_int TO priority;

-- Step 4: Restore original constraints
ALTER TABLE downlink_queue ADD CONSTRAINT priority_range
    CHECK (priority >= 0 AND priority <= 10);

-- Step 5: Set default value
ALTER TABLE downlink_queue ALTER COLUMN priority SET DEFAULT 5;

-- Step 6: Make column NOT NULL
ALTER TABLE downlink_queue ALTER COLUMN priority SET NOT NULL;

-- Remove MIOTY comment
COMMENT ON COLUMN downlink_queue.priority IS NULL;