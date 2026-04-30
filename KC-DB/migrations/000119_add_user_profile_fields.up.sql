ALTER TABLE identity.users
  ADD COLUMN IF NOT EXISTS first_name TEXT,
  ADD COLUMN IF NOT EXISTS last_name TEXT,
  ADD COLUMN IF NOT EXISTS company_name TEXT;

CREATE OR REPLACE VIEW public.users AS SELECT * FROM identity.users;
