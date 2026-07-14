ALTER TABLE endpoints
    ADD COLUMN IF NOT EXISTS blueprint_snapshot JSONB;

-- FK dropped: a snapshot can outlive its deletable blueprint; a live FK would fail the uplink INSERT and lose the message.
ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_blueprint_version_id_fkey;

COMMENT ON COLUMN endpoints.blueprint_snapshot IS 'Self-contained decode snapshot (spec + provenance); NULL = follow catalog default';
COMMENT ON COLUMN messages.blueprint_version_id IS 'Denormalized source blueprint id (not an FK)';
