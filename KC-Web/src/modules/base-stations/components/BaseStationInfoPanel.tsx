import React from "react";

import type { BaseStationUI } from "@api-types/api";
import { Alert, Box, Chip, CircularProgress, Typography } from "@mui/material";

import {
  formatDate,
  formatDateTime,
  formatEUIWithDashes,
  truncateWithEllipsis,
} from "@utils/formatters";
import { getMonoBody1 } from "@utils/typography";
import { TRUNCATION } from "@constants/app";
import {
  BASE_STATION_DETAILS,
  ERR_LOAD_BS_DETAILS,
  LOADER,
} from "@constants/messages";

/** Formats uptime seconds, including days when greater than 24h. */
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  if (days > 0) return `${days}d ${hours}h ${minutes}m ${secs}s`;
  return `${hours}h ${minutes}m ${secs}s`;
}

/** Renders a label/value info row. */
function InfoRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <Box mb={1}>
      <Typography variant="body2" color="text.secondary">
        {label}
      </Typography>
      {children}
    </Box>
  );
}

/** Renders loading spinner, error alert, or children when data is available. */
function DetailColumnContent({
  hasData,
  loading,
  error,
  children,
}: {
  hasData: boolean;
  loading: boolean;
  error: Error | null;
  children: React.ReactNode;
}) {
  if (hasData) return <>{children}</>;
  if (loading)
    return (
      <Box display="flex" alignItems="center" gap={1}>
        <CircularProgress size={16} />
        <Typography variant="body2" color="text.secondary">
          {LOADER.LOADING}
        </Typography>
      </Box>
    );
  if (error)
    return (
      <Alert severity="error" variant="outlined">
        {error instanceof Error ? error.message : ERR_LOAD_BS_DETAILS}
      </Alert>
    );
  return null;
}

/** Formats a nullable percentage value. */
function fmtPercent(value: number | undefined | null, decimals = 1): string {
  return value != null
    ? `${(value * 100).toFixed(decimals)}%`
    : BASE_STATION_DETAILS.NOT_AVAILABLE;
}

interface BaseStationInfoPanelProps {
  baseStation: {
    eui: string;
    name?: string;
    status: "online" | "offline";
    certificateExpiryDate?: string;
  };
  baseStationDetails: BaseStationUI | null | undefined;
  loadingDetails: boolean;
  detailsError: Error | null;
  isExpiringSoon: boolean;
  isExpired: boolean;
}

/** Three-column info display: Basic Information, Performance Metrics, System Information. */
const BaseStationInfoPanel: React.FC<BaseStationInfoPanelProps> = ({
  baseStation,
  baseStationDetails,
  loadingDetails,
  detailsError,
  isExpiringSoon,
  isExpired,
}) => {
  const hasDetails = !!baseStationDetails;

  return (
    <Box
      sx={{
        display: "flex",
        gap: 32,
        flexWrap: { xs: "wrap", md: "nowrap" },
      }}
    >
      {/* Basic Information */}
      <Box sx={{ minWidth: 0 }}>
        <Typography
          variant="subtitle2"
          color="text.secondary"
          gutterBottom
          sx={{ mb: 2 }}
        >
          {BASE_STATION_DETAILS.BASIC_INFORMATION}
        </Typography>
        <InfoRow label={BASE_STATION_DETAILS.BASE_STATION_NAME}>
          <Typography variant="body1">
            {baseStation.name || BASE_STATION_DETAILS.NOT_SET}
          </Typography>
        </InfoRow>
        <InfoRow label={BASE_STATION_DETAILS.BASE_STATION_EUI}>
          <Typography variant="body1" sx={(theme) => getMonoBody1(theme)}>
            {formatEUIWithDashes(baseStation.eui)}
          </Typography>
        </InfoRow>
        <InfoRow label={BASE_STATION_DETAILS.STATUS}>
          <Chip
            label={baseStation.status}
            color={baseStation.status === "online" ? "success" : "default"}
            size="small"
          />
        </InfoRow>
      </Box>

      {/* Performance Metrics */}
      <Box sx={{ minWidth: 0 }}>
        <Typography
          variant="subtitle2"
          color="text.secondary"
          gutterBottom
          sx={{ mb: 2 }}
        >
          {BASE_STATION_DETAILS.PERFORMANCE_METRICS}
        </Typography>
        <DetailColumnContent
          hasData={hasDetails}
          loading={loadingDetails}
          error={detailsError}
        >
          <InfoRow label={BASE_STATION_DETAILS.TEMPERATURE}>
            <Typography variant="body1">
              {baseStationDetails?.temperatureCelsius != null
                ? `${baseStationDetails.temperatureCelsius.toFixed(1)}°C`
                : BASE_STATION_DETAILS.NOT_AVAILABLE}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.CPU_LOAD}>
            <Typography variant="body1">
              {fmtPercent(baseStationDetails?.cpuLoad)}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.MEMORY_LOAD}>
            <Typography variant="body1">
              {fmtPercent(baseStationDetails?.memoryLoad)}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.DUTY_CYCLE}>
            <Typography variant="body1">
              {fmtPercent(baseStationDetails?.dutyCycle, 2)}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.BS_CONFIG}>
            <Typography variant="body1" sx={(theme) => getMonoBody1(theme)}>
              {baseStationDetails?.bsConfig
                ? truncateWithEllipsis(
                    JSON.stringify(baseStationDetails.bsConfig, null, 2),
                    TRUNCATION.CONFIG_PREVIEW_LENGTH,
                    TRUNCATION.ELLIPSIS,
                  )
                : BASE_STATION_DETAILS.NOT_AVAILABLE}
            </Typography>
          </InfoRow>
        </DetailColumnContent>
      </Box>

      {/* System Information */}
      <Box sx={{ minWidth: 0 }}>
        <Typography
          variant="subtitle2"
          color="text.secondary"
          gutterBottom
          sx={{ mb: 2 }}
        >
          {BASE_STATION_DETAILS.SYSTEM_INFORMATION}
        </Typography>
        <DetailColumnContent
          hasData={hasDetails}
          loading={loadingDetails}
          error={detailsError}
        >
          <InfoRow label={BASE_STATION_DETAILS.SYSTEM_TIME}>
            <Typography variant="body1">
              {baseStationDetails?.systemTime
                ? formatDateTime(baseStationDetails.systemTime / 1000000)
                : BASE_STATION_DETAILS.NOT_AVAILABLE}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.UPTIME}>
            <Typography variant="body1">
              {baseStationDetails?.uptimeSeconds
                ? formatUptime(baseStationDetails.uptimeSeconds)
                : BASE_STATION_DETAILS.NOT_AVAILABLE}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.LAST_STATUS_UPDATE}>
            <Typography variant="body1">
              {baseStationDetails?.lastStatusAt
                ? formatDateTime(baseStationDetails.lastStatusAt)
                : BASE_STATION_DETAILS.NOT_AVAILABLE}
            </Typography>
          </InfoRow>
          {baseStation.certificateExpiryDate && (
            <InfoRow label={BASE_STATION_DETAILS.CERTIFICATE_EXPIRY}>
              <Typography
                variant="body1"
                color={
                  isExpired
                    ? "error.main"
                    : isExpiringSoon
                      ? "warning.main"
                      : "text.primary"
                }
              >
                {formatDate(baseStation.certificateExpiryDate)}
              </Typography>
            </InfoRow>
          )}
        </DetailColumnContent>
      </Box>
    </Box>
  );
};

export default BaseStationInfoPanel;
