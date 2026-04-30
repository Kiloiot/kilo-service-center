-- Rollback endpoint_sessions table

-- Drop trigger and function
DROP TRIGGER IF EXISTS endpoint_sessions_update_timestamp ON endpoint_sessions;
DROP FUNCTION IF EXISTS update_endpoint_session_activity();

-- Drop table
DROP TABLE IF EXISTS endpoint_sessions;
