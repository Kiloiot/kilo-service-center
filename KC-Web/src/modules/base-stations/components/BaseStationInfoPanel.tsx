import React from "react";

import type { BaseStationUI } from "@api-types/api";
import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  Typography,
} from "@mui/material";

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
        <Box mb={1}>
          <Typography variant="body2" color="text.secondary">
            {BASE_STATION_DETAILS.BASE_STATION_NAME}
          </Typography>
          <Typography variant="body1">
            {baseStation.name || BASE_STATION_DETAILS.NOT_SET}
          </Typography>
        </Box>
        <Box mb={1}>
          <Typography variant="body2" color="text.secondary">
            {BASE_STATION_DETAILS.BASE_STATION_EUI}
          </Typography>
          <Typography variant="body1" sx={(theme) => getMonoBody1(theme)}>
            {formatEUIWithDashes(baseStation.eui)}
          </Typography>
        </Box>
        <Box mb={1}>
          <Typography variant="body2" color="text.secondary">
            {BASE_STATION_DETAILS.STATUS}
          </Typography>
          <Chip
            label={baseStation.status}
            color={baseStation.status === "online" ? "success" : "default"}
            size="small"
          />
        </Box>
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
        {baseStationDetails ? (
          <>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.TEMPERATURE}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.temperatureCelsius != null
                  ? `${baseStationDetails.temperatureCelsius.toFixed(1)}°C`
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.CPU_LOAD}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.cpuLoad != null
                  ? `${(baseStationDetails.cpuLoad * 100).toFixed(1)}%`
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.MEMORY_LOAD}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.memoryLoad != null
                  ? `${(baseStationDetails.memoryLoad * 100).toFixed(1)}%`
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.DUTY_CYCLE}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.dutyCycle != null
                  ? `${(baseStationDetails.dutyCycle * 100).toFixed(2)}%`
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.BS_CONFIG}
              </Typography>
              <Typography
                variant="body1"
                sx={(theme) => getMonoBody1(theme)}
              >
                {baseStationDetails.bsConfig
                  ? truncateWithEllipsis(
                      JSON.stringify(baseStationDetails.bsConfig, null, 2),
                      TRUNCATION.CONFIG_PREVIEW_LENGTH,
                      TRUNCATION.ELLIPSIS,
                    )
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
          </>
        ) : loadingDetails ? (
          <Box display="flex" alignItems="center" gap={1}>
            <CircularProgress size={16} />
            <Typography variant="body2" color="text.secondary">
              {LOADER.LOADING}
            </Typography>
          </Box>
        ) : detailsError ? (
          <Alert severity="error" variant="outlined">
            {detailsError instanceof Error
              ? detailsError.message
              : ERR_LOAD_BS_DETAILS}
          </Alert>
        ) : null}
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
        {baseStationDetails ? (
          <>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.SYSTEM_TIME}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.systemTime
                  ? formatDateTime(baseStationDetails.systemTime / 1000000)
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.UPTIME}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.uptimeSeconds
                  ? formatUptime(baseStationDetails.uptimeSeconds)
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            <Box mb={1}>
              <Typography variant="body2" color="text.secondary">
                {BASE_STATION_DETAILS.LAST_STATUS_UPDATE}
              </Typography>
              <Typography variant="body1">
                {baseStationDetails.lastStatusAt
                  ? formatDateTime(baseStationDetails.lastStatusAt)
                  : BASE_STATION_DETAILS.NOT_AVAILABLE}
              </Typography>
            </Box>
            {baseStation.certificateExpiryDate && (
              <Box mb={1}>
                <Typography variant="body2" color="text.secondary">
                  {BASE_STATION_DETAILS.CERTIFICATE_EXPIRY}
                </Typography>
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
              </Box>
            )}
          </>
        ) : loadingDetails ? (
          <Box display="flex" alignItems="center" gap={1}>
            <CircularProgress size={16} />
            <Typography variant="body2" color="text.secondary">
              {LOADER.LOADING}
            </Typography>
          </Box>
        ) : detailsError ? (
          <Alert severity="error" variant="outlined">
            {detailsError instanceof Error
              ? detailsError.message
              : ERR_LOAD_BS_DETAILS}
          </Alert>
        ) : null}
      </Box>
    </Box>
  );
};

export default BaseStationInfoPanel;
