-- Return the decision time to its earlier name and comment. The values carry
-- over unchanged. The guard aborts while another column already holds the
-- earlier name, which the rename would otherwise collide with.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'endpoints' AND column_name = 'status_changed_at'
    ) THEN
        RAISE EXCEPTION 'endpoints.status_changed_at already exists';
    END IF;
END $$;

ALTER TABLE endpoints RENAME COLUMN attachment_changed_at TO status_changed_at;

COMMENT ON COLUMN endpoints.status_changed_at IS
    'When the service center last changed ep_status; NULL before the first recorded change';
