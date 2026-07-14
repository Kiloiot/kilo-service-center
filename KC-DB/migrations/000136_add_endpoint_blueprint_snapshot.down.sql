-- FK restore fails if any message references a deleted blueprint (expected).
ALTER TABLE messages
    ADD CONSTRAINT messages_blueprint_version_id_fkey
        FOREIGN KEY (blueprint_version_id) REFERENCES blueprints(id);

ALTER TABLE endpoints
    DROP COLUMN IF EXISTS blueprint_snapshot;
