-- Revert EUI field differentiation: rename 'bs_eui' back to 'eui' in basestations and 'ep_eui' back to 'eui' in endpoints
-- Made idempotent to handle rollback conflicts with migrations 029 and 060

-- Revert basestations table: rename bs_eui back to eui (only if bs_eui exists and eui doesn't)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'basestations' AND column_name = 'bs_eui') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_name = 'basestations' AND column_name = 'eui') THEN
            ALTER TABLE basestations RENAME COLUMN bs_eui TO eui;
        END IF;
    END IF;
END $$;

-- Revert endpoints table: rename ep_eui back to eui (only if ep_eui exists and eui doesn't)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'ep_eui') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                       WHERE table_name = 'endpoints' AND column_name = 'eui') THEN
            ALTER TABLE endpoints RENAME COLUMN ep_eui TO eui;
        END IF;
    END IF;
END $$;

-- Note: messages table remains unchanged (already uses ep_eui and bs_eui snake_case)

-- Revert constraint names (conditionally based on current column names)
DO $$
BEGIN
    -- Basestations constraint
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'basestations' AND column_name = 'eui') THEN
        ALTER TABLE basestations DROP CONSTRAINT IF EXISTS unique_basestation_per_tenant;
        ALTER TABLE basestations ADD CONSTRAINT unique_basestation_per_tenant UNIQUE(eui, tenant_id);
    END IF;

    -- Endpoints constraint
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'eui') THEN
        ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS unique_endpoint_per_tenant;
        ALTER TABLE endpoints ADD CONSTRAINT unique_endpoint_per_tenant UNIQUE(eui, tenant_id);
    END IF;
END $$;

-- Remove comments (conditionally)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'basestations' AND column_name = 'eui') THEN
        COMMENT ON COLUMN basestations.eui IS NULL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'endpoints' AND column_name = 'eui') THEN
        COMMENT ON COLUMN endpoints.eui IS NULL;
    END IF;

    -- Messages table comments (always attempt to remove)
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'messages' AND column_name = 'bs_eui') THEN
        COMMENT ON COLUMN messages.bs_eui IS NULL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'messages' AND column_name = 'ep_eui') THEN
        COMMENT ON COLUMN messages.ep_eui IS NULL;
    END IF;
END $$;
