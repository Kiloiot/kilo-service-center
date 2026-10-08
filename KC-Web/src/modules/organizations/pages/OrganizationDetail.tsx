/**
 * Organization Detail Page
 *
 * Organization detail view with configuration tabs and runtime admin guard.
 */

import React, { useEffect } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";

import type { OrganizationUI } from "@api-types/api";
import { isApiError } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  Typography,
} from "@mui/material";

import { BackButton } from "@components/common/BackButton";
import { ViewTabs } from "@components/common/ViewTabs";
import { useOrganization as useOrganizationContext } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { useOrganization as useOrganizationQuery } from "@hooks/useOrganizations";
import { organizationStateChip } from "@utils/chipMappings";
import { formatRelativeDuration } from "@utils/date-format";
import { getErrorMessage } from "@utils/error-message";
import {
  ORGANIZATION_VIEW,
  ORGANIZATION_VIEWS,
  type OrganizationView,
  ROUTES,
} from "@constants/app";
import {
  ORG_USERS_PAGE,
  ORGANIZATION_FORM,
  ORGANIZATIONS_PAGE,
  SECTION_CONFIG,
  UI_COMMON,
} from "@constants/messages";
import { organizationUsersPath } from "@router/paths";
import { accentTextColor } from "@theme/controls";
import { BusinessIcon, PeopleIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import OrganizationForm from "../components/OrganizationForm";

const VIEW_LABELS: Record<OrganizationView, string> = {
  [ORGANIZATION_VIEW.OVERVIEW]: UI_COMMON.TITLE_DASHBOARD,
  [ORGANIZATION_VIEW.CONFIGURATION]: SECTION_CONFIG,
};

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
        <BackButton
          label={ORGANIZATIONS_PAGE.BACK_TO_LIST}
          onClick={() => navigate(ROUTES.ORGANIZATIONS)}
        />
        <Typography variant="h4" component="h1">
          {org.name}
        </Typography>
        <Chip
          label={organizationStateChip(org.state).label}
          color={organizationStateChip(org.state).color}
          size="small"
        />
      </Box>
      <Button
        variant="outlined"
        startIcon={<PeopleIcon />}
        onClick={() => navigate(organizationUsersPath(orgId))}
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
        <BusinessIcon
          sx={{
            fontSize: componentSpacing.headerIcon.size,
            color: accentTextColor,
          }}
        />
        <Typography variant="h6">{ORGANIZATIONS_PAGE.DETAILS_TITLE}</Typography>
      </Box>

      <Grid container spacing={2}>
        <Grid size={componentSpacing.gridSpan.halfFromSm}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_NAME}
          </Typography>
          <Typography variant="body1">{org.name}</Typography>
        </Grid>
        <Grid size={componentSpacing.gridSpan.halfFromSm}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_STATE}
          </Typography>
          <Typography variant="body1">
            {organizationStateChip(org.state).label}
          </Typography>
        </Grid>
        <Grid size={componentSpacing.gridSpan.full}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.LABEL_DESCRIPTION}
          </Typography>
          <Typography variant="body1">{org.description || "-"}</Typography>
        </Grid>
        <Grid size={componentSpacing.gridSpan.halfFromSm}>
          <Typography variant="body2" color="text.secondary">
            {ORGANIZATION_FORM.INFO_TENANT_ID}
          </Typography>
          <Typography variant="body1">{org.tenantId}</Typography>
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
  const { isHydrated } = useSession();
  const { isServerAdmin: isAdmin } = useCapabilities();

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
        minHeight={componentSpacing.stateView.pageMinHeight}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (isError) {
    const isForbidden = isApiError(error) && error.isForbidden();
    return (
      <Box sx={{ p: 3, pt: 4 }}>
        <Alert severity={isForbidden ? "warning" : "error"}>
          {isForbidden
            ? ORGANIZATIONS_PAGE.ERR_NOT_MEMBER
            : getErrorMessage(error, ORGANIZATIONS_PAGE.ERR_NOT_FOUND)}
        </Alert>
        <BackButton
          label={ORGANIZATIONS_PAGE.BACK_TO_LIST}
          onClick={() => navigate(ROUTES.ORGANIZATIONS)}
          sx={{ mt: 2 }}
        />
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

  return (
    <Box data-testid="organization-detail-page" sx={{ p: 3, pt: 4 }}>
      <OrganizationDetailHeader org={org} orgId={id ?? ""} />

      <ViewTabs
        views={ORGANIZATION_VIEWS}
        labels={VIEW_LABELS}
        ariaLabel={ORGANIZATIONS_PAGE.ARIA_TABS}
      >
        {(view) =>
          view === ORGANIZATION_VIEW.OVERVIEW ? (
            <Grid container spacing={3}>
              <Grid size={componentSpacing.gridSpan.twoThirds}>
                <OrganizationInfoCard org={org} />
              </Grid>
              <Grid size={componentSpacing.gridSpan.third}>
                <OrganizationMetaCard org={org} />
                <OrganizationTagsCard org={org} />
              </Grid>
            </Grid>
          ) : (
            <OrganizationForm
              organization={org}
              onDeleted={() => navigate(ROUTES.ORGANIZATIONS)}
            />
          )
        }
      </ViewTabs>
    </Box>
  );
};

export default OrganizationDetail;
