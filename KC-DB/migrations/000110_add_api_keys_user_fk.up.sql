-- Add foreign key from api_keys.user_id to users(id)
-- Deferred from migration 072 because users table is created at migration 094
-- Idempotent: skips if FK already exists (mixed-environment safety)

-- Step 1: Rename ALL orphan keys to unique names before nulling user_id.
-- Prevents collisions with:
--   (a) other orphans with same (org_id, name)
--   (b) existing service-account keys with same (org_id, name)
-- Suffix format: LEFT(name, 215) || '_orphan_' || REPLACE(id::text, '-', '')
-- UUID without dashes = 32 chars; '_orphan_' = 8 chars; 215 + 8 + 32 = 255 max.
UPDATE api_keys
SET name = LEFT(name, 215) || '_orphan_' || REPLACE(id::text, '-', '')
WHERE user_id IS NOT NULL
  AND user_id NOT IN (SELECT id FROM users);

-- Step 2: Null user_id, deactivate, retype orphan keys to service_account.
UPDATE api_keys
SET user_id = NULL,
    is_active = false,
    key_type = 'service_account'
WHERE user_id IS NOT NULL
  AND user_id NOT IN (SELECT id FROM users);

-- Step 3: Fail-fast guard — abort if any duplicate (org_id, name) WHERE user_id IS NULL.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM api_keys
        WHERE user_id IS NULL
        GROUP BY org_id, name
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'migration 000110: duplicate (org_id, name) found after orphan rename — aborting before FK creation';
    END IF;
END $$;

-- Step 4: Add FK constraint (idempotent, NOT VALID for online safety).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'api_keys_user_id_fkey'
          AND conrelid = 'api_keys'::regclass
    ) THEN
        ALTER TABLE api_keys
            ADD CONSTRAINT api_keys_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
    END IF;
END $$;

-- Step 5: Validate FK (full table scan, but no lock escalation).
ALTER TABLE api_keys VALIDATE CONSTRAINT api_keys_user_id_fkey;
