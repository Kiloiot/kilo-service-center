import React, { useMemo } from "react";
import { useNavigate } from "react-router-dom";

import {
  useDashboardAnalytics,
  useDashboardStats,
  useSystemStatus,
} from "@hooks";
import { Box, CircularProgress, Grid, Paper, Typography } from "@mui/material";

import { ServiceStatusTable } from "@components/common/ServiceStatusWidget";
import { StatCard, StatCardRow } from "@components/common/StatCard";
import { useCapabilities } from "@hooks/useCapabilities";
import { formatDashboardCountTrend } from "@utils/formatters";
import { COMPACT_COUNT, FRACTION_DIGITS, ROUTES } from "@constants/app";
import {
  DASHBOARD_PAGE,
  SYSTEM_STATUS,
  VALUE_FORMAT,
} from "@constants/messages";
import { DeviceHubIcon, MessageIcon, RouterIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { AlertsPanel } from "../components/AlertsPanel";
import { ControlPlaneStrip } from "../components/ControlPlaneStrip";
import {
  baseStationCardCounts,
  endpointCardCounts,
} from "../utils/dashboard-cards";

const formatMessageCount = (count: number): string => {
  if (count >= COMPACT_COUNT.MILLION) {
    return `${(count / COMPACT_COUNT.MILLION).toFixed(FRACTION_DIGITS.COMPACT_COUNT)}${VALUE_FORMAT.MILLIONS_SUFFIX}`;
  }
  if (count >= COMPACT_COUNT.THOUSAND) {
    return `${(count / COMPACT_COUNT.THOUSAND).toFixed(FRACTION_DIGITS.COMPACT_COUNT)}${VALUE_FORMAT.THOUSANDS_SUFFIX}`;
  }
  return count.toString();
};

function baseStationSubtitle(total: number, offline: number): string {
  if (total === 0) return DASHBOARD_PAGE.NO_BASE_STATIONS;
  if (offline === 0) return DASHBOARD_PAGE.ALL_ONLINE;
  return `${offline} ${DASHBOARD_PAGE.OFFLINE_LABEL}`;
}

function endpointSubtitle(total: number): string {
  if (total === 0) return DASHBOARD_PAGE.NO_ENDPOINTS;
  return `${total} ${DASHBOARD_PAGE.REGISTERED_LABEL}`;
}

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const { isServerAdmin, isBaseStationManager, isEndpointManager } =
    useCapabilities();

  // React Query hooks
  const { baseStations, endpoints } = useDashboardStats({
    baseStations: isBaseStationManager,
    endpoints: isEndpointManager,
  });
  const { data: analyticsData } = useDashboardAnalytics();
  const { data: systemStatusData, isLoading: systemStatusLoading } =
    useSystemStatus();

  const stationCounts = useMemo(
    () => baseStationCardCounts(baseStations, new Date()),
    [baseStations],
  );
  const endpointCounts = useMemo(
    () => endpointCardCounts(endpoints, new Date()),
    [endpoints],
  );
  // totalMessages from the backend is already scoped to the last 24h
  const totalMessages = analyticsData?.totalMessages || 0;

  return (
    <Box data-testid="dashboard-page" sx={{ p: 3, pt: 4 }}>
      {/* Header */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h4" component="h1">
          {DASHBOARD_PAGE.TITLE}
        </Typography>
        <Typography variant="body1" color="text.secondary">
          {DASHBOARD_PAGE.SUBTITLE}
        </Typography>
      </Box>

      <StatCardRow>
        {isBaseStationManager && (
          <StatCard
            label={DASHBOARD_PAGE.BASE_STATIONS_ONLINE}
            value={stationCounts.online}
            caption={baseStationSubtitle(
              stationCounts.total,
              stationCounts.offline,
            )}
            icon={<RouterIcon />}
            color="primary"
            trendLabel={formatDashboardCountTrend(
              stationCounts.addedLastWeek,
              DASHBOARD_PAGE.ADDED_LAST_WEEK,
            )}
            onClick={() => navigate(ROUTES.BASE_STATIONS)}
          />
        )}
        {isEndpointManager && (
          <StatCard
            label={DASHBOARD_PAGE.ENDPOINTS_ATTACHED}
            value={endpointCounts.attached}
            caption={endpointSubtitle(endpointCounts.total)}
            icon={<DeviceHubIcon />}
            color="primary"
            trendLabel={formatDashboardCountTrend(
              endpointCounts.addedLastWeek,
              DASHBOARD_PAGE.ADDED_LAST_WEEK,
            )}
            onClick={() => navigate(ROUTES.ENDPOINTS)}
          />
        )}
        <StatCard
          label={DASHBOARD_PAGE.MESSAGES_RECEIVED}
          value={formatMessageCount(totalMessages)}
          caption={DASHBOARD_PAGE.LAST_24_HOURS}
          icon={<MessageIcon />}
          color="primary"
        />
      </StatCardRow>

      <Grid container spacing={3}>
        {isEndpointManager && (
          <Grid size={componentSpacing.gridSpan.full}>
            <ControlPlaneStrip />
          </Grid>
        )}
        {isServerAdmin && (
          <Grid size={componentSpacing.gridSpan.twoThirds}>
            <AlertsPanel />
          </Grid>
        )}
        <Grid
          size={
            isServerAdmin
              ? componentSpacing.gridSpan.third
              : componentSpacing.gridSpan.full
          }
        >
          <Paper
            sx={{
              p: 3,
              minHeight: componentSpacing.statusPanel.minHeight,
              height: "100%",
            }}
          >
            <Typography
              variant="h6"
              gutterBottom
              sx={{ fontWeight: componentSpacing.cardTitle.fontWeight }}
            >
              {DASHBOARD_PAGE.SERVICE_CENTER_STATUS}
            </Typography>
            {systemStatusLoading ? (
              <Box
                sx={{
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                  p: 4,
                }}
              >
                <CircularProgress size={componentSpacing.spinner.section} />
                <Typography sx={{ ml: 1 }} color="text.secondary">
                  {SYSTEM_STATUS.LABEL_CHECKING}
                </Typography>
              </Box>
            ) : systemStatusData?.services ? (
              <ServiceStatusTable
                services={systemStatusData.services}
                timestamp={systemStatusData.timestamp}
              />
            ) : (
              <Typography color="text.secondary">
                {SYSTEM_STATUS.TOOLTIP_UNABLE}
              </Typography>
            )}
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard;
