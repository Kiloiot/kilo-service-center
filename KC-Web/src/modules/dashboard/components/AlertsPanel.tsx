/**
 * NAV 1 alerts: the severity counts (GetAlertSummary) and the open alerts
 * (ListAlerts) in the Events Log columns, whether the dashboard updates live,
 * and a link to the full history in the Events Log.
 */

import React, { useState } from "react";
import { Link as RouterLink } from "react-router-dom";

import type { AlertSummaryUI, AlertUI } from "@api-types/alerts";
import { Box, Button, Chip, Stack } from "@mui/material";

import { DataCard } from "@components/common/DataCard";
import { DataTable } from "@components/common/DataTable";
import {
  eventColumns,
  scopeCell,
  severityCell,
  summaryCell,
  timeCell,
} from "@components/common/events/eventColumns";
import { RealtimeStatusChip } from "@components/common/RealtimeStatusChip";
import { useAlerts, useAlertSummary } from "@hooks/useAlerts";
import { usePageState } from "@hooks/usePageState";
import { eventSeverityChip } from "@utils/chipMappings";
import { ALERT_FILTER_SEVERITIES, ROUTES } from "@constants/app";
import { ALERTS_PANEL } from "@constants/messages";
import { componentSpacing } from "@theme/index";

const columns = eventColumns<AlertUI>({
  time: timeCell,
  source: scopeCell,
  severity: severityCell,
  summary: summaryCell,
});

const headerActions = (
  <Box
    sx={{
      display: "flex",
      alignItems: "center",
      gap: componentSpacing.dataCard.chipGap,
    }}
  >
    <RealtimeStatusChip />
    <Button component={RouterLink} to={ROUTES.LOGS} size="small">
      {ALERTS_PANEL.OPEN_EVENTS_LOG}
    </Button>
  </Box>
);

const chipRow = {
  display: "flex",
  flexWrap: "wrap",
  gap: componentSpacing.dataCard.chipGap,
} as const;

function SeverityCounts({ summary }: { summary?: AlertSummaryUI | null }) {
  return (
    <Box sx={chipRow} title={ALERTS_PANEL.COUNTS_RULE}>
      {ALERT_FILTER_SEVERITIES.map((severity) => {
        const chip = eventSeverityChip(severity);
        return (
          <Chip
            key={severity}
            label={`${chip.label}: ${summary?.[severity] ?? 0}`}
            color={chip.color}
            variant="outlined"
          />
        );
      })}
    </Box>
  );
}

function SeverityFilter({
  selected,
  onSelect,
}: {
  selected?: string;
  onSelect: (severity?: string) => void;
}) {
  return (
    <Box sx={chipRow}>
      <Chip
        label={ALERTS_PANEL.FILTER_ALL}
        color={selected === undefined ? "primary" : "default"}
        onClick={() => onSelect()}
      />
      {ALERT_FILTER_SEVERITIES.map((value) => (
        <Chip
          key={value}
          label={eventSeverityChip(value).label}
          color={selected === value ? "primary" : "default"}
          onClick={() => onSelect(selected === value ? undefined : value)}
        />
      ))}
    </Box>
  );
}

export const AlertsPanel: React.FC = () => {
  const [severity, setSeverity] = useState<string | undefined>(undefined);
  const pages = usePageState();
  const summary = useAlertSummary();
  const alerts = useAlerts(pages.request, { severity });

  const selectSeverity = (value?: string) => {
    setSeverity(value);
    pages.resetPage();
  };

  return (
    <DataCard
      title={ALERTS_PANEL.TITLE}
      action={headerActions}
      isLoading={alerts.isLoading}
      error={alerts.error ?? summary.error}
      errorFallback={ALERTS_PANEL.ERR_LOAD}
    >
      <Stack spacing={componentSpacing.dataCard.headerGap}>
        <SeverityCounts summary={summary.data} />
        <SeverityFilter selected={severity} onSelect={selectSeverity} />
        <DataTable
          columns={columns}
          rows={alerts.data?.items ?? []}
          rowKey={(a) => a.id}
          emptyMessage={ALERTS_PANEL.EMPTY}
          paging={pages.paging}
          totalCount={alerts.data?.totalCount ?? 0}
        />
      </Stack>
    </DataCard>
  );
};
