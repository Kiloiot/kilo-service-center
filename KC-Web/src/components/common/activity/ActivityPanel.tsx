/**
 * A device's Activity (NAV_STRUCTURE §2): its events and uplinks in one
 * server-paged table in the Events Log columns, narrowed by a date range.
 * The page hands in the device's feed hook and what an uplink's Scope cell
 * names; each row expands to the event's data or the uplink's receptions.
 */

import { type ReactNode, useMemo, useState } from "react";

import type { ActivityFilter, ActivityPage } from "@api-types/api";
import { Box } from "@mui/material";

import { DataTable } from "@components/common/DataTable";
import { DateRangeFilter } from "@components/common/DateRangeFilter";
import { useCursorPager } from "@hooks/useCursorPager";
import type { UplinkScope } from "@utils/uplink-summary";
import { ACTIVITY_PANEL_LAYOUT } from "@constants/app";
import { ACTIVITY_TABLE } from "@constants/messages";

import {
  activityColumns,
  activityRowDetails,
  activityRowKey,
} from "./activityRows";

/** A device's activity feed query, one cursor page at a time. */
export type ActivityFeedHook = (
  eui: string,
  filter: ActivityFilter,
  pageToken: string,
  pageSize: number,
) => { data?: ActivityPage; isLoading: boolean; error: Error | null };

interface ActivityPanelProps {
  eui: string;
  useFeed: ActivityFeedHook;
  uplinkScope: UplinkScope;
  /** Controls that act on the rows of the chosen date range, e.g. an export. */
  renderActions?: (filter: ActivityFilter) => ReactNode;
}

export function ActivityPanel({
  eui,
  useFeed,
  uplinkScope,
  renderActions,
}: ActivityPanelProps) {
  const [filter, setFilter] = useState<ActivityFilter>({});
  const pager = useCursorPager();
  const { data, isLoading, error } = useFeed(
    eui,
    filter,
    pager.pageToken,
    pager.pageSize,
  );
  const columns = useMemo(() => activityColumns(uplinkScope), [uplinkScope]);

  const changeFilter = (next: ActivityFilter) => {
    setFilter(next);
    pager.restart();
  };

  return (
    <Box>
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          flexWrap: "wrap",
          gap: ACTIVITY_PANEL_LAYOUT.TOOLBAR_GAP,
        }}
      >
        <DateRangeFilter onChange={changeFilter} />
        {renderActions?.(filter)}
      </Box>
      <DataTable
        columns={columns}
        rows={data?.items ?? []}
        rowKey={activityRowKey}
        isLoading={isLoading}
        error={error}
        emptyMessage={ACTIVITY_TABLE.EMPTY}
        paging={pager.paging(data?.nextPageToken)}
        totalCount={data?.totalCount ?? 0}
        renderDetails={activityRowDetails}
      />
    </Box>
  );
}
