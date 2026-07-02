-- Restore tenant ON DELETE CASCADE foreign keys that migration 000053 declared but
-- that never executed on databases seeded from a dump with a force-stamped version.
-- Scope: public tenant-scoped physical tables only. organizations/api_keys are excluded
-- (they are legacy views in public / real tables in the identity schema, and their
-- lifecycle is handled in application code, not by a tenant cascade).
-- Added NOT VALID: no scan of existing rows (tolerates pre-existing orphans) while the
-- CASCADE action still fires on future tenant deletes. Idempotent; skips non-tables.
DO $$
DECLARE
    rec  RECORD;
    rel  oid;
    kind "char";
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
        rel := to_regclass('public.' || rec.tbl);
        IF rel IS NULL THEN
            CONTINUE;
        END IF;
        SELECT relkind INTO kind FROM pg_class WHERE oid = rel;
        IF kind <> 'r' THEN
            CONTINUE; -- skip views/matviews/etc.
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = rec.tbl AND column_name = 'tenant_id'
        ) THEN
            CONTINUE;
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_constraint c
            JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
            WHERE c.conrelid = rel
              AND c.contype = 'f'
              AND c.confrelid = 'public.tenants'::regclass
              AND array_length(c.conkey, 1) = 1
              AND a.attname = 'tenant_id'
        ) THEN
            CONTINUE; -- a tenant_id FK to tenants already exists
        END IF;
        EXECUTE format(
            'ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE NOT VALID',
            rec.tbl, rec.con
        );
    END LOOP;
END $$;
