import { useState } from "react";

import type { OrganizationUI } from "@api-types/api";
import { useUserOrganizations } from "@hooks";

import { useFeedback } from "@contexts/feedback";
import {
  useAddOrgUser,
  useOrganizations,
  useRemoveOrgUser,
} from "@hooks/useOrganizations";
import { ORG_ROLE, PAGINATION } from "@constants/app";
import { USER_FORM } from "@constants/messages";

export function useUserMemberships(
  userId: string | undefined,
  enabled: boolean,
) {
  const { data: userOrgsData, isLoading: isOrgsLoading } = useUserOrganizations(
    userId ?? "",
    { enabled },
  );
  const memberships = userOrgsData?.memberships ?? [];
  const { data: allOrgsData } = useOrganizations(
    PAGINATION.ORG_PICKER_PAGE_SIZE,
    0,
    undefined,
    { enabled },
  );
  const availableOrgs = (allOrgsData?.organizations ?? []).filter(
    (org) => !memberships.some((m) => m.orgId === org.id),
  );
  const addOrgUser = useAddOrgUser();
  const removeOrgUser = useRemoveOrgUser();
  const [selectedOrg, setSelectedOrg] = useState<OrganizationUI | null>(null);
  const [selectedRole, setSelectedRole] = useState<string>(ORG_ROLE.MEMBER);
  const feedback = useFeedback();

  const add = async () => {
    if (!userId || !selectedOrg) return;
    try {
      await addOrgUser.mutateAsync({
        orgId: selectedOrg.id,
        data: {
          user_id: userId,
          role: selectedRole,
          is_org_admin: false,
          is_base_station_admin: false,
          is_endpoint_admin: false,
        },
      });
      setSelectedOrg(null);
      setSelectedRole(ORG_ROLE.MEMBER);
      feedback.success(USER_FORM.MSG_MEMBERSHIP_ADDED);
    } catch {
      feedback.error(USER_FORM.ERR_MEMBERSHIP_ADD_FAILED);
    }
  };

  const remove = async (orgId: string) => {
    if (!userId) return;
    try {
      await removeOrgUser.mutateAsync({ orgId, userId });
      feedback.success(USER_FORM.MSG_MEMBERSHIP_REMOVED);
    } catch {
      feedback.error(USER_FORM.ERR_MEMBERSHIP_REMOVE_FAILED);
    }
  };

  return {
    memberships,
    availableOrgs,
    isOrgsLoading,
    selectedOrg,
    setSelectedOrg,
    selectedRole,
    setSelectedRole,
    add,
    remove,
    isAddPending: addOrgUser.isPending,
    isRemovePending: removeOrgUser.isPending,
  };
}
