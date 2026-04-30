-- Migration 071: Add Organization UUID to messages Table
-- Purpose: Enable organization-level tracking for propagate messages in multi-tenant roaming scenarios
-- Related: BSSCI §5.8-5.8.3 (Attach/Detach Propagate), roaming architecture (see docs/architecture/roaming/ at repo root)
--
-- Context:
-- When endpoints roam across tenants, attach/detach propagate messages must track:
-- - Endpoint's owner tenant (who owns the endpoint)
-- - Endpoint's owner organization (for cross-tenant roaming agreements)
-- - Session tenant (which base station received the propagate)
-- This enables proper multi-tenant isolation and roaming audit trails.
--
-- Note: Migration 047 renamed mioty_messages → messages. This migration targets the correct table.

-- Add org_uuid to messages table for organization-level tracking
ALTER TABLE messages ADD COLUMN IF NOT EXISTS org_uuid UUID;

-- Create index for efficient organization-level propagate message queries
-- This supports filtering attach/detach propagate messages by organization for audit and debugging
CREATE INDEX IF NOT EXISTS idx_messages_org_propagate
    ON messages(org_uuid, command_type, received_at DESC)
    WHERE org_uuid IS NOT NULL AND command_type IN ('attach_propagate', 'detach_propagate');

-- Create composite index for tenant + org + type queries (multi-tenant roaming analytics)
CREATE INDEX IF NOT EXISTS idx_messages_tenant_org_type
    ON messages(tenant_id, org_uuid, command_type, received_at DESC)
    WHERE org_uuid IS NOT NULL;

-- Documentation
COMMENT ON COLUMN messages.org_uuid IS 'Organization UUID for multi-tenant isolation and roaming support. For propagate messages: tracks endpoint owner organization (may differ from session tenant). Nullable for backward compatibility. See BSSCI §5.8.3 and docs/architecture/roaming/ (repo root) for multi-tenant propagation semantics.';
