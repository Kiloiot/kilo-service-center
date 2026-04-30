-- Rollback: Restore original constraint requiring CA certificate

-- Drop the modified constraint
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS check_bssci_config;

-- Restore original constraint requiring both service_center_url and tls_ca_certificate
ALTER TABLE basestations
ADD CONSTRAINT check_bssci_config CHECK (
    connection_type != 'bssci' OR (
        service_center_url IS NOT NULL AND
        tls_ca_certificate IS NOT NULL
    )
);