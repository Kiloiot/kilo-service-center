-- Reverse identity schema migration: move tables back to public schema.

DROP VIEW IF EXISTS public.users;
DROP VIEW IF EXISTS public.organizations;
DROP VIEW IF EXISTS public.organization_members;
DROP VIEW IF EXISTS public.api_keys;
DROP VIEW IF EXISTS public.refresh_tokens;

ALTER TABLE identity.users SET SCHEMA public;
ALTER TABLE identity.organizations SET SCHEMA public;
ALTER TABLE identity.organization_members SET SCHEMA public;
ALTER TABLE identity.api_keys SET SCHEMA public;
ALTER TABLE identity.refresh_tokens SET SCHEMA public;

DROP SCHEMA IF EXISTS identity;
