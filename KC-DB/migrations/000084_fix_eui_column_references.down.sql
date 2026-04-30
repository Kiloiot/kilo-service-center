-- WARNING: This rollback restores the OLD constraint names while keeping
-- the CURRENT column names (ep_eui/bs_eui). This maintains rollback symmetry
-- without breaking on non-existent columns.
-- NEVER roll back this migration in production.

-- Drop the correctly-named constraints
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS endpoints_ep_eui_length;
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS basestations_bs_eui_length;

-- Restore old constraint names (but reference current column names to avoid failure)
ALTER TABLE endpoints ADD CONSTRAINT endpoints_eui_length CHECK (length(ep_eui) = 8);
ALTER TABLE basestations ADD CONSTRAINT basestations_eui_length CHECK (length(bs_eui) = 8);

-- Restore original function with current column names (ep_eui/bs_eui)
CREATE OR REPLACE FUNCTION enforce_tenant_isolation()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF TG_TABLE_NAME = 'messages' THEN
            IF NOT EXISTS (
                SELECT 1 FROM endpoints
                WHERE ep_eui = NEW.ep_eui
                AND tenant_id = NEW.tenant_id
            ) THEN
                RAISE EXCEPTION 'Endpoint % does not belong to tenant %',
                    NEW.ep_eui, NEW.tenant_id;
            END IF;
        END IF;

        IF TG_TABLE_NAME = 'messages' AND NEW.bs_eui IS NOT NULL THEN
            IF NOT EXISTS (
                SELECT 1 FROM basestations
                WHERE bs_eui = NEW.bs_eui
                AND tenant_id = NEW.tenant_id
            ) THEN
                RAISE EXCEPTION 'Basestation % does not belong to tenant %',
                    NEW.bs_eui, NEW.tenant_id;
            END IF;
        END IF;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.tenant_id != NEW.tenant_id THEN
            RAISE EXCEPTION 'Cannot change tenant_id';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
