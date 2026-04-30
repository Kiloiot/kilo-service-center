-- Remove propagation status from endpoints table
ALTER TABLE endpoints 
DROP COLUMN propagated,
DROP COLUMN propagated_at,
DROP COLUMN propagation_count;