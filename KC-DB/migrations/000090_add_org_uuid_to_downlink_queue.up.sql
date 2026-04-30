-- Migration 000090: Add organization_id for multi-tenant audit trail
-- SCACI §3.10: Organization context persistence
ALTER TABLE downlink_queue ADD COLUMN organization_id UUID;
CREATE INDEX idx_downlink_queue_organization_id ON downlink_queue(organization_id) WHERE organization_id IS NOT NULL;
COMMENT ON COLUMN downlink_queue.organization_id IS 'Organization UUID from SCACI session for audit trail';
