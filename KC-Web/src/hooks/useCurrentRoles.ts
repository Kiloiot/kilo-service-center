import type { UserRolesAPI } from "@api-types/api";
import { useQuery } from "@tanstack/react-query";

import { authApi } from "@services/api";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { TIMING_ROLES_REFRESH } from "@constants/app";
import { queryKeys } from "@config/query-keys";

/**
 * Roles of the signed-in user in the current organization, read from the
 * service and refreshed periodically so a role change applies without a new
 * sign-in.
 */
export function useCurrentRoles() {
  const { isAuthenticated } = useSession();
  const { organizationId } = useOrganization();
  return useQuery<UserRolesAPI>({
    queryKey: queryKeys.auth.roles(organizationId),
    queryFn: () => authApi.getCurrentRoles(),
    enabled: isAuthenticated,
    refetchInterval: TIMING_ROLES_REFRESH,
  });
}
