ALTER TABLE identity.users
  DROP COLUMN IF EXISTS first_name,
  DROP COLUMN IF EXISTS last_name,
  DROP COLUMN IF EXISTS company_name;

CREATE OR REPLACE VIEW public.users AS SELECT * FROM identity.users;
