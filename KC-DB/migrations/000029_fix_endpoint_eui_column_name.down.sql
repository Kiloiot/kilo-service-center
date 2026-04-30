-- Revert to original 'eui' naming (idempotent)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'ep_eui') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_name = 'endpoints'
                       AND column_name IN ('eui', 'epEui', 'epeui')) THEN
            ALTER TABLE endpoints RENAME COLUMN ep_eui TO eui;
        END IF;
    END IF;
END $$;

COMMENT ON COLUMN endpoints.eui IS NULL;
