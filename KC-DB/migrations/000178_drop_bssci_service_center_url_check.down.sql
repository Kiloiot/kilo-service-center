-- Restores check_bssci_config only when every BSSCI base station has a Service
-- Center URL. A station stored with a NULL URL was registered while the server
-- knew no URL a station can reach, so there is no value to fill in; the
-- constraint then stays dropped. Releases before 000178 store the URL or an
-- empty string, never NULL, so they run unchanged without it.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM basestations WHERE connection_type = 'bssci' AND service_center_url IS NULL) THEN
        RAISE NOTICE 'check_bssci_config not restored: BSSCI base stations without a Service Center URL exist';
        RETURN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint
                   WHERE conrelid = 'basestations'::regclass AND conname = 'check_bssci_config') THEN
        ALTER TABLE basestations ADD CONSTRAINT check_bssci_config CHECK (
            connection_type <> 'bssci' OR service_center_url IS NOT NULL
        );
    END IF;
END
$$;
