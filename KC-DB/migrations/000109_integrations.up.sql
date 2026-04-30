-- Integration configurations for event sinks (CRUD only, aligned to proto)
-- API parity for integration management
CREATE TABLE integrations (
    id BIGSERIAL PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(org_id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL CHECK (type IN ('http', 'mqtt', 'database')),
    config JSONB NOT NULL,
    event_filter JSONB,
    delivery_format VARCHAR(50) DEFAULT 'json',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_integrations_org ON integrations(org_id, status);
CREATE INDEX idx_integrations_tenant ON integrations(tenant_id);
