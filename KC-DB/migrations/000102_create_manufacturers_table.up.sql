-- Migration 000102: Create manufacturers table
-- Part of Blueprint feature (MIOTY Application Layer payload decoding)

CREATE TABLE manufacturers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(64) NOT NULL,  -- URL-friendly slug (e.g., "keller-sensors")
    website VARCHAR(512),
    description TEXT,
    logo_url VARCHAR(512),
    is_verified BOOLEAN NOT NULL DEFAULT false,  -- Community-verified flag
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_manufacturers_tenant_code UNIQUE(tenant_id, code)
);

-- Indexes for efficient lookups
CREATE INDEX idx_manufacturers_tenant ON manufacturers(tenant_id);
CREATE INDEX idx_manufacturers_code ON manufacturers(code);
CREATE INDEX idx_manufacturers_name ON manufacturers(name);

-- Comments for documentation
COMMENT ON TABLE manufacturers IS 'Device manufacturers for blueprint catalog';
COMMENT ON COLUMN manufacturers.code IS 'URL-friendly slug identifier unique per tenant';
COMMENT ON COLUMN manufacturers.is_verified IS 'True if manufacturer is community-verified';
