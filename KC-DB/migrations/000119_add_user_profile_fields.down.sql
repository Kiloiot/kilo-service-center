-- The public.users compatibility view depends on the profile columns, so it
-- must be dropped before the columns and recreated afterwards.
DROP VIEW IF EXISTS public.users;

ALTER TABLE identity.users
  DROP COLUMN IF EXISTS first_name,
  DROP COLUMN IF EXISTS last_name,
  DROP COLUMN IF EXISTS company_name;

CREATE VIEW public.users AS SELECT * FROM identity.users;
