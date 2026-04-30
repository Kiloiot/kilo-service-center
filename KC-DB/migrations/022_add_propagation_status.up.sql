-- Add propagation status to endpoints table
ALTER TABLE endpoints 
ADD COLUMN propagated BOOLEAN DEFAULT FALSE,
ADD COLUMN propagated_at TIMESTAMP,
ADD COLUMN propagation_count INTEGER DEFAULT 0;

-- Add index for propagation status
CREATE INDEX idx_endpoints_propagated ON endpoints(propagated);