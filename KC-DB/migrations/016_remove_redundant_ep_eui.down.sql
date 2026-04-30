-- Restore ep_eui field to endpoints table (rollback migration)
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS ep_eui BYTEA CHECK (length(ep_eui) = 8);
