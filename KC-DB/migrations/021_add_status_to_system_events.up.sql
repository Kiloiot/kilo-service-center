-- Add status column to system_events table for alert system
-- This fixes the alert queries that reference a non-existent status column

ALTER TABLE system_events 
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' 
CHECK (status IN ('new', 'active', 'acknowledged', 'resolved', 'ignored'));

-- Add index for efficient status queries
CREATE INDEX IF NOT EXISTS idx_system_events_status 
ON system_events(status, occurred_at DESC) 
WHERE status IN ('new', 'active');

-- Update existing records to have a status
UPDATE system_events 
SET status = 'active' 
WHERE status IS NULL;

-- Add comment
COMMENT ON COLUMN system_events.status IS 'Alert status for tracking acknowledgement and resolution';