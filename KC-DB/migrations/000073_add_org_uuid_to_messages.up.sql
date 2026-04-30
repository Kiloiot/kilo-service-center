-- Migration 073: Add organization UUID to messages table for UL Data multi-tenant routing
-- Note: mioty_messages already has org_uuid from migration 071 (for control/propagate messages)
-- This migration extends org_uuid support to UL Data messages stored in messages table

-- Add org_uuid column to messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS org_uuid UUID;

-- Index for org-based message queries (tenant + org + time)
CREATE INDEX IF NOT EXISTS idx_messages_org_tenant_time
    ON messages(org_uuid, tenant_id, created_at DESC)
    WHERE org_uuid IS NOT NULL;

-- Index for endpoint + org queries (endpoint messages filtered by org)
CREATE INDEX IF NOT EXISTS idx_messages_ep_org
    ON messages(ep_eui, org_uuid, created_at DESC)
    WHERE org_uuid IS NOT NULL;

COMMENT ON COLUMN messages.org_uuid IS
    'Organization UUID resolved from endpoint owner. Enables multi-tenant roaming where uplinks from tenant A endpoints received by tenant B base stations are correctly attributed to tenant A organization.';
