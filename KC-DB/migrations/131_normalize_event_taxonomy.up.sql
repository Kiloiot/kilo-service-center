-- Normalize event type names
UPDATE system_events SET event_type = 'basestation.registered' WHERE event_type = 'registered';
UPDATE system_events SET event_type = 'detach_propagate_completed' WHERE event_type = 'detach_propagate_complete';

-- Remove legacy noise events
DELETE FROM system_events WHERE event_type IN ('uldata_completed', 'ul_data_transmit');

-- Fix invalid categories
UPDATE system_events SET event_category = 'message' WHERE event_category = 'downlink';
UPDATE system_events SET event_category = 'error' WHERE event_category = 'scaci' AND event_type LIKE '%error%';

-- Backfill endpoint_id + source_name from data->>'epEui'
UPDATE system_events se SET endpoint_id = e.id, source_name = COALESCE(se.source_name, se.data->>'epEui')
FROM endpoints e
WHERE se.endpoint_id IS NULL
  AND se.data->>'epEui' IS NOT NULL
  AND LOWER(COALESCE(se.data->>'epEui', se.source_name)) = LOWER(encode(e.ep_eui, 'hex'));

-- Backfill basestation_id + source_name from data->>'bsEui'
UPDATE system_events se SET basestation_id = b.id, source_name = COALESCE(se.source_name, se.data->>'bsEui')
FROM basestations b
WHERE se.basestation_id IS NULL
  AND se.data->>'bsEui' IS NOT NULL
  AND LOWER(COALESCE(se.data->>'bsEui', se.source_name)) = LOWER(encode(b.bs_eui, 'hex'));
