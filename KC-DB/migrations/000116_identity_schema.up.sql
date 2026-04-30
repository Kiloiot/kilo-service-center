-- Move identity tables to a dedicated schema for KC-Identity service isolation.
-- Compatibility views in the public schema ensure existing queries continue to work.

CREATE SCHEMA IF NOT EXISTS identity;

ALTER TABLE public.users SET SCHEMA identity;
ALTER TABLE public.organizations SET SCHEMA identity;
ALTER TABLE public.organization_members SET SCHEMA identity;
ALTER TABLE public.api_keys SET SCHEMA identity;
ALTER TABLE public.refresh_tokens SET SCHEMA identity;

-- Compatibility views for existing queries against public schema
CREATE VIEW public.users AS SELECT * FROM identity.users;
CREATE VIEW public.organizations AS SELECT * FROM identity.organizations;
CREATE VIEW public.organization_members AS SELECT * FROM identity.organization_members;
CREATE VIEW public.api_keys AS SELECT * FROM identity.api_keys;
CREATE VIEW public.refresh_tokens AS SELECT * FROM identity.refresh_tokens;
