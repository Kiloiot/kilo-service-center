-- Migration: Make TLS CA certificate optional for BSSCI connections
-- This allows basestations to be registered before certificates are uploaded

-- Drop the existing constraint
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS check_bssci_config;

-- Add updated constraint that doesn't require tls_ca_certificate
ALTER TABLE basestations
ADD CONSTRAINT check_bssci_config CHECK (
    connection_type != 'bssci' OR (
        service_center_url IS NOT NULL
    )
);