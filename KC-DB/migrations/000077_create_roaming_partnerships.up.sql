-- Migration 000077: Roaming Partnerships Table
-- Provides source of truth for tenant roaming agreements
-- Required for AreTenantsPartners() implementation
-- DEFER-005: Production roaming database layer completion

CREATE TABLE IF NOT EXISTS roaming_partnerships (
    tenant1_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tenant2_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    established_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    notes TEXT,
    created_by VARCHAR(255),
    PRIMARY KEY (tenant1_id, tenant2_id),
    CONSTRAINT roaming_partnerships_order CHECK (tenant1_id < tenant2_id),
    CONSTRAINT roaming_partnerships_not_self CHECK (tenant1_id != tenant2_id)
);

CREATE INDEX idx_roaming_partnerships_tenant1 ON roaming_partnerships(tenant1_id);
CREATE INDEX idx_roaming_partnerships_tenant2 ON roaming_partnerships(tenant2_id);
-- Note: NOW() is NOT IMMUTABLE, cannot be used in index predicate
-- Query layer handles time comparison; index filters on "never expires" partnerships
CREATE INDEX idx_roaming_partnerships_active ON roaming_partnerships(tenant1_id, tenant2_id)
    WHERE expires_at IS NULL;

COMMENT ON TABLE roaming_partnerships IS 'Tenant roaming partnership agreements (DEFER-005)';
COMMENT ON CONSTRAINT roaming_partnerships_order ON roaming_partnerships
    IS 'Ensures tenant1_id < tenant2_id to prevent duplicate (A,B) and (B,A) rows';
COMMENT ON CONSTRAINT roaming_partnerships_not_self ON roaming_partnerships
    IS 'Prevents self-partnership (tenant cannot partner with itself)';
