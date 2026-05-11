import React, { useMemo } from "react";

import type { ActivityItem } from "@api-types/api";
import {
  Box,
  CircularProgress,
  Paper,
  Typography,
  useTheme,
} from "@mui/material";

import { formatDate, formatEUIWithDashes } from "@utils/formatters";
import { ENDPOINT_DETAILS } from "@constants/messages";
import { formatTerminalTime,getTerminalStyles } from "@theme/index";

// =============================================================================
// Types
// =============================================================================

/** Flattened item for internal rendering. Consumers pass canonical ActivityItem[]. */
interface FlatItem {
  id: string;
  time: Date;
  type: "event" | "message";
  // Event fields
  eventType?: string;
  category?: string;
  severity?: string;
  title?: string;
  description?: string;
  sourceName?: string;
  // Message fields
  bsEui?: string;
  epEui?: string;
  packetCnt?: number;
  rssi?: number;
  snr?: number;
  userData?: string;
  baseStations?: Array<{ bsEui: string; snr: number; rssi: number }>;
  duplicate?: boolean;
  decodedPayload?: Record<string, unknown>;
  decodeStatus?: string;
  decodeErrorCode?: string;
}

/** Convert canonical ActivityItem to internal flat rendering shape. */
function toFlatItem(item: ActivityItem, idx: number): FlatItem {
  if (item.type === "event" && item.event) {
    return {
      id: item.event.id || `evt-${idx}`,
      time: item.occurredAt,
      type: "event",
      eventType: item.event.eventType,
      category: item.event.category,
      severity: item.event.severity,
      title: item.event.title,
      description: item.event.description,
      sourceName: item.event.sourceName,
    };
  }
  const msg = item.message;
  return {
    id: msg?.id || `msg-${idx}`,
    time: item.occurredAt,
    type: "message",
    bsEui: msg?.bsEui,
    epEui: msg?.epEui,
    packetCnt: msg?.packetCnt,
    rssi: msg?.rssi,
    snr: msg?.snr,
    userData: msg?.userData,
    baseStations: msg?.baseStations?.map((bs) => ({
      bsEui: bs.bsEui,
      snr: bs.snr,
      rssi: bs.rssi,
    })),
    duplicate: msg?.duplicate,
    decodedPayload: msg?.decodedPayload,
    decodeStatus: msg?.decodeStatus,
    decodeErrorCode: msg?.decodeErrorCode,
  };
}

interface ActivityTimelineProps {
  items: ActivityItem[];
  variant: "compact" | "detailed";
  loading?: boolean;
  emptyMessage?: string;
}

// =============================================================================
// Helpers
// =============================================================================

/** Group items by date string, returning entries in reverse-chronological order. */
function groupByDate(items: FlatItem[]): Map<string, FlatItem[]> {
  const groups = new Map<string, FlatItem[]>();
  const todayKey = new Date().toDateString();

  for (const item of items) {
    const itemKey = item.time.toDateString();
    const label =
      itemKey === todayKey
        ? ENDPOINT_DETAILS.LABEL_TODAY
        : formatDate(item.time.toISOString());
    const group = groups.get(label);
    if (group) {
      group.push(item);
    } else {
      groups.set(label, [item]);
    }
  }
  return groups;
}

/** Map severity to terminal color key. */
function severityColor(
  severity: string | undefined,
  colors: ReturnType<typeof getTerminalStyles>["colors"],
): string {
  switch (severity?.toLowerCase()) {
    case "error":
    case "critical":
      return colors.error;
    case "warning":
    case "warn":
      return colors.warning;
    case "info":
      return colors.info;
    default:
      return colors.text;
  }
}

/** Format a base64 userData string for display (first 32 chars with ellipsis). */
function formatUserData(userData: string | undefined): string {
  if (!userData) return "";
  return userData.length > 32 ? `${userData.slice(0, 32)}...` : userData;
}

// =============================================================================
// Sub-components
// =============================================================================

interface EventLineProps {
  item: FlatItem;
  variant: "compact" | "detailed";
  terminalStyles: ReturnType<typeof getTerminalStyles>;
}

const EventLine: React.FC<EventLineProps> = ({
  item,
  variant,
  terminalStyles,
}) => {
  const { colors } = terminalStyles;
  const color = severityColor(item.severity, colors);
  const displayTitle =
    item.title || item.eventType || ENDPOINT_DETAILS.MSG_SYSTEM_EVENT_FALLBACK;

  return (
    <Box sx={terminalStyles.line}>
      <Box component="span" sx={{ color: colors.timestamp }}>
        {formatTerminalTime(item.time)}
      </Box>{" "}
      {item.sourceName && (
        <>
          <Box component="span" sx={{ color: colors.info, opacity: 0.7 }}>
            {item.sourceName}
          </Box>
          <Box component="span" sx={{ color: colors.text, opacity: 0.5 }}>
            {" | "}
          </Box>
        </>
      )}
      <Box component="span" sx={{ color }}>
        {displayTitle}
      </Box>
      {variant === "detailed" && item.description && (
        <Box sx={{ pl: 10, color: colors.text, opacity: 0.8 }}>
          {item.description}
        </Box>
      )}
    </Box>
  );
};

interface MessageLineProps {
  item: FlatItem;
  variant: "compact" | "detailed";
  terminalStyles: ReturnType<typeof getTerminalStyles>;
}

const MessageLine: React.FC<MessageLineProps> = ({
  item,
  variant,
  terminalStyles,
}) => {
  const { colors } = terminalStyles;
  const bsList = item.baseStations ?? [];
  const hasBsList = bsList.length > 0;
  const multiBs = bsList.length > 1;

  const primaryLine = (
    <Box sx={terminalStyles.line}>
      <Box component="span" sx={{ color: colors.timestamp }}>
        {formatTerminalTime(item.time)}
      </Box>{" "}
      <Box component="span" sx={{ color: colors.success }}>
        {ENDPOINT_DETAILS.MSG_UPLINK_RECEIVED}
      </Box>
      {item.duplicate && (
        <>
          {" "}
          <Box component="span" sx={{ color: colors.warning }}>
            {ENDPOINT_DETAILS.TERM_DUPLICATE}
          </Box>
        </>
      )}
      {hasBsList && !multiBs && (
        <>
          {ENDPOINT_DETAILS.TERM_HEARD_BY_BS}
          <Box component="span" sx={{ color: colors.identifier }}>
            {formatEUIWithDashes(bsList[0].bsEui)}
          </Box>
          {ENDPOINT_DETAILS.MSG_PACKET_PREFIX}
          <Box component="span" sx={{ color: colors.packet }}>
            {item.packetCnt ?? 0}
          </Box>
          {ENDPOINT_DETAILS.MSG_RSSI_PREFIX}
          <Box component="span" sx={{ color: colors.data }}>
            {bsList[0].rssi.toFixed(1)}dBm
          </Box>
          {ENDPOINT_DETAILS.MSG_SNR_PREFIX}
          <Box component="span" sx={{ color: colors.data }}>
            {bsList[0].snr.toFixed(1)}dB
          </Box>
        </>
      )}
      {hasBsList && multiBs && (
        <>
          {ENDPOINT_DETAILS.TERM_HEARD_BY_N_BS}
          <Box component="span" sx={{ color: colors.info }}>
            {bsList.length}
          </Box>
          {ENDPOINT_DETAILS.TERM_BASE_STATIONS_SUFFIX}
          {ENDPOINT_DETAILS.MSG_PACKET_PREFIX}
          <Box component="span" sx={{ color: colors.packet }}>
            {item.packetCnt ?? 0}
          </Box>
        </>
      )}
      {!hasBsList && item.bsEui && (
        <>
          {" "}
          {ENDPOINT_DETAILS.MSG_FROM_BS}{" "}
          <Box component="span" sx={{ color: colors.identifier }}>
            {formatEUIWithDashes(item.bsEui)}
          </Box>
          {ENDPOINT_DETAILS.MSG_PACKET_PREFIX}
          <Box component="span" sx={{ color: colors.packet }}>
            {item.packetCnt ?? 0}
          </Box>
          {item.rssi != null && (
            <>
              {ENDPOINT_DETAILS.MSG_RSSI_PREFIX}
              <Box component="span" sx={{ color: colors.data }}>
                {item.rssi.toFixed(1)}dBm
              </Box>
            </>
          )}
          {item.snr != null && (
            <>
              {ENDPOINT_DETAILS.MSG_SNR_PREFIX}
              <Box component="span" sx={{ color: colors.data }}>
                {item.snr.toFixed(1)}dB
              </Box>
            </>
          )}
        </>
      )}
      {item.userData && (
        <>
          {ENDPOINT_DETAILS.MSG_DATA_PREFIX}
          <Box component="span" sx={{ color: colors.data }}>
            {formatUserData(item.userData)}
          </Box>
        </>
      )}
    </Box>
  );

  if (variant === "compact") {
    return primaryLine;
  }

  return (
    <>
      {primaryLine}
      {multiBs &&
        bsList.map((bs, idx) => (
          <Box
            key={`${item.id}-bs-${idx}`}
            sx={{ ...terminalStyles.line, pl: 10 }}
          >
            <Box component="span" sx={{ color: colors.identifier }}>
              {formatEUIWithDashes(bs.bsEui)}
            </Box>
            {ENDPOINT_DETAILS.MSG_RSSI_PREFIX}
            <Box component="span" sx={{ color: colors.data }}>
              {bs.rssi.toFixed(1)}dBm
            </Box>
            {ENDPOINT_DETAILS.MSG_SNR_PREFIX}
            <Box component="span" sx={{ color: colors.data }}>
              {bs.snr.toFixed(1)}dB
            </Box>
          </Box>
        ))}
      {item.decodedPayload && item.decodeStatus === "success" && (
        <Box sx={{ ...terminalStyles.line, pl: 10, color: colors.info }}>
          {ENDPOINT_DETAILS.MSG_DECODED_PREFIX}{" "}
          {JSON.stringify(item.decodedPayload)}
        </Box>
      )}
    </>
  );
};

// =============================================================================
// Main Component
// =============================================================================

const ActivityTimelineInner: React.FC<ActivityTimelineProps> = ({
  items,
  variant,
  loading = false,
  emptyMessage = ENDPOINT_DETAILS.MSG_NO_MESSAGES,
}) => {
  const theme = useTheme();
  const mode = theme.palette.mode;
  const terminalStyles = useMemo(() => getTerminalStyles(mode), [mode]);

  const flatItems = useMemo(() => items.map(toFlatItem), [items]);
  const dateGroups = useMemo(() => groupByDate(flatItems), [flatItems]);

  if (loading) {
    return (
      <Paper
        variant="outlined"
        sx={{ ...terminalStyles.container, minHeight: 120 }}
      >
        <Box
          sx={{
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            py: 4,
          }}
        >
          <CircularProgress size={24} />
        </Box>
      </Paper>
    );
  }

  if (items.length === 0) {
    return (
      <Paper
        variant="outlined"
        sx={{ ...terminalStyles.container, minHeight: 120 }}
      >
        <Box sx={{ ...terminalStyles.line, textAlign: "center", py: 4 }}>
          <Typography
            sx={{
              fontFamily: "inherit",
              fontSize: "inherit",
              color: terminalStyles.colors.text,
              opacity: 0.6,
            }}
          >
            {emptyMessage}
          </Typography>
        </Box>
      </Paper>
    );
  }

  return (
    <Paper
      variant="outlined"
      sx={{
        ...terminalStyles.container,
        maxHeight: 600,
        overflow: "auto",
      }}
    >
      {Array.from(dateGroups.entries()).map(([dateLabel, groupItems]) => (
        <React.Fragment key={dateLabel}>
          <Box sx={terminalStyles.header}>=== {dateLabel} ===</Box>
          {groupItems.map((item) =>
            item.type === "event" ? (
              <EventLine
                key={item.id}
                item={item}
                variant={variant}
                terminalStyles={terminalStyles}
              />
            ) : (
              <MessageLine
                key={item.id}
                item={item}
                variant={variant}
                terminalStyles={terminalStyles}
              />
            ),
          )}
        </React.Fragment>
      ))}
    </Paper>
  );
};

export const ActivityTimeline = React.memo(ActivityTimelineInner);
