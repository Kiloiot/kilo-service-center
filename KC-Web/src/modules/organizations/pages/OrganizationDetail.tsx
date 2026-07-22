/**
 * Organization Detail Page
 *
 * Organization detail view with configuration tabs and runtime admin guard.
 */

import React, { useEffect, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";

import type { OrganizationUI } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  Tab,
  Tabs,
  Typography,
} from "@mui/material";

import { ApiError } from "@services/api";
import { useOrganization as useOrganizationContext } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useOrganization as useOrganizationQuery } from "@hooks/useOrganizations";
import { formatRelativeDuration } from "@utils/formatters";
import { ORG_STATE, ROUTES } from "@constants/app";
import {
  ORG_USERS_PAGE,
  ORGANIZATION_FORM,
  ORGANIZATIONS_PAGE,
  SECTION_CONFIG,
  UI_COMMON,
} from "@constants/messages";
import { ArrowBackIcon, BusinessIcon, PeopleIcon } from "@theme/icons";

import OrganizationForm from "../components/OrganizationForm";

type ChipColor = "success" | "warning" | "error" | "default";

function getOrganizationStateColor(state: string): ChipColor {
  switch (state) {
    case ORG_STATE.ACTIVE:
      return "success";
    case ORG_STATE.SUSPENDED:
      return "warning";
    case ORG_STATE.ARCHIVED:
      return "error";
    default:
      return "default";
  }
}

function getOrganizationStateLabel(state: string): string {
  switch (state) {
    case ORG_STATE.ACTIVE:
      return ORGANIZATIONS_PAGE.STATE_ACTIVE;
    case ORG_STATE.SUSPENDED:
      return ORGANIZATIONS_PAGE.STATE_SUSPENDED;
    case ORG_STATE.ARCHIVED:
      return ORGANIZATIONS_PAGE.STATE_ARCHIVED;
    default:
      return state;
  }
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
  <div
    role="tabpanel"
    hidden={value !== index}
    id={`organization-tabpanel-${index}`}
  >
    {value === index && <Box sx={{ pt: 3 }}>{children}</Box>}
  </div>
);

interface OrganizationDetailHeaderProps {
  org: OrganizationUI;
  orgId: string;
}

const OrganizationDetailHeader: React.FC<OrganizationDetailHeaderProps> = ({
  org,
  orgId,
}) => {
  const navigate = useNavigate();
  return (
    <Box
      display="flex"
      justifyContent="space-between"
      alignItems="center"
      mb={3}
    >
      <Box display="flex" alignItems="center" gap={2}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate(ROUTES.ORGANIZATIONS)}
        >
          {ORGANIZATIONS_PAGE.BACK_TO_LIST}
        </Button>
        <Typography variant="h4" component="h1">
          {org.name}
        </Typography>
        <Chip
          label={getOrganizationStateLabel(org.state)}
          color={getOrganizationStateColor(org.state)}
          size="small"
        />
      </Box>
      <Button
        variant="outlined"
        startIcon={<PeopleIcon />}
        onClick={() =>
          navigate(ROUTES.ORGANIZATION_USERS.replace(":id", orgId))
        }
      >
        {ORG_USERS_PAGE.VIEW_MEMBERS}
      </Button>
    </Box>
  );
};

const OrganizationInfoCard: React.FC<{ org: OrganizationUI }> = ({ org }) => (
  <Card>
    <CardContent>
      <Box display="flex" alignItems="center" gap={2} mb={3}>
        <BusinessIcon sx={{ fontSize: 40, color: "primary.main" }} />
        <Typography variant="h6">{ORGANIZATIONS_PAGE.DETAILS_TITLE}</Typography>
      </Box>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_NAME}
          </Typography>
          <Typography variant="body1">{org.name}</Typography>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_STATE}
          </Typography>
          <Typography variant="body1">
            {getOrganizationStateLabel(org.state)}
          </Typography>
        </Grid>
        <Grid size={{ xs: 12 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_DESCRIPTION}
          </Typography>
          <Typography variant="body1">{org.description || "-"}</Typography>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_CAN_HAVE_BS}
          </Typography>
          <Chip
            label={
              org.canHaveBaseStations
                ? ORGANIZATIONS_PAGE.QUOTA_ALLOWED
                : ORGANIZATIONS_PAGE.QUOTA_NOT_ALLOWED
            }
            color={org.canHaveBaseStations ? "success" : "default"}
            size="small"
            variant="outlined"
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.INFO_TENANT_ID}
          </Typography>
          <Typography variant="body1">{org.tenantId}</Typography>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_MAX_BS_COUNT}
          </Typography>
          <Typography variant="body1">
            {org.maxBaseStationCount ?? ORGANIZATIONS_PAGE.QUOTA_UNLIMITED}
          </Typography>
        </Grid>
        <Grid size={{ xs: 12, sm: 6 }}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_MAX_EP_COUNT}
          </Typography>
          <Typography variant="body1">
            {org.maxEndpointCount ?? ORGANIZATIONS_PAGE.QUOTA_UNLIMITED}
          </Typography>
        </Grid>
      </Grid>
    </CardContent>
  </Card>
);

const OrganizationMetaCard: React.FC<{ org: OrganizationUI }> = ({ org }) => (
  <Card>
    <CardContent>
      <Typography variant="h6" mb={2}>
        {ORGANIZATION_FORM.INFO_TITLE}
      </Typography>
      <Box mb={2}>
        <Typography variant="body2" color="text.secondary">
          {ORGANIZATION_FORM.INFO_CREATED}
        </Typography>
        <Typography variant="body1">
          {formatRelativeDuration(org.createdAt)}
        </Typography>
      </Box>
      <Box>
        <Typography variant="body2" color="text.secondary">
          {ORGANIZATION_FORM.INFO_UPDATED}
        </Typography>
        <Typography variant="body1">
          {formatRelativeDuration(org.updatedAt)}
        </Typography>
      </Box>
    </CardContent>
  </Card>
);

const OrganizationTagsCard: React.FC<{ org: OrganizationUI }> = ({ org }) => {
  if (!org.tags || Object.keys(org.tags).length === 0) return null;
  return (
    <Card sx={{ mt: 2 }}>
      <CardContent>
        <Typography variant="h6" mb={2}>
          {ORGANIZATION_FORM.LABEL_TAGS}
        </Typography>
        <Box display="flex" flexWrap="wrap" gap={1}>
          {Object.entries(org.tags).map(([key, value]) => (
            <Chip
              key={key}
              label={`${key}: ${value}`}
              size="small"
              variant="outlined"
            />
          ))}
        </Box>
      </CardContent>
    </Card>
  );
};

const OrganizationDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { setOrganization } = useOrganizationContext();
  const { isAdmin, isHydrated } = useSession();
  const [activeTab, setActiveTab] = useState(0);

  const canQuery = isHydrated && isAdmin && Boolean(id);
  const {
    data: org,
    isLoading,
    isError,
    error,
  } = useOrganizationQuery(id || "", {
    enabled: canQuery,
  });

  useEffect(() => {
    if (org) {
      setOrganization(org.id, org.name);
    }
  }, [org, setOrganization]);

  if (!isHydrated) return null;

  // Runtime admin guard (deep link protection)
  if (!isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (isLoading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="400px"
      >
        <CircularProgress />
      </Box>
    );
  }

  if (isError) {
    const isForbidden = error instanceof ApiError && error.isForbidden();
    return (
      <Box sx={{ p: 3, pt: 4 }}>
        <Alert severity={isForbidden ? "warning" : "error"}>
          {isForbidden
            ? ORGANIZATIONS_PAGE.ERR_NOT_MEMBER
            : error instanceof Error
              ? error.message
              : ORGANIZATIONS_PAGE.ERR_NOT_FOUND}
        </Alert>
        <Button
          sx={{ mt: 2 }}
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate(ROUTES.ORGANIZATIONS)}
        >
          {ORGANIZATIONS_PAGE.BACK_TO_LIST}
        </Button>
      </Box>
    );
  }

  if (!org) {
    return (
      <Box sx={{ p: 3, pt: 4 }}>
        <Alert severity="warning">{ORGANIZATIONS_PAGE.ERR_NOT_FOUND}</Alert>
      </Box>
    );
  }

  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Box data-testid="organization-detail-page" sx={{ p: 3, pt: 4 }}>
      <OrganizationDetailHeader org={org} orgId={id ?? ""} />

      <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
        <Tabs
          value={activeTab}
          onChange={handleTabChange}
          aria-label={ORGANIZATIONS_PAGE.ARIA_TABS}
        >
          <Tab
            label={UI_COMMON.TITLE_DASHBOARD}
            id="organization-tab-0"
            aria-controls="organization-tabpanel-0"
          />
          <Tab
            label={SECTION_CONFIG}
            id="organization-tab-1"
            aria-controls="organization-tabpanel-1"
          />
        </Tabs>
      </Box>

      <TabPanel value={activeTab} index={0}>
        <Grid container spacing={3}>
          <Grid size={{ xs: 12, md: 8 }}>
            <OrganizationInfoCard org={org} />
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <OrganizationMetaCard org={org} />
            <OrganizationTagsCard org={org} />
          </Grid>
        </Grid>
      </TabPanel>

      <TabPanel value={activeTab} index={1}>
        <OrganizationForm
          organization={org}
          onDeleted={() => navigate(ROUTES.ORGANIZATIONS)}
        />
      </TabPanel>
    </Box>
  );
};

export default OrganizationDetail;
