import React from "react";

import { Chip } from "@mui/material";

import { useFeatureFlags } from "@contexts/FeatureFlagContext";
import { useOrganization } from "@contexts/OrganizationContext";
import { ORG_BADGE } from "@constants/messages";
import { AdminIcon } from "@theme/icons";

/**
 * Organization Badge Component
 * Displays current organization name in application shell.
 * Hidden in CE (enterprise_organizations flag controls visibility).
 */
export const OrgBadge: React.FC = () => {
  const { organizationName, organizationId } = useOrganization();
  const { isEnabled } = useFeatureFlags();

  if (!isEnabled("enterprise_organizations")) return null;

  if (!organizationId) {
    return (
      <Chip
        icon={<AdminIcon />}
        label={ORG_BADGE.NO_ORG}
        color="warning"
        size="small"
        variant="outlined"
      />
    );
  }

  return (
    <Chip
      icon={<AdminIcon />}
      label={organizationName || ORG_BADGE.LABEL_ORG}
      color="primary"
      size="small"
    />
  );
};

export default OrgBadge;
