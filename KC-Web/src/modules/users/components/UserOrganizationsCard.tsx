import React from "react";

import { Card, CardContent, Typography } from "@mui/material";

import { USER_FORM } from "@constants/messages";

import { useOrganizationAssignment, useUserMemberships } from "../hooks";
import InstallationOrganizationNote from "./InstallationOrganizationNote";
import UserMembershipsCard from "./UserMembershipsCard";

interface UserOrganizationsCardProps {
  userId: string;
  enabled: boolean;
}

const MembershipManagement: React.FC<UserOrganizationsCardProps> = ({
  userId,
  enabled,
}) => {
  const membership = useUserMemberships(userId, enabled);
  return (
    <UserMembershipsCard
      memberships={membership.memberships}
      availableOrgs={membership.availableOrgs}
      isOrgsLoading={membership.isOrgsLoading}
      isRemovePending={membership.isRemovePending}
      isAddPending={membership.isAddPending}
      selectedOrg={membership.selectedOrg}
      selectedRole={membership.selectedRole}
      onSelectedOrgChange={membership.setSelectedOrg}
      onSelectedRoleChange={membership.setSelectedRole}
      onAddMembership={membership.add}
      onRemoveMembership={membership.remove}
    />
  );
};

/**
 * A user's organizations: managed memberships in the enterprise edition, the
 * installation's organization in the community edition.
 */
const UserOrganizationsCard: React.FC<UserOrganizationsCardProps> = (props) => {
  if (useOrganizationAssignment()) return <MembershipManagement {...props} />;
  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          {USER_FORM.SECTION_ORG_MEMBERSHIPS}
        </Typography>
        <InstallationOrganizationNote />
      </CardContent>
    </Card>
  );
};

export default UserOrganizationsCard;
