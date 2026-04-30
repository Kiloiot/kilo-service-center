/**
 * Organization Mapper
 *
 * Organization and organization member API to UI transformation utilities.
 * Timestamps kept as ISO strings for consistency with other UI types.
 */

import type {
  OrganizationAPI,
  OrganizationUI,
  OrganizationUserAPI,
  OrganizationUserUI,
} from '@api-types/api';

/**
 * Maps a single OrganizationAPI to OrganizationUI.
 */
export function mapOrganization(api: OrganizationAPI): OrganizationUI {
  return {
    id: api.id,
    tenantId: api.tenant_id,
    name: api.name,
    description: api.description,
    state: api.state,
    canHaveBaseStations: api.can_have_base_stations,
    maxBaseStationCount: api.max_base_station_count,
    maxEndpointCount: api.max_endpoint_count,
    tags: api.tags,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
  };
}

/**
 * Maps an array of OrganizationAPI to OrganizationUI.
 */
export function mapOrganizationList(orgs: OrganizationAPI[]): OrganizationUI[] {
  return orgs.map(mapOrganization);
}

/**
 * Maps a single OrganizationUserAPI to OrganizationUserUI.
 */
export function mapOrgUser(api: OrganizationUserAPI): OrganizationUserUI {
  return {
    orgId: api.org_id,
    userId: api.user_id,
    email: api.email,
    role: api.role,
    status: api.status,
    isOrgAdmin: api.is_org_admin,
    isBaseStationAdmin: api.is_base_station_admin,
    isEndpointAdmin: api.is_endpoint_admin,
    createdAt: api.created_at,
    updatedAt: api.updated_at,
  };
}

/**
 * Maps an array of OrganizationUserAPI to OrganizationUserUI.
 */
export function mapOrgUserList(users: OrganizationUserAPI[]): OrganizationUserUI[] {
  return users.map(mapOrgUser);
}
