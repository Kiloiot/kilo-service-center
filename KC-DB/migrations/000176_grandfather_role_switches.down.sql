-- Restore the switches 000176 turned on, for the accounts it changed.

UPDATE identity.users u
SET is_base_station_manager = g.was_base_station_manager,
    is_endpoint_manager = g.was_endpoint_manager
FROM identity.role_grandfathered_users g
WHERE u.id = g.user_id;

UPDATE identity.organization_members m
SET is_base_station_admin = g.was_base_station_admin,
    is_endpoint_admin = g.was_endpoint_admin
FROM identity.role_grandfathered_memberships g
WHERE m.org_id = g.org_id AND m.user_id = g.user_id;

DROP TABLE identity.role_grandfathered_memberships;
DROP TABLE identity.role_grandfathered_users;
