/**
 * A server-paged system event log. The view fixes part of the filter (a
 * device scope, the audit category, failures only) and offers the remaining
 * fields; each row expands to the event's data.
 */

import type {
  DeviceScope,
  EventLogEntryUI,
  EventLogFilter,
} from "@api-types/api";
import { useEventLog } from "@hooks";
import { Box } from "@mui/material";

import { DataTable, type DataTableColumn } from "@components/common/DataTable";
import { EventData } from "@components/common/events/EventData";
import { FilterBar } from "@components/common/filters/FilterBar";
import type { FilterField } from "@components/common/filters/types";
import { usePagedFilter } from "@hooks/usePagedFilter";
import { omitFixed } from "@utils/fixed-filter";
import { TIME_RANGE } from "@constants/app";
import { EVENT_LOG_TABLE } from "@constants/messages";

interface EventLogPanelProps {
  scope: DeviceScope;
  /** Filter values the view fixes; their fields are not offered. */
  fixed?: Partial<EventLogFilter>;
  fields: readonly FilterField<EventLogFilter>[];
  columns: readonly DataTableColumn<EventLogEntryUI>[];
}

export function EventLogPanel({
  scope,
  fixed,
  fields,
  columns,
}: EventLogPanelProps) {
  const { filter, updateFilter, paging } = usePagedFilter<EventLogFilter>({
    timeRange: TIME_RANGE.ALL,
  });
  const pinned = { ...fixed, ...scope };
  const { data, isLoading, error } = useEventLog(
    { ...filter, ...pinned },
    paging.page,
    paging.pageSize,
  );

  return (
    <Box>
      <FilterBar
        fields={omitFixed(fields, pinned)}
        filter={filter}
        onChange={updateFilter}
      />
      <DataTable
        columns={columns}
        rows={data?.items ?? []}
        rowKey={(entry) => entry.id}
        isLoading={isLoading}
        error={error}
        emptyMessage={EVENT_LOG_TABLE.EMPTY}
        paging={paging}
        totalCount={data?.totalCount ?? 0}
        renderDetails={(entry) => <EventData entry={entry} />}
      />
    </Box>
  );
}
