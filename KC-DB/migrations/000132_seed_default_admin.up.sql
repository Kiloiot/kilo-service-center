-- Seed default admin user for fresh installations.
-- Default credentials: admin@kilocenter.local / admin123!
-- WARNING: Change the password or remove this user before production use.
--
-- This migration is idempotent — it skips if a user with this email already exists.

INSERT INTO identity.users (
    id, email, email_verified, password_hash,
    is_admin, is_active, is_tenant_manager, is_base_station_manager, is_endpoint_manager,
    first_name, last_name, note
)
SELECT
    '00000000-0000-0000-0000-000000000001'::uuid,
    'admin@kilocenter.local',
    true,
    '$pbkdf2-sha512$v=1$i=210000$a2lsb2NlbnRlci1zZWVkIQ$BE/4zDpqakF78UfY5YgGMpnI8OOn8Z9/HU8QYh0mdcP9bp4MIxqoFnOicDAohXBjCR7rIGsylTIClj8lPr9ncg',
    true,   -- is_admin
    true,   -- is_active
    true,   -- is_tenant_manager
    true,   -- is_base_station_manager
    true,   -- is_endpoint_manager
    'Admin',
    'KiloCenter',
    'Default admin account — remove or change password before production use'
WHERE NOT EXISTS (
    SELECT 1 FROM identity.users WHERE email = 'admin@kilocenter.local'
);

-- Add the default admin to the default organization (tenant 1) if not already a member
INSERT INTO identity.organization_members (
    org_id, user_id, role, status,
    is_org_admin, is_base_station_admin, is_endpoint_admin
)
SELECT
    o.org_id,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'owner',
    'active',
    true,
    true,
    true
FROM identity.organizations o
WHERE o.tenant_id = 1
  AND NOT EXISTS (
    SELECT 1 FROM identity.organization_members
    WHERE user_id = '00000000-0000-0000-0000-000000000001'::uuid
      AND org_id = o.org_id
);
