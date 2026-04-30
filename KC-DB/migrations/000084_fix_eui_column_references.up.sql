-- Fix orphaned references to old 'eui' column names (renamed to ep_eui/bs_eui in migration 029/060).
-- Migration 013 created these constraints and triggers before the column renames.

-- 1. Drop orphaned constraints (reference non-existent 'eui' columns)
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS endpoints_eui_length;
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS basestations_eui_length;

-- 2. Add correctly named constraints on current column names for clarity/auditability.
-- (The inline CHECK from migration 001 was auto-updated during column rename,
-- but explicit named constraints improve schema introspection.)
ALTER TABLE endpoints ADD CONSTRAINT endpoints_ep_eui_length CHECK (length(ep_eui) = 8);
ALTER TABLE basestations ADD CONSTRAINT basestations_bs_eui_length CHECK (length(bs_eui) = 8);

-- 3. Update enforce_tenant_isolation() function to use correct column names
CREATE OR REPLACE FUNCTION enforce_tenant_isolation()
RETURNS trigger AS $$
BEGIN
    -- For INSERT operations
    IF TG_OP = 'INSERT' THEN
        -- Verify endpoint belongs to same tenant (for messages)
        IF TG_TABLE_NAME = 'messages' THEN
            IF NOT EXISTS (
                SELECT 1 FROM endpoints
                WHERE ep_eui = NEW.ep_eui   -- Fixed: was 'eui'
                AND tenant_id = NEW.tenant_id
            ) THEN
                RAISE EXCEPTION 'Endpoint % does not belong to tenant %',
                    NEW.ep_eui, NEW.tenant_id;
            END IF;
        END IF;

        -- Verify basestation belongs to same tenant (for messages)
        IF TG_TABLE_NAME = 'messages' AND NEW.bs_eui IS NOT NULL THEN
            IF NOT EXISTS (
                SELECT 1 FROM basestations
                WHERE bs_eui = NEW.bs_eui   -- Fixed: was 'eui'
                AND tenant_id = NEW.tenant_id
            ) THEN
                RAISE EXCEPTION 'Basestation % does not belong to tenant %',
                    NEW.bs_eui, NEW.tenant_id;
            END IF;
        END IF;
    END IF;

    -- For UPDATE operations, prevent changing tenant_id
    IF TG_OP = 'UPDATE' THEN
        IF OLD.tenant_id != NEW.tenant_id THEN
            RAISE EXCEPTION 'Cannot change tenant_id';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
