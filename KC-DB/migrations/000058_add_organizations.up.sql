-- Migration 029: Organization/Tenant Bridge
-- Creates organization directory for Kilo Cloud multi-tenant integration
-- Strategy: One default organization PER existing tenant (maintains isolation)

-- Organizations table: extends tenant model with UUID-based org hierarchy
CREATE TABLE organizations (
    org_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    state VARCHAR(50) NOT NULL DEFAULT 'active', -- active, suspended, archived
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for UUID → tenant resolution (hot path for org.Resolver.LookupTenant)
CREATE UNIQUE INDEX idx_organizations_org_id ON organizations(org_id);

-- Index for tenant → default org lookup (GetDefaultOrgForTenant)
CREATE INDEX idx_organizations_tenant_id ON organizations(tenant_id);

-- Organization members: mirrors Kilo Cloud org_membership semantics
-- Ensures ownership sticks to org even if members churn
CREATE TABLE organization_members (
    org_id UUID NOT NULL REFERENCES organizations(org_id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, invited, removed
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, user_id)
);

-- Index for user membership lookups (CheckUserMembership hot path)
CREATE INDEX idx_org_members_user ON organization_members(user_id, status);

-- Index for org member listing (ListOrgMembers)
CREATE INDEX idx_org_members_org ON organization_members(org_id, status);

-- CRITICAL: Create one default organization PER existing tenant
-- This maintains existing tenant isolation boundaries (no cross-tenant data leakage)
-- Each tenant gets its own unique org UUID
INSERT INTO organizations (org_id, tenant_id, name, state)
SELECT
    gen_random_uuid() AS org_id,
    id AS tenant_id,
    CONCAT('Tenant ', id, ' Default Org') AS name,
    'active' AS state
FROM tenants;

-- Table comments for documentation
COMMENT ON TABLE organizations IS 'Organization directory for Kilo Cloud multi-tenant integration - 1 org per tenant default, extensible for multi-org scenarios';
COMMENT ON TABLE organization_members IS 'Organization membership per Kilo Cloud org_membership pattern - tracks user access and roles';

COMMENT ON COLUMN organizations.org_id IS 'Organization UUID - primary identifier used in X-Organization-ID headers and JWT claims';
COMMENT ON COLUMN organizations.tenant_id IS 'Maps organization to numeric tenant (shard key for DB operations)';
COMMENT ON COLUMN organizations.state IS 'Organization lifecycle: active=normal, suspended=temp block, archived=soft delete';

COMMENT ON COLUMN organization_members.role IS 'Member role: owner=full control, admin=manage members, member=read/write';
COMMENT ON COLUMN organization_members.status IS 'Membership status: active=current member, invited=pending, removed=revoked access';
