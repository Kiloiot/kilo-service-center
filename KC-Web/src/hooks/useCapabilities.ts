import type { UserRolesAPI } from "@api-types/api";

import { useSession } from "@contexts/SessionContext";
import { ROLE_REQUIREMENT, type RoleRequirement } from "@constants/app";

import { useCurrentRoles } from "./useCurrentRoles";

export interface Capabilities {
  isServerAdmin: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
  /** True when the user holds at least one role. */
  hasAnyRole: boolean;
  /** True once the roles are known; until then nothing is granted or refused. */
  rolesLoaded: boolean;
  /** Whether the roles satisfy a requirement. */
  can: (requirement: RoleRequirement) => boolean;
}

const NO_ROLES: UserRolesAPI = {
  admin: false,
  tenantManager: false,
  baseStationManager: false,
  endpointManager: false,
};

function satisfies(roles: UserRolesAPI, requirement: RoleRequirement): boolean {
  if (roles.admin) return true;
  switch (requirement) {
    case ROLE_REQUIREMENT.ANY_ROLE:
      return (
        roles.tenantManager || roles.baseStationManager || roles.endpointManager
      );
    case ROLE_REQUIREMENT.DEVICE_MANAGER:
      return roles.baseStationManager || roles.endpointManager;
    case ROLE_REQUIREMENT.BASE_STATION_MANAGER:
      return roles.baseStationManager;
    case ROLE_REQUIREMENT.ENDPOINT_MANAGER:
      return roles.endpointManager;
    case ROLE_REQUIREMENT.TENANT_MANAGER:
      return roles.tenantManager;
    default:
      return false;
  }
}

/** Capabilities the given roles grant. */
export function capabilitiesOf(
  roles: UserRolesAPI,
  rolesLoaded: boolean,
): Capabilities {
  return {
    isServerAdmin: roles.admin,
    isTenantManager: satisfies(roles, ROLE_REQUIREMENT.TENANT_MANAGER),
    isBaseStationManager: satisfies(
      roles,
      ROLE_REQUIREMENT.BASE_STATION_MANAGER,
    ),
    isEndpointManager: satisfies(roles, ROLE_REQUIREMENT.ENDPOINT_MANAGER),
    hasAnyRole: satisfies(roles, ROLE_REQUIREMENT.ANY_ROLE),
    rolesLoaded,
    can: (requirement) => satisfies(roles, requirement),
  };
}

/**
 * The signed-in user's capabilities, the single source navigation, route
 * guards and action buttons consult. The service composes the roles, so the
 * UI mirrors exactly what the backend enforces; without a signed-in user
 * there are none.
 */
export function useCapabilities(): Capabilities {
  const { isAuthenticated } = useSession();
  const rolesQuery = useCurrentRoles();

  const rolesLoaded =
    !isAuthenticated || rolesQuery.data !== undefined || rolesQuery.isError;
  return capabilitiesOf(rolesQuery.data ?? NO_ROLES, rolesLoaded);
}
