import { useFeatureFlags } from "@contexts/FeatureFlagContext";
import { FEATURE_FLAG } from "@constants/app";

/**
 * Whether users are assigned to organizations: in the enterprise edition an
 * admin assigns them; in the community edition every user belongs to the
 * installation's organization.
 */
export function useOrganizationAssignment(): boolean {
  const { isEnabled } = useFeatureFlags();
  return isEnabled(FEATURE_FLAG.ENTERPRISE_ORGANIZATIONS);
}
