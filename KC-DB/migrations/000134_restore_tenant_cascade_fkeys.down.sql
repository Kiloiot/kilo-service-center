-- Drop the tenant cascade foreign keys restored by the up migration.
DO $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT * FROM (VALUES
            ('basestations',                'basestations_tenant_id_fkey'),
            ('endpoints',                   'endpoints_tenant_id_fkey'),
            ('downlink_queue',              'downlink_queue_tenant_id_fkey'),
            ('roaming_agreements',          'roaming_agreements_tenant_id_fkey'),
            ('basestation_receptions',      'basestation_receptions_tenant_id_fkey'),
            ('endpoint_sessions',           'endpoint_sessions_tenant_id_fkey1'),
            ('endpoint_keys',               'endpoint_keys_tenant_id_fkey1'),
            ('basestation_sessions',        'basestation_sessions_tenant_id_fkey1'),
            ('system_events',               'system_events_tenant_id_fkey'),
            ('mioty_subpackets',            'mioty_subpackets_tenant_id_fkey'),
            ('mioty_basestation_status',    'mioty_basestation_status_tenant_id_fkey'),
            ('mioty_message_deduplication', 'mioty_message_deduplication_tenant_id_fkey'),
            ('dl_rx_status',                'dl_rx_status_tenant_id_fkey')
        ) AS t(tbl, con)
    LOOP
        IF to_regclass('public.' || rec.tbl) IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE format('ALTER TABLE public.%I DROP CONSTRAINT IF EXISTS %I', rec.tbl, rec.con);
    END LOOP;
END $$;
