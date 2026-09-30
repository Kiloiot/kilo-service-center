/**
 * Uplink totals of one device: every figure its stats RPC reports.
 */

import type { TrafficSummaryUI } from "@api-types/api";
import { Box, Paper, Skeleton, Typography } from "@mui/material";

import { formatDateTime } from "@utils/date-format";
import { getErrorMessage } from "@utils/error-message";
import { formatMeasurement } from "@utils/formatters";
import { TRAFFIC_SUMMARY_LAYOUT } from "@constants/app";
import { DATA_TABLE, TRAFFIC_SUMMARY } from "@constants/messages";

interface SummaryItem {
  label: string;
  value: string | number | undefined;
}

function summaryItems(summary: TrafficSummaryUI): SummaryItem[] {
  return [
    { label: TRAFFIC_SUMMARY.TOTAL, value: summary.totalMessages },
    { label: TRAFFIC_SUMMARY.UNIQUE_ENDPOINTS, value: summary.uniqueEndpoints },
    { label: TRAFFIC_SUMMARY.TODAY, value: summary.messagesToday },
    { label: TRAFFIC_SUMMARY.THIS_WEEK, value: summary.messagesThisWeek },
    { label: TRAFFIC_SUMMARY.THIS_MONTH, value: summary.messagesThisMonth },
    { label: TRAFFIC_SUMMARY.ACTIVE_DAYS, value: summary.activeDays },
    {
      label: TRAFFIC_SUMMARY.AVG_RSSI,
      value: formatMeasurement(summary.avgRssi),
    },
    {
      label: TRAFFIC_SUMMARY.AVG_SNR,
      value: formatMeasurement(summary.avgSnr),
    },
    {
      label: TRAFFIC_SUMMARY.FIRST_SEEN,
      value: formatDateTime(summary.firstSeen),
    },
    {
      label: TRAFFIC_SUMMARY.LAST_SEEN,
      value: formatDateTime(summary.lastSeen),
    },
  ].filter((item) => item.value !== undefined);
}

interface TrafficSummaryCardProps {
  summary: TrafficSummaryUI | undefined;
  error: Error | null;
}

export function TrafficSummaryCard({
  summary,
  error,
}: TrafficSummaryCardProps) {
  return (
    <Paper variant="outlined" sx={{ p: TRAFFIC_SUMMARY_LAYOUT.PADDING, mb: 2 }}>
      <Typography variant="subtitle2" gutterBottom>
        {TRAFFIC_SUMMARY.TITLE}
      </Typography>
      {error ? (
        <Typography variant="body2" color="error">
          {getErrorMessage(error, DATA_TABLE.LOAD_FAILED)}
        </Typography>
      ) : summary ? (
        <Box
          sx={{
            display: "flex",
            flexWrap: "wrap",
            gap: TRAFFIC_SUMMARY_LAYOUT.ITEM_GAP,
          }}
        >
          {summaryItems(summary).map((item) => (
            <Box key={item.label}>
              <Typography variant="caption" color="text.secondary">
                {item.label}
              </Typography>
              <Typography variant="body1">{item.value}</Typography>
            </Box>
          ))}
        </Box>
      ) : (
        <Skeleton
          variant="rectangular"
          height={TRAFFIC_SUMMARY_LAYOUT.SKELETON_HEIGHT}
        />
      )}
    </Paper>
  );
}
