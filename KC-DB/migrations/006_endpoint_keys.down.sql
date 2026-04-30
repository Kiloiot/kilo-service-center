-- Remove endpoint_keys table

-- Drop trigger and function
DROP TRIGGER IF EXISTS endpoint_keys_single_active ON endpoint_keys;
DROP FUNCTION IF EXISTS ensure_single_active_key();

-- Drop table
DROP TABLE IF EXISTS endpoint_keys;
