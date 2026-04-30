-- Migration 093: Enforce positive tenant_id to prevent data corruption
-- Part of UL data flow fix per BSSCI §5.10.1 compliance

-- Enforce tenant_id > 0 (matches ep_eui/bs_eui pattern from migration 047)
ALTER TABLE messages ADD CONSTRAINT valid_tenant_id CHECK (tenant_id > 0);

-- Enforce owner_tenant_id > 0 when present (column added in migration 075)
ALTER TABLE messages ADD CONSTRAINT valid_owner_tenant_id
    CHECK (owner_tenant_id IS NULL OR owner_tenant_id > 0);

COMMENT ON CONSTRAINT valid_tenant_id ON messages IS
    'Prevents tenant_id=0 corruption per UL data flow fix (BSSCI §5.10.1)';
COMMENT ON CONSTRAINT valid_owner_tenant_id ON messages IS
    'Prevents owner_tenant_id=0 corruption per roaming fix';
