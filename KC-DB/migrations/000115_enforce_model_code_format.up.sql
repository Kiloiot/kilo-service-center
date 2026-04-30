-- Backfill: normalize codes to slug form and resolve per-manufacturer collisions.
-- Three-step staged assignment ensures oldest rows claim base slugs first,
-- canonical rows are preserved when non-conflicting, and truncation collisions are handled.
-- Locks table to prevent concurrent writes during constraint drop/re-add.
DO $$
DECLARE
    rec RECORD;
    base_slug TEXT;
    final_code TEXT;
    suffix INT;
    max_base_len INT;
    row_count INT;
BEGIN
    -- Prevent concurrent writes during migration
    LOCK TABLE device_models IN ACCESS EXCLUSIVE MODE;

    CREATE TEMP TABLE _code_staging (
        id UUID PRIMARY KEY,
        manufacturer_id UUID NOT NULL,
        assigned_code VARCHAR(64) NOT NULL,
        UNIQUE (manufacturer_id, assigned_code)
    );

    -- Compute suffix bound from data (total rows per table = max possible collisions + margin)
    SELECT COUNT(*) INTO row_count FROM device_models;

    -- Identify base winners: oldest row per (manufacturer_id, normalized_slug).
    CREATE TEMP TABLE _base_winners AS
    SELECT DISTINCT ON (sub.manufacturer_id, sub.normalized)
        sub.id, sub.manufacturer_id, sub.normalized, sub.created_at
    FROM (
        SELECT id, manufacturer_id, created_at,
            COALESCE(NULLIF(
                TRIM(BOTH '-' FROM
                    regexp_replace(
                        regexp_replace(
                            REPLACE(LOWER(TRIM(code)), ' ', '-'),
                            '[^a-z0-9-]', '', 'g'),
                        '-{2,}', '-', 'g')
                ), ''), 'model') AS normalized
        FROM device_models
    ) sub
    ORDER BY sub.manufacturer_id, sub.normalized, sub.created_at, sub.id;

    -- Step 1: allocate base winners in age order against staging.
    -- Handles truncation collisions: if two different 65+ char slugs truncate to the
    -- same 64-char prefix, the older winner gets the base and the younger gets a suffix.
    FOR rec IN
        SELECT id, manufacturer_id, normalized
        FROM _base_winners
        ORDER BY created_at, id
    LOOP
        base_slug := left(rec.normalized, 64);
        IF NOT EXISTS (
            SELECT 1 FROM _code_staging
            WHERE manufacturer_id = rec.manufacturer_id
            AND assigned_code = base_slug
        ) THEN
            final_code := base_slug;
        ELSE
            suffix := 2;
            LOOP
                max_base_len := 64 - length('-' || suffix::text);
                final_code := left(base_slug, max_base_len) || '-' || suffix::text;
                EXIT WHEN NOT EXISTS (
                    SELECT 1 FROM _code_staging
                    WHERE manufacturer_id = rec.manufacturer_id
                    AND assigned_code = final_code
                );
                suffix := suffix + 1;
                IF suffix > row_count + 1 THEN
                    RAISE EXCEPTION 'model code collision exhausted for id=%, tried % suffixes', rec.id, suffix - 2;
                END IF;
            END LOOP;
        END IF;
        INSERT INTO _code_staging (id, manufacturer_id, assigned_code)
        VALUES (rec.id, rec.manufacturer_id, final_code);
    END LOOP;

    DROP TABLE _base_winners;

    -- Step 2: preserve canonical rows that don't collide with staged assignments.
    -- A canonical row is one whose current code already matches the normalized slug form
    -- and the CHECK regex. Keeps codes like "a-2" when no base winner occupies that slot.
    INSERT INTO _code_staging (id, manufacturer_id, assigned_code)
    SELECT dm.id, dm.manufacturer_id, dm.code
    FROM device_models dm
    WHERE NOT EXISTS (SELECT 1 FROM _code_staging cs WHERE cs.id = dm.id)
      AND dm.code ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?$'
      AND dm.code = COALESCE(NULLIF(
          TRIM(BOTH '-' FROM
              regexp_replace(
                  regexp_replace(
                      REPLACE(LOWER(TRIM(dm.code)), ' ', '-'),
                      '[^a-z0-9-]', '', 'g'),
                  '-{2,}', '-', 'g')
          ), ''), 'model')
      AND NOT EXISTS (
          SELECT 1 FROM _code_staging cs
          WHERE cs.manufacturer_id = dm.manufacturer_id
          AND cs.assigned_code = dm.code
      );

    -- Step 3: allocate suffixes for remaining rows against fully-seeded staging.
    FOR rec IN
        SELECT dm.id, dm.manufacturer_id, dm.code, dm.created_at,
            COALESCE(NULLIF(
                TRIM(BOTH '-' FROM
                    regexp_replace(
                        regexp_replace(
                            REPLACE(LOWER(TRIM(dm.code)), ' ', '-'),
                            '[^a-z0-9-]', '', 'g'),
                        '-{2,}', '-', 'g')
                ), ''), 'model') AS normalized
        FROM device_models dm
        WHERE NOT EXISTS (SELECT 1 FROM _code_staging cs WHERE cs.id = dm.id)
        ORDER BY dm.created_at, dm.id
    LOOP
        base_slug := left(rec.normalized, 64);
        IF NOT EXISTS (
            SELECT 1 FROM _code_staging
            WHERE manufacturer_id = rec.manufacturer_id
            AND assigned_code = base_slug
        ) THEN
            final_code := base_slug;
        ELSE
            suffix := 2;
            LOOP
                max_base_len := 64 - length('-' || suffix::text);
                final_code := left(base_slug, max_base_len) || '-' || suffix::text;
                EXIT WHEN NOT EXISTS (
                    SELECT 1 FROM _code_staging
                    WHERE manufacturer_id = rec.manufacturer_id
                    AND assigned_code = final_code
                );
                suffix := suffix + 1;
                IF suffix > row_count + 1 THEN
                    RAISE EXCEPTION 'model code collision exhausted for id=%, tried % suffixes', rec.id, suffix - 2;
                END IF;
            END LOOP;
        END IF;
        INSERT INTO _code_staging (id, manufacturer_id, assigned_code)
        VALUES (rec.id, rec.manufacturer_id, final_code);
    END LOOP;

    -- Drop unique constraint to avoid intermediate violations during batch update
    ALTER TABLE device_models DROP CONSTRAINT IF EXISTS uq_device_models_manufacturer_code;

    -- Batch update from staging
    UPDATE device_models dm
    SET code = cs.assigned_code, updated_at = NOW()
    FROM _code_staging cs
    WHERE dm.id = cs.id AND dm.code IS DISTINCT FROM cs.assigned_code;

    -- Re-add unique constraint (staging guarantees per-manufacturer uniqueness)
    ALTER TABLE device_models
        ADD CONSTRAINT uq_device_models_manufacturer_code UNIQUE(manufacturer_id, code);

    DROP TABLE _code_staging;
END $$;

-- Enforce slug format: lowercase alphanumeric + hyphens, no leading/trailing hyphens
ALTER TABLE device_models
    ADD CONSTRAINT chk_device_models_code_format
    CHECK (code ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?$');
