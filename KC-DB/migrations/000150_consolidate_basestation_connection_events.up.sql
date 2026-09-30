-- Retire the legacy basestation_connection_events table (019).
--
-- Connection lifecycle is recorded through system_events by the BSSCI
-- server; nothing ever wrote to or read from basestation_connection_events.
-- Each remaining row is archived to a tenant-scoped system_events record
-- (legacy type, payload, remote address and timestamp), the tenant resolved
-- through the base station, and the table is dropped once the archive count
-- matches the source count.

DO $$
DECLARE
    source_count bigint;
    archived_count bigint;
BEGIN
    SELECT count(*) INTO source_count FROM basestation_connection_events;

    WITH archived AS (
        INSERT INTO system_events (
            tenant_id, event_type, event_category, severity,
            source_type, source_name, title, description,
            basestation_id, data, occurred_at, recorded_at, status
        )
        SELECT
            b.tenant_id,
            'basestation.connection.archived',
            'basestation',
            'info',
            'migration',
            encode(b.bs_eui, 'hex'),
            format('Archived connection event %s', e.event_type),
            format('Legacy basestation_connection_events row %s archived by migration 000150', e.id),
            e.basestation_id,
            jsonb_build_object(
                'legacyTable', 'basestation_connection_events',
                'legacyId', e.id,
                'eventType', e.event_type,
                'eventData', COALESCE(e.event_data, '{}'::jsonb),
                'remoteAddress', host(e.remote_address),
                'createdAt', e.created_at
            ),
            COALESCE(e.created_at, NOW()),
            NOW(),
            'resolved'
        FROM basestation_connection_events e
        JOIN basestations b ON b.id = e.basestation_id
        RETURNING 1
    )
    SELECT count(*) INTO archived_count FROM archived;

    IF archived_count <> source_count THEN
        RAISE EXCEPTION 'basestation_connection_events archive mismatch: % source row(s), % archived', source_count, archived_count;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_basestation_connection_events_type;
DROP INDEX IF EXISTS idx_basestation_connection_events_bs;

DROP TABLE IF EXISTS basestation_connection_events;
