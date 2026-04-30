-- Revert basestation EUI to original naming (idempotent)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'basestations' AND column_name = 'bs_eui') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_name = 'basestations'
                       AND column_name IN ('eui', 'bsEui', 'bseui')) THEN
            ALTER TABLE basestations RENAME COLUMN bs_eui TO eui;
        END IF;
    END IF;
END $$;

COMMENT ON COLUMN basestations.eui IS NULL;
