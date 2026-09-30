/**
 * User management: System Users and, in the enterprise edition, Organization
 * Users (or the OrganizationRequired prompt) on tabs selected by the URL.
 *
 * Admin-only page with runtime guard for deep link protection.
 */

import React, { useId, useState } from "react";
import { Navigate, useSearchParams } from "react-router-dom";

import { Box, Button, Typography } from "@mui/material";

import OrganizationRequired from "@components/common/OrganizationRequired";
import { TabBar, type TabBarItem } from "@components/common/TabBar";
import { TabPanel } from "@components/common/TabPanel";
import { useFeatureFlags } from "@contexts/FeatureFlagContext";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import {
  FEATURE_FLAG,
  ROUTES,
  USERS_VIEW,
  USERS_VIEW_QUERY_PARAM,
  type UsersView,
} from "@constants/app";
import { USERS_AND_ROLES, USERS_PAGE } from "@constants/messages";
import { AddIcon } from "@theme/icons";

import OrganizationUsers from "./OrganizationUsers";
import Users from "./Users";

const SYSTEM_TAB: TabBarItem<UsersView> = {
  value: USERS_VIEW.SYSTEM,
  label: USERS_AND_ROLES.TABS.SYSTEM_USERS,
};

const ORGANIZATION_TAB: TabBarItem<UsersView> = {
  value: USERS_VIEW.ORGANIZATION,
  label: USERS_AND_ROLES.TABS.ORGANIZATION_USERS,
};

// Server admins see both views; an organization admin sees only its organization.
const visibleTabs = (isServerAdmin: boolean) =>
  isServerAdmin ? [SYSTEM_TAB, ORGANIZATION_TAB] : [ORGANIZATION_TAB];

const activeView = (isServerAdmin: boolean, param: string | null): UsersView =>
  isServerAdmin && param !== USERS_VIEW.ORGANIZATION
    ? USERS_VIEW.SYSTEM
    : USERS_VIEW.ORGANIZATION;

const UsersAndRolesHeader: React.FC<{ onAdd: () => void }> = ({ onAdd }) => (
  <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
    <Typography variant="h4" component="h1">
      {USERS_AND_ROLES.TITLE}
    </Typography>
    <Button variant="contained" startIcon={<AddIcon />} onClick={onAdd}>
      {USERS_PAGE.ADD_USER}
    </Button>
  </Box>
);

const UsersAndRoles: React.FC = () => {
  const tabsId = useId();
  const { isHydrated } = useSession();
  const { isServerAdmin, isTenantManager } = useCapabilities();
  const { organizationId } = useOrganization();
  const { isEnabled } = useFeatureFlags();
  const showOrgUsers = isEnabled(FEATURE_FLAG.ENTERPRISE_ORGANIZATIONS);
  const [searchParams, setSearchParams] = useSearchParams();
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  // Runtime guard: requires server admin or org admin
  if (isHydrated && !isServerAdmin && !isTenantManager) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (!isHydrated) {
    return null;
  }

  const systemUsers = (
    <Users
      embedded
      addDialogOpen={addDialogOpen}
      onAddDialogOpenChange={setAddDialogOpen}
    />
  );

  // CE: single System Users view, no tabs
  if (!showOrgUsers) {
    return (
      <Box data-testid="users-and-roles-page" sx={{ p: 3, pt: 4 }}>
        <UsersAndRolesHeader onAdd={() => setAddDialogOpen(true)} />
        {systemUsers}
      </Box>
    );
  }

  const view = activeView(
    isServerAdmin,
    searchParams.get(USERS_VIEW_QUERY_PARAM),
  );

  return (
    <Box data-testid="users-and-roles-page" sx={{ p: 3, pt: 4 }}>
      <UsersAndRolesHeader onAdd={() => setAddDialogOpen(true)} />
      <TabBar
        value={view}
        onChange={(next) =>
          setSearchParams({ [USERS_VIEW_QUERY_PARAM]: next }, { replace: true })
        }
        items={visibleTabs(isServerAdmin)}
        ariaLabel={USERS_AND_ROLES.ARIA_TABS}
        idPrefix={tabsId}
      />
      <TabPanel idPrefix={tabsId} value={view}>
        {view === USERS_VIEW.SYSTEM ? (
          systemUsers
        ) : organizationId ? (
          <OrganizationUsers
            orgId={organizationId}
            embedded
            addDialogOpen={addDialogOpen}
            onAddDialogOpenChange={setAddDialogOpen}
          />
        ) : (
          <OrganizationRequired />
        )}
      </TabPanel>
    </Box>
  );
};

export default UsersAndRoles;
