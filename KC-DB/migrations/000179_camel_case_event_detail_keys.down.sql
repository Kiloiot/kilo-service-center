-- Returns the renamed event detail keys to the snake_case the previous release
-- reads. baseStationName stays on certificate events, whose writer always used
-- it, and sessionID stays on every event but a station's connectivity events,
-- the only ones that wrote session_id; SCACI events read sessionID.

CREATE FUNCTION pg_temp.rename_event_details(data jsonb, old_keys text[], new_keys text[]) RETURNS jsonb
LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    renamed jsonb := data;
BEGIN
    FOR i IN 1 .. array_length(old_keys, 1) LOOP
        IF renamed ? old_keys[i] THEN
            IF NOT renamed ? new_keys[i] THEN
                renamed := renamed || jsonb_build_object(new_keys[i], renamed -> old_keys[i]);
            END IF;
            renamed := renamed - old_keys[i];
        END IF;
    END LOOP;
    RETURN renamed;
END
$$;

UPDATE system_events
   SET data = pg_temp.rename_event_details(data,
           ARRAY['baseStationId', 'endpointId', 'operationId', 'operationType',
                 'targetBs', 'targetBsList', 'targetBsCount', 'isOnline', 'connectionType'],
           ARRAY['basestation_id', 'endpoint_id', 'operation_id', 'operation_type',
                 'target_bs', 'target_bs_list', 'target_bs_count', 'is_online', 'connection_type'])
 WHERE jsonb_typeof(data) = 'object'
   AND data ?| ARRAY['baseStationId', 'endpointId', 'operationId', 'operationType',
                     'targetBs', 'targetBsList', 'targetBsCount', 'isOnline', 'connectionType'];

UPDATE system_events
   SET data = pg_temp.rename_event_details(data, ARRAY['baseStationName'], ARRAY['basestation_name'])
 WHERE jsonb_typeof(data) = 'object'
   AND data ? 'baseStationName'
   AND event_type <> 'certificate.generated';

UPDATE system_events
   SET data = pg_temp.rename_event_details(data, ARRAY['sessionID'], ARRAY['session_id'])
 WHERE jsonb_typeof(data) = 'object'
   AND data ? 'sessionID'
   AND event_type IN ('basestation_online', 'basestation_offline');

DROP FUNCTION pg_temp.rename_event_details(jsonb, text[], text[]);
