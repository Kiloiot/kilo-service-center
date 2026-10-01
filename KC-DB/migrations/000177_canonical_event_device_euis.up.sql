-- Station and endpoint timelines match an event's details by containment
-- against the canonical EUI form (16 uppercase hex digits, mioty.FormatEUI64),
-- which the event store writes from now on. Events stored before that may name
-- their device EUIs dashed, in lowercase or as a JSON number; this rewrites
-- data.bsEui and data.epEui of those events to the canonical form, with the
-- same rules as the store: hex with dashes or colons in any case, or an
-- unsigned 64-bit integer. Any other value is left as written.
--
-- Locking: one UPDATE that matches only the non-canonical rows. It takes the
-- ROW EXCLUSIVE table lock every UPDATE takes, which blocks no reads, inserts or
-- updates of other rows, and row locks on the rows it rewrites. The candidate
-- rows are found through the data GIN index (the ? operator); their number is
-- the preflight row "000177 system_events with non-canonical device EUIs".
--
-- Idempotent: a rewritten value is canonical and no longer matches.

CREATE FUNCTION pg_temp.canonical_event_eui(value jsonb) RETURNS text
LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE
        WHEN jsonb_typeof(value) = 'string'
             AND translate(value #>> '{}', '-:', '') ~ '^[0-9A-Fa-f]{16}$'
            THEN upper(translate(value #>> '{}', '-:', ''))
        WHEN jsonb_typeof(value) = 'number'
             AND (value #>> '{}') ~ '^[0-9]{1,20}$'
             AND (value #>> '{}')::numeric <= 18446744073709551615
            THEN lpad(upper(to_hex(CASE
                     WHEN (value #>> '{}')::numeric >= 9223372036854775808
                         THEN ((value #>> '{}')::numeric - 18446744073709551616)::bigint
                     ELSE (value #>> '{}')::numeric::bigint
                 END)), 16, '0')
    END
$$;

-- The rewrite a detail key needs: an object naming its canonical form, or an
-- empty object when the value is already canonical or is not an EUI.
CREATE FUNCTION pg_temp.canonical_event_eui_patch(data jsonb, key text) RETURNS jsonb
LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE
        WHEN pg_temp.canonical_event_eui(data -> key) IS NULL
          OR data -> key = to_jsonb(pg_temp.canonical_event_eui(data -> key))
            THEN '{}'::jsonb
        ELSE jsonb_build_object(key, pg_temp.canonical_event_eui(data -> key))
    END
$$;

UPDATE system_events
   SET data = data
           || pg_temp.canonical_event_eui_patch(data, 'bsEui')
           || pg_temp.canonical_event_eui_patch(data, 'epEui')
 WHERE (data ? 'bsEui' OR data ? 'epEui')
   AND jsonb_typeof(data) = 'object'
   AND (pg_temp.canonical_event_eui_patch(data, 'bsEui') <> '{}'::jsonb
     OR pg_temp.canonical_event_eui_patch(data, 'epEui') <> '{}'::jsonb);

DROP FUNCTION pg_temp.canonical_event_eui_patch(jsonb, text);
DROP FUNCTION pg_temp.canonical_event_eui(jsonb);
