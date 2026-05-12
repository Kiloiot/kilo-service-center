/**
 * Tabbed container for user management:
 * - Tab 0: System Users (existing Users component)
 * - Tab 1: Organization Users (or OrganizationRequired prompt)
 *
 * Admin-only page with runtime guard for deep link protection.
 */

import React, { useState } from "react";
import { Navigate, useSearchParams } from "react-router-dom";

import { Box, Button, Tab, Tabs, Typography } from "@mui/material";

import OrganizationRequired from "@components/common/OrganizationRequired";
import { useFeatureFlags } from "@contexts/FeatureFlagContext";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { ROUTES } from "@constants/app";
import { USERS_AND_ROLES, USERS_PAGE } from "@constants/messages";
import { AddIcon } from "@theme/icons";

import OrganizationUsers from "./OrganizationUsers";
import Users from "./Users";

const TAB_PARAM = "tab";
const TAB_VALUES = ["system", "organization"] as const;

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`users-tabpanel-${index}`}
    >
      {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
    </div>
  );
};

const UsersAndRoles: React.FC = () => {
  const { isHydrated } = useSession();
  const { isServerAdmin, isOrgAdmin } = useCapabilities();
  const { organizationId } = useOrganization();
  const { isEnabled } = useFeatureFlags();
  const showOrgUsers = isEnabled("enterprise_organizations");
  const [searchParams, setSearchParams] = useSearchParams();
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  // Derive active tab from URL query param
  const tabParam = searchParams.get(TAB_PARAM);
  const defaultTab = 0;
  const activeTab = isServerAdmin
    ? tabParam === TAB_VALUES[1]
      ? 1
      : tabParam === TAB_VALUES[0]
        ? 0
        : defaultTab
    : 0;

  // Runtime guard: requires server admin or org admin
  if (isHydrated && !isServerAdmin && !isOrgAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (!isHydrated) {
    return null;
  }

  // CE: single System Users view, no tabs
  if (!showOrgUsers) {
    return (
      <Box data-testid="users-and-roles-page" sx={{ p: 3, pt: 4 }}>
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={3}
        >
          <Typography variant="h4" component="h1">
            {USERS_AND_ROLES.TITLE}
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setAddDialogOpen(true)}
          >
            {USERS_PAGE.ADD_USER}
          </Button>
        </Box>
        <Users
          embedded
          addDialogOpen={addDialogOpen}
          onAddDialogOpenChange={setAddDialogOpen}
        />
      </Box>
    );
  }

  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    const tabKey = isServerAdmin ? TAB_VALUES[newValue] : TAB_VALUES[1];
    setSearchParams({ [TAB_PARAM]: tabKey }, { replace: true });
  };

  // ECE: existing tabbed layout
  return (
    <Box data-testid="users-and-roles-page" sx={{ p: 3, pt: 4 }}>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h4" component="h1">
          {USERS_AND_ROLES.TITLE}
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setAddDialogOpen(true)}
        >
          {USERS_PAGE.ADD_USER}
        </Button>
      </Box>

      <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
        <Tabs
          value={activeTab}
          onChange={handleTabChange}
          aria-label={USERS_AND_ROLES.ARIA_TABS}
        >
          {/* Server admin sees both tabs; org admin sees only Organization Users */}
          {isServerAdmin && (
            <Tab
              label={USERS_AND_ROLES.TABS.SYSTEM_USERS}
              id="users-tab-0"
              aria-controls="users-tabpanel-0"
            />
          )}
          <Tab
            label={USERS_AND_ROLES.TABS.ORGANIZATION_USERS}
            id="users-tab-1"
            aria-controls="users-tabpanel-1"
          />
        </Tabs>
      </Box>

      {isServerAdmin && (
        <TabPanel value={activeTab} index={0}>
          <Users
            embedded
            addDialogOpen={addDialogOpen}
            onAddDialogOpenChange={setAddDialogOpen}
          />
        </TabPanel>
      )}

      <TabPanel value={activeTab} index={isServerAdmin ? 1 : 0}>
        {organizationId ? (
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
