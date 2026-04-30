-- Recreate the function (original from migration 028) for rollback parity.
-- Uses valid column names from current schema (packet_cnt, not packet_counter).
-- Note: Pre-migration 112 state had the function with 0 triggers attached,
-- so we only restore the function — no trigger recreation.
CREATE OR REPLACE FUNCTION update_endpoint_last_packet_cnt()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE endpoints
    SET last_packet_cnt = GREATEST(last_packet_cnt, NEW.packet_cnt),
        last_seen_at = NEW.received_at,
        updated_at = NOW()
    WHERE ep_eui = NEW.ep_eui
    AND (last_packet_cnt IS NULL OR last_packet_cnt < NEW.packet_cnt);

    UPDATE endpoint_sessions es
    SET last_packet_cnt = GREATEST(last_packet_cnt, NEW.packet_cnt),
        updated_at = NOW()
    FROM endpoints e
    WHERE e.ep_eui = NEW.ep_eui
    AND es.endpoint_id = e.id
    AND es.status = 'active'
    AND (es.last_packet_cnt IS NULL OR es.last_packet_cnt < NEW.packet_cnt);

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
