-- Role switches are enforced from this release on. Accounts that exist when
-- this migration runs keep the access they had: before enforcement every
-- signed-in non-admin read and wrote base stations and endpoints, so each
-- non-admin user and each active membership gains the base station and
-- endpoint manager switches. Tenant manager is not granted: managing members
-- already required organization_members.is_org_admin, which the role resolver
-- maps to tenant manager unchanged. Administrator is never granted.
--
-- The previous values are kept so the down migration restores exactly what
-- this one changed. Accounts created later start with every switch off.
--
-- SQL cannot use Go constants: 'active' must match
-- OrganizationMemberStatusActive (documented SQL migration exemption).

CREATE TABLE identity.role_grandfathered_users (
    user_id UUID PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
    was_base_station_manager BOOLEAN NOT NULL,
    was_endpoint_manager BOOLEAN NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE identity.role_grandfathered_memberships (
    org_id UUID NOT NULL,
    user_id UUID NOT NULL,
    was_base_station_admin BOOLEAN NOT NULL,
    was_endpoint_admin BOOLEAN NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, user_id),
    FOREIGN KEY (org_id, user_id) REFERENCES identity.organization_members(org_id, user_id) ON DELETE CASCADE
);

INSERT INTO identity.role_grandfathered_users (user_id, was_base_station_manager, was_endpoint_manager)
SELECT id, is_base_station_manager, is_endpoint_manager
FROM identity.users
WHERE is_admin = false
  AND (is_base_station_manager = false OR is_endpoint_manager = false);

UPDATE identity.users u
SET is_base_station_manager = true,
    is_endpoint_manager = true
FROM identity.role_grandfathered_users g
WHERE u.id = g.user_id;

INSERT INTO identity.role_grandfathered_memberships (org_id, user_id, was_base_station_admin, was_endpoint_admin)
SELECT org_id, user_id, is_base_station_admin, is_endpoint_admin
FROM identity.organization_members
WHERE status = 'active'
  AND (is_base_station_admin = false OR is_endpoint_admin = false);

UPDATE identity.organization_members m
SET is_base_station_admin = true,
    is_endpoint_admin = true
FROM identity.role_grandfathered_memberships g
WHERE m.org_id = g.org_id AND m.user_id = g.user_id;
