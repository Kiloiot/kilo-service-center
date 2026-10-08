import React from "react";

import type { BaseStationUI } from "@api-types/api";
import { Alert, Box, CircularProgress, Typography } from "@mui/material";

import { formatDateTime, formatDurationSeconds } from "@utils/date-format";
import { getErrorMessage } from "@utils/error-message";
import { formatEui } from "@utils/eui";
import { formatFraction, truncateWithEllipsis } from "@utils/formatters";
import { getMonoBody1 } from "@utils/typography";
import type { BaseStationStatus } from "@constants/app";
import {
  FRACTION_DIGITS,
  JSON_PREVIEW,
  NANOSECONDS_PER_MILLISECOND,
  TRUNCATION,
} from "@constants/app";
import {
  BASE_STATION_DETAILS,
  ERR_LOAD_BS_DETAILS,
  LOADER,
  VALUE_FORMAT,
} from "@constants/messages";
import { componentSpacing } from "@theme/index";

import { BaseStationStatusChip } from "./BaseStationStatusChip";

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
        <CircularProgress size={componentSpacing.spinner.inline} />
        <Typography variant="body2" color="text.secondary">
          {LOADER.LOADING}
        </Typography>
      </Box>
    );
  if (error)
    return (
      <Alert severity="error" variant="outlined">
        {getErrorMessage(error, ERR_LOAD_BS_DETAILS)}
      </Alert>
    );
  return null;
}

/** Formats a nullable 0..1 load as a percentage. */
function fmtPercent(
  value: number | undefined | null,
  decimals: number = FRACTION_DIGITS.PERCENT,
): string {
  return value != null
    ? formatFraction(value, decimals)
    : BASE_STATION_DETAILS.NOT_AVAILABLE;
}

interface BaseStationInfoPanelProps {
  baseStation: {
    eui: string;
    name?: string;
    status: BaseStationStatus;
  };
  baseStationDetails: BaseStationUI | null | undefined;
  loadingDetails: boolean;
  detailsError: Error | null;
}

/** Three-column info display: Basic Information, Performance Metrics, System Information. */
const BaseStationInfoPanel: React.FC<BaseStationInfoPanelProps> = ({
  baseStation,
  baseStationDetails,
  loadingDetails,
  detailsError,
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
            {formatEui(baseStation.eui)}
          </Typography>
        </InfoRow>
        <InfoRow label={BASE_STATION_DETAILS.STATUS}>
          <BaseStationStatusChip status={baseStation.status} />
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
                ? `${baseStationDetails.temperatureCelsius.toFixed(FRACTION_DIGITS.TEMPERATURE)}${VALUE_FORMAT.CELSIUS_SUFFIX}`
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
              {fmtPercent(
                baseStationDetails?.dutyCycle,
                FRACTION_DIGITS.DUTY_CYCLE_PERCENT,
              )}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.BS_CONFIG}>
            <Typography variant="body1" sx={(theme) => getMonoBody1(theme)}>
              {baseStationDetails?.bsConfig
                ? truncateWithEllipsis(
                    JSON.stringify(
                      baseStationDetails.bsConfig,
                      null,
                      JSON_PREVIEW.INDENT,
                    ),
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
                ? formatDateTime(
                    baseStationDetails.systemTime / NANOSECONDS_PER_MILLISECOND,
                  )
                : BASE_STATION_DETAILS.NOT_AVAILABLE}
            </Typography>
          </InfoRow>
          <InfoRow label={BASE_STATION_DETAILS.UPTIME}>
            <Typography variant="body1">
              {baseStationDetails?.uptimeSeconds
                ? formatDurationSeconds(baseStationDetails.uptimeSeconds)
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
        </DetailColumnContent>
      </Box>
    </Box>
  );
};

export default BaseStationInfoPanel;
