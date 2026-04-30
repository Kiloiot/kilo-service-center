-- GIN index on base_stations JSONB for multi-BS query performance
-- Enables efficient @> containment queries for secondary receiver lookups
CREATE INDEX IF NOT EXISTS idx_messages_base_stations_gin
  ON messages USING GIN (base_stations jsonb_path_ops);
