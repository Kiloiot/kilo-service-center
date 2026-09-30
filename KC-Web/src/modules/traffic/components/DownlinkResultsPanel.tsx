/**
 * Downlink Results (SCACI §3.12.1 dlDataRes), tenant-wide or for one device:
 * the outcome of every downlink and, when sent, the transmitting station.
 */

import type { DeviceScope, DownlinkResultFilter } from "@api-types/api";
import { useDownlinkResults } from "@hooks";
import { Box } from "@mui/material";

import { DataTable } from "@components/common/DataTable";
import { FilterBar } from "@components/common/filters/FilterBar";
import { usePagedFilter } from "@hooks/usePagedFilter";
import { omitFixed } from "@utils/fixed-filter";
import { TIME_RANGE } from "@constants/app";
import { DOWNLINK_TABLE } from "@constants/messages";

import { DOWNLINK_RESULT_COLUMNS } from "./downlinkColumns";
import { DOWNLINK_RESULT_FIELDS } from "./trafficFields";

export function DownlinkResultsPanel({ scope }: { scope: DeviceScope }) {
  const { filter, updateFilter, paging } = usePagedFilter<DownlinkResultFilter>(
    { timeRange: TIME_RANGE.ALL },
  );
  const { data, isLoading, error } = useDownlinkResults(
    { ...filter, ...scope },
    paging.page,
    paging.pageSize,
  );

  return (
    <Box>
      <FilterBar
        fields={omitFixed(DOWNLINK_RESULT_FIELDS, scope)}
        filter={filter}
        onChange={updateFilter}
      />
      <DataTable
        columns={omitFixed(DOWNLINK_RESULT_COLUMNS, scope)}
        rows={data?.items ?? []}
        rowKey={(result) => result.queId}
        isLoading={isLoading}
        error={error}
        emptyMessage={DOWNLINK_TABLE.EMPTY_RESULTS}
        paging={paging}
        totalCount={data?.totalCount ?? 0}
      />
    </Box>
  );
}
