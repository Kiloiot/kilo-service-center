import { useFeatureFlags } from '@contexts/FeatureFlagContext';
import { useOrganization } from '@contexts/OrganizationContext';
import { useSession } from '@contexts/SessionContext';

export function useCapabilities() {
  const { user, isAdmin } = useSession();
  const { organizationId } = useOrganization();
  const { isEnabled } = useFeatureFlags();

  // CE defense-in-depth: derive capabilities from server admin status
  if (!isEnabled('enterprise_organizations')) {
    return {
      isServerAdmin: isAdmin,
      isOrgAdmin: false,
      isBaseStationAdmin: isAdmin,
      isEndpointAdmin: isAdmin,
    };
  }

  // ECE: membership-based capabilities
  const currentMembership = user?.memberships?.find((m) => m.orgId === organizationId);

  return {
    isServerAdmin: isAdmin,
    isOrgAdmin: currentMembership?.isOrgAdmin ?? false,
    isBaseStationAdmin: currentMembership?.isBaseStationAdmin ?? false,
    isEndpointAdmin: currentMembership?.isEndpointAdmin ?? false,
  };
}
