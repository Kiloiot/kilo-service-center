-- Drop orphaned trigger (references non-existent column packet_counter on messages table)
DROP TRIGGER IF EXISTS trg_update_packet_counter ON messages;

-- Drop orphaned function (0 triggers attached, references columns that don't match current schema)
DROP FUNCTION IF EXISTS update_endpoint_last_packet_cnt();
