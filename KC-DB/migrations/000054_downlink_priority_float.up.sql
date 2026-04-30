-- Migration 050: Change downlink_queue priority to REAL for BSSCI compliance
-- BSSCI §3.12.1 specifies priority as "single precision floating point"

-- Step 1: Add new column with REAL type
ALTER TABLE downlink_queue ADD COLUMN priority_float REAL;

-- Step 2: Convert existing INTEGER priorities to float (0-10 scale to 0.0-1.0)
UPDATE downlink_queue
SET priority_float = CASE
    WHEN priority IS NULL THEN 0.0
    WHEN priority > 10 THEN 1.0
    WHEN priority < 0 THEN 0.0
    ELSE priority::REAL / 10.0
END;

-- Step 3: Drop the old column and rename new one
ALTER TABLE downlink_queue DROP COLUMN priority;
ALTER TABLE downlink_queue RENAME COLUMN priority_float TO priority;

-- Step 4: Add constraint for valid range
ALTER TABLE downlink_queue ADD CONSTRAINT priority_range
    CHECK (priority >= 0.0 AND priority <= 1.0);

-- Step 5: Set default value
ALTER TABLE downlink_queue ALTER COLUMN priority SET DEFAULT 0.0;

-- Step 6: Update any existing NULL values
UPDATE downlink_queue SET priority = 0.0 WHERE priority IS NULL;

-- Step 7: Make column NOT NULL
ALTER TABLE downlink_queue ALTER COLUMN priority SET NOT NULL;

-- Add comment explaining MIOTY compliance
COMMENT ON COLUMN downlink_queue.priority IS 'MIOTY BSSCI §3.12.1 priority - single precision float, higher values are prioritized';