-- Event details reach the browser as written, so their keys are camelCase at
-- the source (models.EventDetailKey*). Events stored before that named some
-- details in snake_case beside camelCase ones (a station's connectivity
-- events, the attach and detach propagates, removed endpoints and stations,
-- downlink revokes); this renames those keys to the names the writers use from
-- now on. A row that already holds the new key keeps its value and drops the
-- old one.
--
-- Locking: one UPDATE that matches only the rows holding an old key. It takes
-- the ROW EXCLUSIVE table lock every UPDATE takes, which blocks no reads,
-- inserts or updates of other rows, and row locks on the rows it rewrites.
-- The candidate rows are found through the data GIN index (the ?| operator);
-- their number is the preflight row "000179 system_events with snake_case
-- detail keys".
--
-- Idempotent: a rewritten row holds no old key and no longer matches.

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
           ARRAY['basestation_name', 'basestation_id', 'endpoint_id', 'operation_id', 'operation_type',
                 'target_bs', 'target_bs_list', 'target_bs_count', 'is_online', 'connection_type', 'session_id'],
           ARRAY['baseStationName', 'baseStationId', 'endpointId', 'operationId', 'operationType',
                 'targetBs', 'targetBsList', 'targetBsCount', 'isOnline', 'connectionType', 'sessionID'])
 WHERE jsonb_typeof(data) = 'object'
   AND data ?| ARRAY['basestation_name', 'basestation_id', 'endpoint_id', 'operation_id', 'operation_type',
                     'target_bs', 'target_bs_list', 'target_bs_count', 'is_online', 'connection_type', 'session_id'];

DROP FUNCTION pg_temp.rename_event_details(jsonb, text[], text[]);
