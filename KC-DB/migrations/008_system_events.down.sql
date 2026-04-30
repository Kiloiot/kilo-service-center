-- Rollback system_events table

-- Drop function
DROP FUNCTION IF EXISTS expire_old_system_events();

-- Drop table
DROP TABLE IF EXISTS system_events;
