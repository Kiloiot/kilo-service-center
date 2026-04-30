-- Migration 047 Down: Remove the proper messages table

DROP TRIGGER IF EXISTS trigger_update_messages_updated_at ON messages;
DROP FUNCTION IF EXISTS update_messages_updated_at();
DROP TABLE IF EXISTS messages CASCADE;