-- Normalize email addresses to lowercase+trimmed for case-insensitive uniqueness.

-- Step 1: Fail fast if case/whitespace-variant duplicate emails exist
DO $$
BEGIN
    IF EXISTS (
        SELECT LOWER(TRIM(email)), COUNT(*)
        FROM identity.users
        GROUP BY LOWER(TRIM(email))
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'Cannot normalize: case-variant duplicate emails exist. Resolve manually before migrating.';
    END IF;
END $$;

-- Step 2: Normalize existing emails to lowercase+trimmed
UPDATE identity.users SET email = LOWER(TRIM(email))
WHERE email != LOWER(TRIM(email));

-- Step 3: Drop the table-level UNIQUE constraint
ALTER TABLE identity.users DROP CONSTRAINT IF EXISTS users_email_key;

-- Step 4: Add case-insensitive unique index
CREATE UNIQUE INDEX users_email_lower_unique ON identity.users (LOWER(TRIM(email)));
