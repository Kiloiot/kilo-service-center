-- Standardize endpoint EUI to snake_case (ep_eui) - idempotent
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'epeui') THEN
        ALTER TABLE endpoints RENAME COLUMN epeui TO ep_eui;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_name = 'endpoints' AND column_name = 'epEui') THEN
        ALTER TABLE endpoints RENAME COLUMN "epEui" TO ep_eui;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_name = 'endpoints' AND column_name = 'eui') THEN
        ALTER TABLE endpoints RENAME COLUMN eui TO ep_eui;
    END IF;
END $$;

COMMENT ON COLUMN endpoints.ep_eui IS 'Endpoint EUI-64 identifier (8 bytes) per MIOTY specification';
