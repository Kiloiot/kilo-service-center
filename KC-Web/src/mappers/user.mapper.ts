/**
 * User Mapper
 *
 * System user API to UI transformation utilities.
 * Timestamps kept as ISO strings for consistency with other UI types.
 */

import type { SystemUserAPI, SystemUserUI } from "@api-types/api";

/**
 * Maps a single SystemUserAPI to SystemUserUI.
 */
export function mapUser(api: SystemUserAPI): SystemUserUI {
  return {
    id: api.id,
    email: api.email,
    isAdmin: api.is_admin,
    isActive: api.is_active,
    isTenantManager: api.is_tenant_manager,
    isBaseStationManager: api.is_base_station_manager,
    isEndpointManager: api.is_endpoint_manager,
    note: api.note,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
  };
}

/**
 * Maps an array of SystemUserAPI to SystemUserUI.
 */
export function mapUserList(users: SystemUserAPI[]): SystemUserUI[] {
  return users.map(mapUser);
}
