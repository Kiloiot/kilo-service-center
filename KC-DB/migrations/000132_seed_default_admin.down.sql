-- Remove default admin user and membership
DELETE FROM identity.organization_members WHERE user_id = '00000000-0000-0000-0000-000000000001'::uuid;
DELETE FROM identity.users WHERE id = '00000000-0000-0000-0000-000000000001'::uuid;
