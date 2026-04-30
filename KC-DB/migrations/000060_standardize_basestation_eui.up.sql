-- Standardize basestation EUI to snake_case (bs_eui) - idempotent
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'basestations' AND column_name = 'bsEui') THEN
        ALTER TABLE basestations RENAME COLUMN "bsEui" TO bs_eui;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_name = 'basestations' AND column_name = 'bseui') THEN
        ALTER TABLE basestations RENAME COLUMN bseui TO bs_eui;
    ELSIF EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_name = 'basestations' AND column_name = 'eui') THEN
        ALTER TABLE basestations RENAME COLUMN eui TO bs_eui;
    END IF;
END $$;

COMMENT ON COLUMN basestations.bs_eui IS 'Base Station EUI-64 identifier (8 bytes) per MIOTY specification';
