-- Migration 000104: Create blueprints table
-- Part of Blueprint feature (MIOTY Application Layer payload decoding)

CREATE TABLE blueprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_model_id UUID NOT NULL REFERENCES device_models(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,  -- Semantic version (e.g., "1.0.0")
    type_eui BYTEA NOT NULL CHECK (length(type_eui) = 8),  -- 8-byte MIOTY Type EUI (mandatory per spec)
    spec_json JSONB NOT NULL,  -- Full blueprint JSON per MIOTY Application Layer spec
    is_default BOOLEAN NOT NULL DEFAULT false,  -- Default blueprint for this model

    -- GitHub registry integration
    registry_repo VARCHAR(512),  -- e.g., "mioty-alliance/device-blueprints"
    registry_commit_sha VARCHAR(64),  -- Commit SHA if synced from registry
    registry_verified BOOLEAN NOT NULL DEFAULT false,  -- True if verified in registry
    registry_pr_url VARCHAR(512),  -- PR URL if submission pending

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_blueprints_model_version UNIQUE(device_model_id, version),
    CONSTRAINT uq_blueprints_tenant_type_eui UNIQUE(tenant_id, type_eui)
);

-- Indexes for efficient lookups
CREATE INDEX idx_blueprints_model ON blueprints(device_model_id);
CREATE INDEX idx_blueprints_tenant ON blueprints(tenant_id);
CREATE INDEX idx_blueprints_type_eui ON blueprints(type_eui);
CREATE INDEX idx_blueprints_default ON blueprints(device_model_id) WHERE is_default = true;
CREATE INDEX idx_blueprints_spec_gin ON blueprints USING GIN (spec_json);

-- Comments for documentation
COMMENT ON TABLE blueprints IS 'MIOTY device blueprints for payload decoding per Application Layer spec';
COMMENT ON COLUMN blueprints.version IS 'Semantic version of the blueprint (e.g., 1.0.0)';
COMMENT ON COLUMN blueprints.type_eui IS '8-byte MIOTY Type EUI (mandatory per MIOTY spec §2.2.4)';
COMMENT ON COLUMN blueprints.spec_json IS 'Full blueprint JSON per MIOTY Application Layer spec';
COMMENT ON COLUMN blueprints.is_default IS 'True if this is the default blueprint for the model';
COMMENT ON COLUMN blueprints.registry_verified IS 'True if blueprint is verified in community registry';
