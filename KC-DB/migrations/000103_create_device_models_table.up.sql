-- Migration 000103: Create device_models table
-- Part of Blueprint feature (MIOTY Application Layer payload decoding)

CREATE TABLE device_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manufacturer_id UUID NOT NULL REFERENCES manufacturers(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(64) NOT NULL,  -- URL-friendly slug (e.g., "eco-sensor-v2")
    type_eui BYTEA CHECK (type_eui IS NULL OR length(type_eui) = 8),  -- 8-byte MIOTY Type EUI
    description TEXT,
    datasheet_url VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_device_models_manufacturer_code UNIQUE(manufacturer_id, code)
);

-- Indexes for efficient lookups
CREATE INDEX idx_device_models_manufacturer ON device_models(manufacturer_id);
CREATE INDEX idx_device_models_tenant ON device_models(tenant_id);
CREATE INDEX idx_device_models_type_eui ON device_models(type_eui) WHERE type_eui IS NOT NULL;
CREATE INDEX idx_device_models_name ON device_models(name);

-- Comments for documentation
COMMENT ON TABLE device_models IS 'Device models within manufacturer hierarchy';
COMMENT ON COLUMN device_models.code IS 'URL-friendly slug identifier unique per manufacturer';
COMMENT ON COLUMN device_models.type_eui IS '8-byte MIOTY Type EUI per MIOTY Application Layer spec';
